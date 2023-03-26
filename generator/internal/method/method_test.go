package method_test

import (
	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
	"go/ast"
	"testing"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name               string
		passThroughMethods map[string]bool
		funcDecl           *ast.FuncDecl
		ident              *ast.Ident
		hasStar            bool
		expected           method.Method
	}{
		{
			name:               "Simple function",
			passThroughMethods: map[string]bool{},
			funcDecl: &ast.FuncDecl{
				Name: &ast.Ident{Name: "Foo"},
				Type: &ast.FuncType{},
			},
			ident:   &ast.Ident{Name: "MyType"},
			hasStar: false,
			expected: method.Method{
				Name:                         "Foo",
				Params:                       "",
				Results:                      "",
				ParamNames:                   "",
				ParamNamesWithTypeAssertions: "",
				Receiver:                     "MyType",
				ResultTypes:                  nil,
				ParamTypes:                   nil,
				ResultExprs:                  nil,
				Passthrough:                  false,
			},
		},
		{
			name:               "Function with parameters and results",
			passThroughMethods: map[string]bool{},
			funcDecl: &ast.FuncDecl{
				Name: &ast.Ident{Name: "Bar"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{{Name: "a"}},
								Type:  &ast.Ident{Name: "int"},
							},
							{
								Names: []*ast.Ident{{Name: "b"}},
								Type:  &ast.Ident{Name: "string"},
							},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{
								Type: &ast.Ident{Name: "error"},
							},
						},
					},
				},
			},
			ident:   &ast.Ident{Name: "MyType"},
			hasStar: true,
			expected: method.Method{
				Name:                         "Bar",
				Params:                       "a int,b string",
				Results:                      "error",
				ParamNames:                   "a,b",
				ParamNamesWithTypeAssertions: "args[0].(int),args[1].(string)",
				Receiver:                     "*MyType",
				ResultTypes:                  []string{"error"},
				ParamTypes: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: "a"}},
						Type:  &ast.Ident{Name: "int"},
					},
					{
						Names: []*ast.Ident{{Name: "b"}},
						Type:  &ast.Ident{Name: "string"},
					},
				},
				ResultExprs: []*ast.Field{
					{
						Type: &ast.Ident{Name: "error"},
					},
				},
				Passthrough: false,
			},
		},
		{
			name: "Passthrough function",
			passThroughMethods: map[string]bool{
				"Baz": true,
			},
			funcDecl: &ast.FuncDecl{
				Name: &ast.Ident{Name: "Baz"},
				Type: &ast.FuncType{},
			},
			ident:   &ast.Ident{Name: "MyType"},
			hasStar: false,
			expected: method.Method{
				Name:                         "Baz",
				Params:                       "",
				Results:                      "",
				ParamNames:                   "",
				ParamNamesWithTypeAssertions: "",
				Receiver:                     "MyType",
				ResultTypes:                  nil,
				ParamTypes:                   nil,
				ResultExprs:                  nil,
				Passthrough:                  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := method.New(tc.passThroughMethods, tc.funcDecl, tc.ident, tc.hasStar)
			if !equals(got, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}

func equals(m method.Method, n method.Method) bool {
	return m.Name == n.Name &&
		m.Params == n.Params &&
		m.Results == n.Results &&
		m.ParamNames == n.ParamNames &&
		m.ParamNamesWithTypeAssertions == n.ParamNamesWithTypeAssertions &&
		m.Receiver == n.Receiver &&
		m.Passthrough == n.Passthrough
}
