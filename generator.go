package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

type method struct {
	Name                         string
	Params                       string
	Results                      string
	ParamNames                   string
	ParamNamesWithTypeAssertions string
	ResultTypes                  []string
	ParamTypes                   []*ast.Field
	ResultExprs                  []*ast.Field
}

type data struct {
	PackageName         string
	StructName          string
	DecoratorName       string
	Methods             []method
	Imports             []string
	ConstructorName     string
	ConstructorArgs     string
	ConstructorArgNames string
}

func main() {
	excludeMethods, typeName := parseFlags()

	if !isExported(typeName) {
		log.Fatalf("type is unexported")
	}

	excludeMap := createExcludeMap(excludeMethods)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current working directory: %v", err)
	}

	files, err := filepath.Glob(wd + "/*.go")
	if err != nil {
		log.Fatalf("error finding go files: %v", err)
	}

	fset := token.NewFileSet()

	var structDecl *ast.GenDecl
	var methods []method
	var packageName string

	var constructorName, constructorArgs, constructorArgNames string
	imports := make(map[string]struct{})

	for _, file := range files {
		fileNode, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if err != nil {
			log.Printf("error parsing file %s: %v", file, err)
			continue
		}

		if structDecl == nil {
			structDecl = findStructDeclaration(fileNode, typeName)
			packageName = fileNode.Name.Name
		}

		newMethods, newImports := findMethods(fileNode, typeName, excludeMap)
		methods = append(methods, newMethods...)
		for _, newImport := range newImports {
			imports[newImport] = struct{}{}
		}

		constructorName = "New" + typeName
		// TODO is still necessary?
		constructorFunc := findConstructor(fileNode, constructorName)

		if constructorFunc != nil {
			constructorArgs = signature(constructorFunc.Type.Params)
			constructorArgNames = signatureParamNames(constructorFunc.Type.Params)
		}
	}

	if structDecl == nil {
		log.Fatalf("could not find struct declaration with name %s", typeName)
	}

	//TODO remove Funcs
	tmpl := template.Must(template.New("decorator").Funcs(map[string]any{
		"add": func(a, b int) int { return a + b },
	}).Parse(decoratorTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data{
		PackageName:         packageName,
		StructName:          typeName,
		DecoratorName:       typeName + "Decorator",
		Methods:             methods,
		Imports:             toSlice(imports),
		ConstructorName:     constructorName,
		ConstructorArgs:     constructorArgs,
		ConstructorArgNames: constructorArgNames,
	})
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error formatting code: %v", err)
	}

	fmt.Println(string(formatted))
}

func parseFlags() (excludeMethods string, typeName string) {
	flag.StringVar(&excludeMethods, "exclude-methods", "", "Comma-separated list of method names to exclude from decoration")
	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.Parse()

	if typeName == "" {
		log.Fatal("usage: custom-decorator --type <type> [--exclude-methods <method1,method2>]")
	}
	return excludeMethods, typeName
}

func toSlice(m map[string]struct{}) (slice []string) {
	for k := range m {
		slice = append(slice, k)
	}
	return slice
}

func createExcludeMap(excludeMethods string) map[string]bool {
	excludeMap := map[string]bool{}
	if excludeMethods != "" {
		excludeList := strings.Split(excludeMethods, ",")
		for _, m := range excludeList {
			excludeMap[m] = true
		}
	}
	return excludeMap
}

func findStructDeclaration(fileNode ast.Node, typeName string) *ast.GenDecl {
	var structDecl *ast.GenDecl
	ast.Inspect(fileNode, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == typeName {
				structDecl = genDecl
				return false
			}
		}
		return true
	})
	return structDecl
}

func signature(fields *ast.FieldList) string {
	var params []string
	for _, param := range fields.List {
		if len(param.Names) > 0 {
			paramName := param.Names[0].Name
			paramType := param.Type.(*ast.Ident).Name
			params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
		} else {
			params = append(params, param.Type.(*ast.Ident).Name)
		}
	}
	return strings.Join(params, ", ")
}

func signatureParamNames(fields *ast.FieldList) string {
	var paramNames []string
	for _, param := range fields.List {
		if len(param.Names) > 0 {
			paramNames = append(paramNames, param.Names[0].Name)
		}
	}
	return strings.Join(paramNames, ", ")
}

func findMethods(fileNode ast.Node, structName string, excludeMap map[string]bool) ([]method, []string) {
	var methods []method
	imports := make(map[string]struct{})

	ast.Inspect(fileNode, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		// TODO  || !isExported(funcDecl.Name.Name)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 || excludeMap[funcDecl.Name.Name] {
			return true
		}

		recvType := funcDecl.Recv.List[0].Type
		starExpr, isStar := recvType.(*ast.StarExpr)
		if isStar {
			recvType = starExpr.X
		}

		ident, ok := recvType.(*ast.Ident)
		if !ok || ident.Name != structName {
			return true
		}

		m := method{
			Name: funcDecl.Name.Name,
		}

		if funcDecl.Type.Params != nil {
			m.Params = fieldsString(funcDecl.Type.Params.List)
			m.ParamNames = fieldNamesCommaDelimited(funcDecl.Type.Params.List)
			m.ParamNamesWithTypeAssertions = fieldNamesWithTypeAssertions(funcDecl.Type.Params.List)
			m.ParamTypes = funcDecl.Type.Params.List
		}

		if funcDecl.Type.Results != nil {
			m.Results = returnValuesString(funcDecl.Type.Results.List)
			m.ResultTypes = typeNames(funcDecl.Type.Results.List)
			m.ResultExprs = funcDecl.Type.Results.List
		}

		methods = append(methods, m)

		// Collect required imports
		findImportsInMethods([]method{m}, imports)

		return true
	})

	importsList := make([]string, 0, len(imports))
	for k := range imports {
		importsList = append(importsList, k)
	}

	return methods, importsList
}

func findImportsInMethods(methods []method, imports map[string]struct{}) {
	for _, m := range methods {
		for _, p := range m.ParamTypes {
			addImportForType(imports, p.Type)
		}
		for _, r := range m.ResultExprs {
			addImportForType(imports, r.Type)
		}
	}
}

func addImportForType(imports map[string]struct{}, expr ast.Expr) {
	if se, ok := (expr).(*ast.SelectorExpr); ok {
		if id, ok := se.X.(*ast.Ident); ok {
			imports[id.Name] = struct{}{}
		}
	}
}

func fieldNamesWithTypeAssertions(fields []*ast.Field) string {
	var namesWithTypeAssertions []string

	for _, field := range fields {
		typeExpr := getTypeString(field.Type)
		for range field.Names {
			namesWithTypeAssertions = append(namesWithTypeAssertions, fmt.Sprintf("args[%d].(%s)", len(namesWithTypeAssertions), typeExpr))
		}
	}

	return strings.Join(namesWithTypeAssertions, ", ")
}

func findConstructor(fileNode ast.Node, constructorName string) *ast.FuncDecl {
	var constructorFunc *ast.FuncDecl

	ast.Inspect(fileNode, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != constructorName {
			return true
		}
		constructorFunc = funcDecl
		return false
	})

	return constructorFunc
}

func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", typeName(t.X), t.Sel.Name)
	case *ast.StarExpr:
		return "*" + typeName(t.X)
	case *ast.ArrayType:
		return "[]" + typeName(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeName(t.Key), typeName(t.Value))
	case *ast.InterfaceType:
		return "any"
	case *ast.ChanType:
		return "chan " + typeName(t.Value)
	default:
		return fmt.Sprintf("unknown(%T)", t)
	}
}

func getTypeString(expr ast.Expr) string {
	return typeName(expr)
}

func fieldsString(fields []*ast.Field) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s %s", strings.Join(names(field.Names), " "), getTypeString(field.Type)))
	}
	return strings.Join(parts, ", ")
}

func returnValuesString(fields []*ast.Field) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s %s", strings.Join(names(field.Names), " "), getTypeString(field.Type)))
	}

	joined := strings.Join(parts, ", ")

	if len(fields) > 1 {
		joined = "(" + joined + ")"
	}

	return joined
}

func fieldNamesCommaDelimited(fields []*ast.Field) string {
	names := fieldNames(fields)
	return strings.Join(names, ", ")
}

func fieldNames(fields []*ast.Field) []string {
	var names []string
	for _, field := range fields {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func names(idents []*ast.Ident) []string {
	parts := make([]string, 0, len(idents))

	for _, ident := range idents {
		parts = append(parts, ident.Name)
	}
	return parts
}

func typeNames(idents []*ast.Field) []string {
	typeNames := make([]string, 0, len(idents))

	for _, ident := range idents {
		typeNames = append(typeNames, typeName(ident.Type))
	}

	return typeNames
}

const decoratorTemplate = `package {{.PackageName}}

// Code generated by Mikaël's proxy generator. DO NOT EDIT.

{{if .Imports}}import ({{range .Imports}}
	"{{.}}"{{end}}
){{end}}

type {{.DecoratorName}} struct {
	original *{{.StructName}}
	advice   func({{.StructName}}MethodInfo, []any, func([]any) []any) []any
}

type {{.StructName}}MethodInfo struct {
	methodName string
}

func (m *{{.StructName}}MethodInfo) MethodName() string {
	return m.methodName
}

{{range .Methods}}
func (d *{{$.DecoratorName}}) {{.Name}}({{.Params}}) {{.Results}} {
	methodInfo := {{$.StructName}}MethodInfo{
		methodName: "{{.Name}}",
	};

	var args []any{{- if .Params}} = []any{ {{.ParamNames}} }{{end}}

	callOriginal := func(args []any) []any {
		{{- if .Results}}{{range $index, $_ := .ResultTypes}}{{if $index}},{{end}}result{{$index}}{{end}} := d.original.{{.Name}}({{.ParamNamesWithTypeAssertions}}){{end}};
		return []any{ {{- if .Results}}{{range $index, $_ := .ResultTypes}}{{if $index}},{{end}}result{{$index}}{{end}}{{end}}}
	};

	{{- if .Results}}results := d.advice(methodInfo, args, callOriginal);
	return {{- range $index, $element := .ResultTypes}}{{if gt $index 0}}, {{end}} results[{{$index}}].({{$element}}){{end}}{{else}}d.advice(methodInfo, args, callOriginal){{end}}}
{{end}}





func New{{.DecoratorName}}(delegate *{{.StructName}}, advice func(methodInfo {{.StructName}}MethodInfo, args []any, proxiedFunc func(args []any) (retVal []any)) (retVal []any)) *{{.DecoratorName}} {
	if advice == nil {
		advice = func(info {{.StructName}}MethodInfo, args []any, proxiedFunc func([]any) []any) []any {
			return proxiedFunc(args)
		}
	}

	return &{{.DecoratorName}}{
		original: delegate,
		advice:   advice,
	}
}
`
