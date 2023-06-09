package main

import (
	"context"
	"fmt"
	"github.com/LeMikaelF/proxy-generator/tests"
	"testing"
)

func Test_MyStuff(_ *testing.T) {
	service := tests.NewMyService("a", "b")
	invocationHandler := func(method interface {
		Package() string
		Receiver() string
		Name() string
		Invoke(args []any) []any
	}, args []any) (retVals []any) {
		fmt.Printf("In method %s\n", method.Name())

		retVals = method.Invoke(args)
		var delegateError error

		if len(retVals) == 1 {
			if err, ok := retVals[0].(error); ok {
				delegateError = err
			}
		}

		if len(retVals) == 2 {
			if err, ok := retVals[1].(error); ok {
				delegateError = err
			}
		}

		if delegateError != nil {
			fmt.Printf("delegate error was %v\n", delegateError)
		}
		return retVals
	}

	proxy := tests.NewMyServiceProxy(service, invocationHandler)

	proxy.NoArgsMethod()
	proxy.ContextMethod(context.Background())
}
