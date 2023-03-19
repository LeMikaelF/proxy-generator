package main

import (
	"context"
	"fmt"
	"testing"
)

func Test_MyStuff(_ *testing.T) {
	decorator := NewMyServiceDecorator(NewMyService("a", "b"), func(info MyServiceMethodInfo, args []any, fn func([]any) []any) []any {
		retVals := fn(args)
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
	})
	fmt.Println("ici")
	decorator.MyDecoratedMethod()
	decorator.MyContextMethod(context.Background())
}
