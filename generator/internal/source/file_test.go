package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
)

type mockInspector struct{}

func (mi *mockInspector) Inspect(node ast.Node, pre func(ast.Node) bool) {
	ast.Inspect(node, pre)
}

// TODO test struct can't be found
func TestFindStructDeclaration(t *testing.T) {
	src := `
package test
type TestStruct struct {}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "", src, 0)
	file := newTestFile(f, &mockInspector{})

	result := file.FindStructDeclaration("TestStruct")

	if result == nil {
		t.Fatal("FindStructDeclaration should return a non-nil result")
	}
}

// TODO test pointer receiver
// TODO test methods with imported parameters.
func TestFindMethods(t *testing.T) {
	src := `
package test
type TestStruct struct {}
func (t TestStruct) TestMethod() {}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "", src, 0)
	file := newTestFile(f, &mockInspector{})

	methods, imports := file.FindMethods("TestStruct", map[string]bool{})

	if len(methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(methods))
	}

	m := methods[0]

	expectedMethod := method.Method{
		Name:     "TestMethod",
		Receiver: "TestStruct",
	}

	if m.Name != expectedMethod.Name {
		t.Errorf("Expected method name '%s', got '%s'", expectedMethod.Name, m.Name)
	}

	if m.Receiver != expectedMethod.Receiver {
		t.Errorf("Expected method receiver '%s', got '%s'", expectedMethod.Receiver, m.Receiver)
	}

	if len(imports) != 0 {
		t.Errorf("Expected 0 imports, got %d", len(imports))
	}
}

func newTestFile(f *ast.File, m *mockInspector) *File {
	inspect := &inspectorAdapter{m.Inspect, f}
	file := &File{file: f, inspect: inspect.Inspect}
	return file
}

func TestCollectImports(t *testing.T) {
	src := `
package test
import (
	"fmt"
	alias "os"
)
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "", src, 0)
	file := newTestFile(f, &mockInspector{})

	importMap := file.collectImports()

	expectedImportMap := map[string]importInfo{
		"fmt": {
			Path:  "fmt",
			Alias: "fmt",
		},
		"alias": {
			Path:  "os",
			Alias: "alias",
		},
	}

	if !reflect.DeepEqual(importMap, expectedImportMap) {
		t.Errorf("Expected import map to be '%v', got '%v'", expectedImportMap, importMap)
	}
}
