package generator

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type Generator struct {
	workingDir         string
	pkg                string
	typeName           string
	passthroughMethods map[string]bool
	outputter          func(generatedCode string) error
}

func New() (*Generator, error) {
	typeName, passthroughMethods, err := parseFlags()
	if err != nil {
		return nil, err
	}

	workingDir, err := getWorkingDir()
	if err != nil {
		return nil, err
	}

	g := &Generator{
		workingDir:         workingDir,
		pkg:                os.Getenv("GOPACKAGE"),
		typeName:           typeName,
		passthroughMethods: passthroughMethods,
	}
	g.outputter = g.output

	return g, nil
}

func getWorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current working directory: %v", err)
	}
	return wd, nil
}

type data struct {
	PackageName string
	StructName  string
	ProxyName   string
	Methods     []method.Method
	Imports     []string
}

//go:embed proxy.tmpl
var proxyTemplate string

func (g *Generator) Run() error {
	files, err := g.getFilesInDirectory()
	if err != nil {
		return fmt.Errorf("error getting files in working directory: %v", err)
	}

	fset := token.NewFileSet()

	var structDecl *ast.GenDecl
	var methods []method.Method
	var packageName string
	imports := make(map[string]struct{})

	for _, file := range files {
		fileNode, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if err != nil {
			return fmt.Errorf("error parsing file %s: %v", file, err)
		}

		if fileNode.Name.Name != g.pkg {
			continue
		}

		if structDecl == nil {
			structDecl = findStructDeclaration(fileNode, g.typeName)
			packageName = fileNode.Name.Name
		}

		newMethods, newImports := findMethods(fileNode, g.typeName, g.passthroughMethods)
		methods = append(methods, newMethods...)
		for _, newImport := range newImports {
			imports[newImport] = struct{}{}
		}
	}

	if structDecl == nil {
		return fmt.Errorf("could not find struct declaration with name %s", g.typeName)
	}

	generatedCode, err := g.generateCode(packageName, methods, imports)
	if err != nil {
		return err
	}

	if err := g.outputter(string(generatedCode)); err != nil {
		return fmt.Errorf("error outputting code: %v", err)
	}

	return nil
}

func parseFlags() (typeName string, passthroughMethods map[string]bool, err error) {
	//TODO rename.
	var excludeMethodsString string
	flag.StringVar(&excludeMethodsString, "exclude-methods", "", "Comma-separated list of method names to pass through to the delegate, without interception by the invocationHandler.")
	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.Parse()

	if typeName == "" {
		return "", nil, errors.New("usage: go run github.com/LeMikaelF/proxy-generator --type <type> [--passthrough-methods <method1,method2>]")
	}
	return typeName, csvToMap(excludeMethodsString), nil
}

func csvToMap(csv string) map[string]bool {
	m := map[string]bool{}
	if csv != "" {
		slice := strings.Split(csv, ",")
		for _, element := range slice {
			m[element] = true
		}
	}
	return m
}

func (g *Generator) output(generatedCode string) error {
	generatedFileName := fmt.Sprintf("%s_proxy_gen.go", g.typeName)

	return os.WriteFile(generatedFileName, []byte(generatedCode), 0666)
}

func (g *Generator) getFilesInDirectory() ([]string, error) {
	files, err := filepath.Glob(g.workingDir + "/*.go")
	if err != nil {
		return nil, fmt.Errorf("error finding go files: %v", err)
	}

	return files, nil
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

func findMethods(fileNode ast.Node, structName string, passThroughMethods map[string]bool) ([]method.Method, []string) {
	var methods []method.Method
	imports := make(map[string]struct{})
	importMap := collectImports(fileNode)

	ast.Inspect(fileNode, func(n ast.Node) bool {
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

func collectImports(fileNode ast.Node) map[string]importInfo {
	importMap := make(map[string]importInfo)

	ast.Inspect(fileNode, func(n ast.Node) bool {
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

func (g *Generator) generateCode(packageName string, methods []method.Method, imports map[string]struct{}) ([]byte, error) {
	var buf bytes.Buffer
	err := template.Must(template.New("proxy").Parse(proxyTemplate)).
		Execute(&buf, data{
			PackageName: packageName,
			StructName:  g.typeName,
			ProxyName:   g.typeName + "Proxy",
			Methods:     methods,
			Imports:     toSlice(imports),
		})
	if err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error formatting code: %v", err)
	}
	return formatted, nil
}

func toSlice(m map[string]struct{}) (slice []string) {
	for k := range m {
		slice = append(slice, k)
	}
	return slice
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
