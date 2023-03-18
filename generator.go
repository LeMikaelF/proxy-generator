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
	Name        string
	Params      string
	Results     string
	ParamNames  string
	Exclude     bool
	ExcludeName string
}

func (m *method) IsExcluded() bool {
	if m.Exclude {
		return true
	}
	if m.ExcludeName != "" && m.ExcludeName == m.Name {
		return true
	}
	return false
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

	excludeMap := map[string]bool{}
	if excludeMethods != "" {
		excludeList := strings.Split(excludeMethods, ",")
		for _, m := range excludeList {
			excludeMap[m] = true
		}
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("error parsing file %s: %v", filename, err)
	}

	structDecl := findDeclaration(typeName, node)
	if structDecl == nil {
		log.Fatalf("could not find struct declaration with name %s", typeName)
	}

	typeSpec, ok := structDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		log.Fatalf("unexpected type specification %T", structDecl.Specs[0])
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		log.Fatalf("unexpected struct type %T", typeSpec.Type)
	}

	var methods []method

	for _, field := range structType.Fields.List {
		if field.Names == nil {
			continue
		}

		for _, name := range field.Names {
			if _, ok := field.Type.(*ast.FuncType); !ok {
				continue
			}

			paramNames := make([]string, len(field.Type.(*ast.FuncType).Params.List))
			for i, param := range field.Type.(*ast.FuncType).Params.List {
				paramNames[i] = param.Names[0].Name
			}

			method := method{
				Name:       name.Name,
				Params:     signature(field.Type.(*ast.FuncType).Params),
				Results:    signature(field.Type.(*ast.FuncType).Results),
				ParamNames: strings.Join(paramNames, ", "),
			}

			if excludeMap[name.Name] {
				method.Exclude = true
				method.ExcludeName = name.Name
			}

			methods = append(methods, method)
		}
	}

	tmpl := template.Must(template.New("decorator").Parse(decoratorTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data{
		PackageName:           node.Name.Name,
		StructName:            typeSpec.Name.Name,
		DecoratorName:         typeSpec.Name.Name + "Decorator",
		Methods:               methods,
		ConstructorParams:     signature(structType.Fields),
		ConstructorParamNames: signatureParamNames(structType.Fields),
		//ConstructorParams:     signature(structType.Fields.List[0].Type.(*ast.FuncType).Params),
		//ConstructorParamNames: signatureParamNames(structType.Fields.List[0].Type.(*ast.FuncType).Params),
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

func findDeclaration(name string, node *ast.File) *ast.GenDecl {
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == name && genDecl.Tok == token.TYPE {
					return genDecl
				}
			}
		}
	}
	return nil
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

const decoratorTemplate = `package {{.PackageName}}

// import (
// 	"context"
// )

type {{.DecoratorName}} struct {
	original *{{.StructName}}
	advice func(func())
}

{{range .Methods}}
func (d *{{$.DecoratorName}}) {{.Name}}({{.Params}}) {{.Results}} {
	if d.advice != nil {
		d.advice(func() {
			d.original.{{.Name}}({{.ParamNames}})
		})
	} else {
		d.original.{{.Name}}({{.ParamNames}})
	}
}
{{end}}

func New{{.DecoratorName}}(advice func(func()), {{.ConstructorParams}}) *{{.DecoratorName}} {
	return &{{.DecoratorName}}{
		original: New{{.StructName}}({{.ConstructorParamNames}}),
		advice: advice,
	}
}
`
