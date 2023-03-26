package tmpl

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/LeMikaelF/proxy-generator/generator/internal/method"
	"go/format"
	"text/template"
)

type Template struct {
	packageName string
	structName  string
	methods     []method.Method
	imports     []string
}

func New(packageName string, structName string, methods []method.Method, imports []string) *Template {
	return &Template{packageName: packageName, structName: structName, methods: methods, imports: imports}
}

//go:embed proxy.tmpl
var proxyTemplate string

type data struct {
	PackageName string
	StructName  string
	ProxyName   string
	Methods     []method.Method
	Imports     []string
}

func (t *Template) Render() ([]byte, error) {
	var buf bytes.Buffer
	err := template.Must(template.New("proxy").Parse(proxyTemplate)).
		Execute(&buf, data{
			PackageName: t.packageName,
			StructName:  t.structName,
			ProxyName:   t.structName + "Proxy",
			Methods:     t.methods,
			Imports:     t.imports,
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
