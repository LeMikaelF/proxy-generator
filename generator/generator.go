package generator

import (
	_ "embed"
	"fmt"
	"github.com/LeMikaelF/proxy-generator/generator/internal/flags"
	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
	"github.com/LeMikaelF/proxy-generator/generator/internal/source"
	"github.com/LeMikaelF/proxy-generator/generator/internal/tmpl"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

type Generator struct {
	workingDir         string
	pkg                string
	typeName           string
	passthroughMethods map[string]bool
	outputter          func(generatedCode string) error
}

func New() (*Generator, error) {
	parsedFlags, err := flags.Parse()
	if err != nil {
		return nil, err
	}

	workingDir, err := getWorkingDir()
	if err != nil {
		return nil, err
	}

	g := &Generator{
		workingDir:         workingDir,
		pkg:                parsedFlags.PackageName,
		typeName:           parsedFlags.TypeName,
		passthroughMethods: parsedFlags.PassthroughMethods,
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
		sourceFile := (*source.File)(fileNode)

		if fileNode.Name.Name != g.pkg {
			continue
		}

		if structDecl == nil {
			structDecl = sourceFile.FindStructDeclaration(g.typeName)
			packageName = fileNode.Name.Name
		}

		newMethods, newImports := sourceFile.FindMethods(g.typeName, g.passthroughMethods)
		methods = append(methods, newMethods...)
		for _, newImport := range newImports {
			imports[newImport] = struct{}{}
		}
	}

	if structDecl == nil {
		return fmt.Errorf("could not find struct declaration with name %s", g.typeName)
	}

	template := tmpl.New(packageName, g.typeName, methods, toSlice(imports))
	generatedCode, err := template.Render()
	if err != nil {
		return err
	}

	if err := g.outputter(string(generatedCode)); err != nil {
		return fmt.Errorf("error outputting code: %v", err)
	}

	return nil
}

func toSlice(m map[string]struct{}) (slice []string) {
	for k := range m {
		slice = append(slice, k)
	}
	return slice
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
