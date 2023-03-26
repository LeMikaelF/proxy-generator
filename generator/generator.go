package generator

import (
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

type FileHandler interface {
	Getwd() (string, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Glob(pattern string) ([]string, error)
}

type OsFileHandler struct{}

func (fh *OsFileHandler) Getwd() (string, error) {
	return os.Getwd()
}

func (fh *OsFileHandler) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fh *OsFileHandler) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

type Generator struct {
	workingDir         string
	pkg                string
	typeName           string
	passthroughMethods map[string]bool
	fileHandler        FileHandler
}

func New(fh FileHandler) (*Generator, error) {
	if fh == nil {
		fh = &OsFileHandler{}
	}

	workingDir, err := fh.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %v", err)
	}

	g := &Generator{
		workingDir:  workingDir,
		fileHandler: fh,
	}

	parsedFlags, err := flags.Parse()
	if err != nil {
		return nil, err
	}
	g.pkg = parsedFlags.PackageName
	g.typeName = parsedFlags.TypeName
	g.passthroughMethods = parsedFlags.PassthroughMethods

	return g, nil
}

func (g *Generator) Run() error {
	files, err := g.fileHandler.Glob(g.workingDir + "/*.go")
	if err != nil {
		return fmt.Errorf("error finding go files: %v", err)
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
		sourceFile := source.NewFile(fileNode)

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

	generatedFileName := fmt.Sprintf("%s_proxy_gen.go", g.typeName)
	if err := g.fileHandler.WriteFile(generatedFileName, []byte(generatedCode), 0666); err != nil {
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
