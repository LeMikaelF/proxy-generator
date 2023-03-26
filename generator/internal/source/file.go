package source

import (
	"fmt"
	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
	"go/ast"
	"go/token"
	"path"
	"strings"
)

type inspectorAdapter struct {
	delegate func(node ast.Node, f func(ast.Node) bool)
	file     *ast.File
}

func (i *inspectorAdapter) Inspect(pre func(ast.Node) bool) {
	i.delegate(i.file, pre)
}

type File struct {
	file    *ast.File
	inspect func(pre func(ast.Node) bool)
}

func NewFile(file *ast.File) *File {
	inspect := &inspectorAdapter{ast.Inspect, file}
	return &File{file: file, inspect: inspect.Inspect}
}

func (s *File) FindStructDeclaration(typeName string) *ast.GenDecl {
	var structDecl *ast.GenDecl

	s.inspect(func(n ast.Node) bool {
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

func (s *File) toAstFile() *ast.File {
	return s.file
}

func (s *File) FindMethods(structName string, passThroughMethods map[string]bool) ([]method.Method, []string) {
	var methods []method.Method
	imports := make(map[string]struct{})
	importMap := s.collectImports()

	s.inspect(func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			return true
		}

		recvType, hasStar := getReceiver(funcDecl)

		ident, ok := recvType.(*ast.Ident)
		if !ok || ident.Name != structName {
			return true
		}

		m := method.New(passThroughMethods, funcDecl, ident, hasStar)
		methods = append(methods, m)

		populateImports(imports, m, importMap)

		return true
	})

	return methods, mapToSlice(imports)
}

type importInfo struct {
	Path  string
	Alias string
}

func (s *File) collectImports() map[string]importInfo {
	importMap := make(map[string]importInfo)

	s.inspect(func(n ast.Node) bool {
		importSpec, ok := n.(*ast.ImportSpec)
		if !ok {
			return true
		}

		importPath := strings.Trim(importSpec.Path.Value, "\"")

		alias := ""
		if importSpec.Name != nil {
			alias = importSpec.Name.Name
		} else {
			_, alias = path.Split(importPath)
		}

		importMap[alias] = importInfo{Path: importPath, Alias: alias}

		return true
	})

	return importMap
}

func getReceiver(funcDecl *ast.FuncDecl) (ast.Expr, bool) {
	recvType := funcDecl.Recv.List[0].Type
	starExpr, isStar := recvType.(*ast.StarExpr)
	if isStar {
		recvType = starExpr.X
	}
	return recvType, isStar
}

func populateImports(imports map[string]struct{}, m method.Method, importMap map[string]importInfo) {
	for _, field := range append(m.ParamTypes, m.ResultExprs...) {
		switch t := field.Type.(type) {
		case *ast.Ident:
			if info, ok := importMap[t.Name]; ok {
				imports[fmt.Sprintf("%s %q", info.Alias, info.Path)] = struct{}{}
			}
		case *ast.SelectorExpr:
			if x, ok := t.X.(*ast.Ident); ok {
				if info, ok := importMap[x.Name]; ok {
					imports[fmt.Sprintf("%s %q", info.Alias, info.Path)] = struct{}{}
				}
			}
		case *ast.StarExpr:
			if se, ok := t.X.(*ast.SelectorExpr); ok {
				if x, ok := se.X.(*ast.Ident); ok {
					if info, ok := importMap[x.Name]; ok {
						imports[fmt.Sprintf("%s %q", info.Alias, info.Path)] = struct{}{}
					}
				}
			}
		}
	}
}

func mapToSlice(imports map[string]struct{}) []string {
	importsList := make([]string, 0, len(imports))
	for k := range imports {
		importsList = append(importsList, k)
	}
	return importsList
}
