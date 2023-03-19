package main

import (
	"bytes"
	_ "embed"
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
	PackageName string
	StructName  string
	ProxyName   string
	Methods     []method
	Imports     []string
}

//go:embed proxy.tmpl
var proxyTemplate string

func main() {
	typeName, excludedMethods := parseFlags()
	if !isExported(typeName) {
		log.Fatalf("type is unexported")
	}

	files, err := getFilesInDirectory()
	if err != nil {
		log.Fatalf("error getting files in working directory: %v", err)
	}

	fset := token.NewFileSet()

	var structDecl *ast.GenDecl
	var methods []method
	var packageName string
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

		newMethods, newImports := findMethods(fileNode, typeName, excludedMethods)
		methods = append(methods, newMethods...)
		for _, newImport := range newImports {
			imports[newImport] = struct{}{}
		}
	}

	if structDecl == nil {
		log.Fatalf("could not find struct declaration with name %s", typeName)
	}

	var buf bytes.Buffer
	err = template.Must(template.New("proxy").Parse(proxyTemplate)).
		Execute(&buf, data{
			PackageName: packageName,
			StructName:  typeName,
			ProxyName:   typeName + "Proxy",
			Methods:     methods,
			Imports:     toSlice(imports),
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

func getFilesInDirectory() ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %v", err)
	}

	files, err := filepath.Glob(wd + "/*.go")
	if err != nil {
		return nil, fmt.Errorf("error finding go files: %v", err)
	}

	return files, nil
}

func parseFlags() (typeName string, excludedMethods map[string]bool) {
	var excludeMethods string
	flag.StringVar(&excludeMethods, "exclude-methods", "", "Comma-separated list of method names to exclude from decoration")
	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.Parse()

	if typeName == "" {
		log.Fatal("usage: go run github.com/LeMikaelF/proxy-generator --type <type> [--exclude-methods <method1,method2>]")
	}
	return typeName, csvToMap(excludeMethods)
}

func csvToMap(excludeMethods string) map[string]bool {
	excludeMap := map[string]bool{}
	if excludeMethods != "" {
		excludeList := strings.Split(excludeMethods, ",")
		for _, m := range excludeList {
			excludeMap[m] = true
		}
	}
	return excludeMap
}

func toSlice(m map[string]struct{}) (slice []string) {
	for k := range m {
		slice = append(slice, k)
	}
	return slice
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

func findMethods(fileNode ast.Node, structName string, excludeMap map[string]bool) ([]method, []string) {
	var methods []method
	imports := make(map[string]struct{})

	ast.Inspect(fileNode, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 || excludeMap[funcDecl.Name.Name] || !isExported(funcDecl.Name.Name) {
			return true
		}

		recvType := getRecvType(funcDecl)

		ident, ok := recvType.(*ast.Ident)
		if !ok || ident.Name != structName {
			return true
		}

		m := method{}
		populateName(&m, funcDecl)
		populateParameters(&m, funcDecl)
		populateResults(&m, funcDecl)
		methods = append(methods, m)

		populateImports(imports, m)

		return true
	})

	return methods, mapToSlice(imports)
}

func mapToSlice(imports map[string]struct{}) []string {
	importsList := make([]string, 0, len(imports))
	for k := range imports {
		importsList = append(importsList, k)
	}
	return importsList
}

func populateName(m *method, funcDecl *ast.FuncDecl) {
	m.Name = funcDecl.Name.Name
}

func populateParameters(m *method, funcDecl *ast.FuncDecl) {
	if funcDecl.Type.Params != nil {
		m.Params = parameterStrings(funcDecl.Type.Params.List)
		m.ParamNames = fieldNamesCommaDelimited(funcDecl.Type.Params.List)
		m.ParamNamesWithTypeAssertions = fieldNamesWithTypeAssertions(funcDecl.Type.Params.List)
		m.ParamTypes = funcDecl.Type.Params.List
	}
}

func populateResults(m *method, funcDecl *ast.FuncDecl) {
	if funcDecl.Type.Results != nil {
		m.Results = returnParametersString(funcDecl.Type.Results.List)
		m.ResultTypes = typeNames(funcDecl.Type.Results.List)
		m.ResultExprs = funcDecl.Type.Results.List
	}
}

func getRecvType(funcDecl *ast.FuncDecl) ast.Expr {
	recvType := funcDecl.Recv.List[0].Type
	starExpr, isStar := recvType.(*ast.StarExpr)
	if isStar {
		recvType = starExpr.X
	}
	return recvType
}

func populateImports(imports map[string]struct{}, m method) {
	for _, p := range m.ParamTypes {
		addImportForType(imports, p.Type)
	}
	for _, r := range m.ResultExprs {
		addImportForType(imports, r.Type)
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
		panic(fmt.Sprintf("could not infer name for type %T", t))
	}
}

func getTypeString(expr ast.Expr) string {
	return typeName(expr)
}

func parameterStrings(fields []*ast.Field) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s %s", strings.Join(names(field.Names), " "), getTypeString(field.Type)))
	}
	return strings.Join(parts, ", ")
}

func returnParametersString(fields []*ast.Field) string {
	parts := make([]string, 0, len(fields))
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
