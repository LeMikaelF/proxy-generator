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
	ResultTypes                  string
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
	var excludeMethods string
	var typeName string
	flag.StringVar(&excludeMethods, "exclude-methods", "", "Comma-separated list of method names to exclude from decoration")
	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.Parse()

	if typeName == "" {
		log.Fatal("usage: custom-decorator --type <type> [--exclude-methods <method1,method2>]")
	}

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

		methods = append(methods, findMethods(fileNode, typeName, excludeMap)...)

		constructorName = "New" + typeName
		constructorFunc := findConstructor(fileNode, constructorName)

		if constructorFunc != nil {
			constructorArgs = signature(constructorFunc.Type.Params)
			constructorArgNames = signatureParamNames(constructorFunc.Type.Params)
		}
	}

	if structDecl == nil {
		log.Fatalf("could not find struct declaration with name %s", typeName)
	}

	tmpl := template.Must(template.New("decorator").Parse(decoratorTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data{
		PackageName:         packageName,
		StructName:          typeName,
		DecoratorName:       typeName + "Decorator",
		Methods:             methods,
		Imports:             getImportsForMethods(methods),
		ConstructorName:     constructorName,
		ConstructorArgs:     constructorArgs,
		ConstructorArgNames: constructorArgNames,
	})
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	fmt.Println(string(buf.Bytes()))
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error formatting code: %v", err)
	}

	fmt.Println(string(formatted))
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

func findStructDeclaration(fileNode *ast.File, typeName string) *ast.GenDecl {
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

func findMethods(fileNode *ast.File, structName string, excludeMap map[string]bool) []method {
	var methods []method

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
			m.ParamNames = fieldNames(funcDecl.Type.Params.List)
			m.ParamNamesWithTypeAssertions = fieldNamesWithTypeAssertions(funcDecl.Type.Params.List)
		}

		if funcDecl.Type.Results != nil {
			m.Results = returnValuesString(funcDecl.Type.Results.List)
			m.ResultTypes = fieldNames(funcDecl.Type.Results.List)
		}

		methods = append(methods, m)
		return true
	})

	return methods
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

func findConstructor(fileNode *ast.File, constructorName string) *ast.FuncDecl {
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

func getImportsForMethods(methods []method) []string {
	imports := []string{"context", "fmt"}
	for _, m := range methods {
		if strings.Contains(m.Results, "error") {
			return imports
		}
	}
	return imports[:1]
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

func fieldNames(fields []*ast.Field) string {
	var names []string
	for _, field := range fields {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return strings.Join(names, ", ")
}

func names(idents []*ast.Ident) []string {
	var parts []string
	for _, ident := range idents {
		parts = append(parts, ident.Name)
	}
	return parts
}

const decoratorTemplate = `package {{.PackageName}}

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
		{{- if .Results}}result := d.original.{{.Name}}({{.ParamNamesWithTypeAssertions}});
		return []any{result}{{else}}d.original.{{.Name}}({{.ParamNamesWithTypeAssertions}}); return nil;{{end}}
	};

	{{- if .Results}}results := d.advice(methodInfo, args, callOriginal);
	return {{if gt (len .Results) 1}}[]interface{}{{else}}results[0].{{end}}({{.ResultTypes}}){{else}}d.advice(methodInfo, args, callOriginal){{end}}}
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
