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
	"strings"
	"text/template"
)

type method struct {
	Name       string
	Params     string
	Results    string
	ParamNames string
}

type data struct {
	PackageName           string
	StructName            string
	DecoratorName         string
	Methods               []method
	ConstructorParams     string
	ConstructorParamNames string
}

func main() {
	var excludeMethods string
	var typeName string
	flag.StringVar(&excludeMethods, "exclude-methods", "", "Comma-separated list of method names to exclude from decoration")
	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 || typeName == "" {
		log.Fatal("usage: custom-decorator --type <type> [--exclude-methods <method1,method2>] <file.go>")
	}

	filename := args[0]

	excludeMap := createExcludeMap(excludeMethods)

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("error parsing file %s: %v", filename, err)
	}

	structDecl := findStructDeclaration(fileNode, typeName)
	if structDecl == nil {
		log.Fatalf("could not find struct declaration with name %s", typeName)
	}

	typeSpec := extractTypeSpec(structDecl)
	structType := extractStructType(typeSpec)

	methods := findMethods(fileNode, typeName, excludeMap)

	tmpl := template.Must(template.New("decorator").Parse(decoratorTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data{
		PackageName:           fileNode.Name.Name,
		StructName:            typeSpec.Name.Name,
		DecoratorName:         typeSpec.Name.Name + "Decorator",
		Methods:               methods,
		ConstructorParams:     signature(structType.Fields),
		ConstructorParamNames: signatureParamNames(structType.Fields),
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

func extractTypeSpec(genDecl *ast.GenDecl) *ast.TypeSpec {
	return genDecl.Specs[0].(*ast.TypeSpec)
}

func extractStructType(typeSpec *ast.TypeSpec) *ast.StructType {
	return typeSpec.Type.(*ast.StructType)
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
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 || excludeMap[funcDecl.Name.Name] {
			return true
		}

		recvType := funcDecl.Recv.List[0].Type
		starExpr, isStar := recvType.(*ast.StarExpr)
		if isStar {
			recvType = starExpr.X
		}

		ident, ok := recvType.(*ast.Ident)
		if ok && ident.Name == structName {
			m := method{
				Name:       funcDecl.Name.Name,
				Params:     extractParams(funcDecl),
				Results:    extractResults(funcDecl),
				ParamNames: extractParamNames(funcDecl),
			}
			methods = append(methods, m)
		}

		return true
	})

	return methods
}

func extractParams(funcDecl *ast.FuncDecl) string {
	var params []string
	for _, field := range funcDecl.Type.Params.List {
		paramType := getTypeString(field.Type)
		for _, paramName := range field.Names {
			params = append(params, paramName.Name+" "+paramType)
		}
	}
	return strings.Join(params, ", ")
}

func extractResults(funcDecl *ast.FuncDecl) string {
	var results []string
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			results = append(results, getTypeString(field.Type))
		}
	}
	return strings.Join(results, ", ")
}

func extractParamNames(funcDecl *ast.FuncDecl) string {
	var paramNames []string
	for _, field := range funcDecl.Type.Params.List {
		for _, paramName := range field.Names {
			paramNames = append(paramNames, paramName.Name)
		}
	}
	return strings.Join(paramNames, ", ")
}

func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.SelectorExpr:
		return t.X.(*ast.Ident).Name + "." + t.Sel.Name
	default:
		return ""
	}
}

const decoratorTemplate = `package {{.PackageName}}

type {{.DecoratorName}} struct {
	original *{{.StructName}}
	advice func(func())
}

{{range .Methods}}
func (d *{{$.DecoratorName}}) {{.Name}}({{.Params}}) {{.Results}} {
	d.advice(func() {
		d.original.{{.Name}}({{.ParamNames}})
	})
}
{{end}}

func New{{.DecoratorName}}(advice func(func()), {{.ConstructorParams}}) *{{.DecoratorName}} {
	if advice == nil {
		advice = func(fn func()) { fn() }
	}

	return &{{.DecoratorName}}{
		original: New{{.StructName}}({{.ConstructorParamNames}}),
		advice:   advice,
	}
}
`
