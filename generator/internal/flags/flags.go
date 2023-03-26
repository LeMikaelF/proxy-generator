package flags

import (
	"errors"
	"flag"
	"strings"
)

type ParsedFlags struct {
	TypeName           string
	PassthroughMethods map[string]bool
}

func Parse() (flags *ParsedFlags, err error) {
	var typeName, passthroughMethodsString string

	flag.StringVar(&typeName, "type", "", "Name of the type to decorate")
	flag.StringVar(&passthroughMethodsString, "passthrough-methods", "", "Comma-separated list of method names to pass through to the delegate, without interception by the invocationHandler.")
	flag.Parse()

	if typeName == "" {
		return nil, errors.New("usage: go run github.com/LeMikaelF/proxy-generator --type <type> [--passthrough-methods <method1,method2>]")
	}
	return &ParsedFlags{typeName, csvToMap(passthroughMethodsString)}, nil
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
