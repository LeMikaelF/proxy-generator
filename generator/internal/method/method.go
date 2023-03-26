package method

import (
	"fmt"
	"go/ast"
	"strings"
)

type Method struct {
	Name                         string
	Params                       string
	Results                      string
	ParamNames                   string
	ParamNamesWithTypeAssertions string
	Receiver                     string
	ResultTypes                  []string
	ParamTypes                   []*ast.Field
	ResultExprs                  []*ast.Field
	Passthrough                  bool
}

func New(passThroughMethods map[string]bool, funcDecl *ast.FuncDecl, ident *ast.Ident, hasStar bool) Method {
	m := Method{}
	populatePassthrough(&m, passThroughMethods[funcDecl.Name.Name])
	populateName(&m, funcDecl)
	populateReceiver(&m, ident.Name, hasStar)
	populateParameters(&m, funcDecl)
	populateResults(&m, funcDecl)
	return m
}

func populatePassthrough(m *Method, isPassthrough bool) {
	m.Passthrough = isPassthrough
}

func populateName(m *Method, funcDecl *ast.FuncDecl) {
	m.Name = funcDecl.Name.Name
}

func populateReceiver(m *Method, receiver string, hasStar bool) {
	var starPrefix string
	if hasStar {
		starPrefix = "*"
	}
	m.Receiver = fmt.Sprintf("%s%s", starPrefix, receiver)
}

func populateParameters(m *Method, funcDecl *ast.FuncDecl) {
	if funcDecl.Type.Params != nil {
		m.Params = parameterStrings(funcDecl.Type.Params.List)
		m.ParamNames = fieldNamesCommaDelimited(funcDecl.Type.Params.List)
		m.ParamNamesWithTypeAssertions = fieldNamesWithTypeAssertions(funcDecl.Type.Params.List)
		m.ParamTypes = funcDecl.Type.Params.List
	}
}

func parameterStrings(fields []*ast.Field) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s %s", strings.Join(names(field.Names), " "), typeName(field.Type)))
	}
	return strings.Join(parts, ",")
}

func names(idents []*ast.Ident) []string {
	parts := make([]string, 0, len(idents))

	for _, ident := range idents {
		parts = append(parts, ident.Name)
	}
	return parts
}

func fieldNamesCommaDelimited(fields []*ast.Field) string {
	names := fieldNames(fields)
	return strings.Join(names, ",")
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

func fieldNamesWithTypeAssertions(fields []*ast.Field) string {
	var namesWithTypeAssertions []string

	for _, field := range fields {
		typeExpr := typeName(field.Type)
		for range field.Names {
			namesWithTypeAssertions = append(namesWithTypeAssertions, fmt.Sprintf("args[%d].(%s)", len(namesWithTypeAssertions), typeExpr))
		}
	}

	return strings.Join(namesWithTypeAssertions, ",")
}

func populateResults(m *Method, funcDecl *ast.FuncDecl) {
	if funcDecl.Type.Results != nil {
		m.Results = returnParametersString(funcDecl.Type.Results.List)
		m.ResultTypes = typeNames(funcDecl.Type.Results.List)
		m.ResultExprs = funcDecl.Type.Results.List
	}
}

func returnParametersString(fields []*ast.Field) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, typeName(field.Type))
	}

	joined := strings.Join(parts, ",")

	if len(fields) > 1 {
		joined = "(" + joined + ")"
	}

	return joined
}

func typeNames(idents []*ast.Field) []string {
	typeNames := make([]string, 0, len(idents))

	for _, ident := range idents {
		typeNames = append(typeNames, typeName(ident.Type))
	}

	return typeNames
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
