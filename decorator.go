package main

// Code generated by Mikaël's proxy generator. DO NOT EDIT.

import (
	"context"
)

type MyServiceDecorator struct {
	original *MyService
	advice   func(MyServiceMethodInfo, []any, func([]any) []any) []any
}

type MyServiceMethodInfo struct {
	methodName string
	typeName   string
}

func (m *MyServiceMethodInfo) MethodName() string {
	return m.methodName
}

func (d *MyServiceDecorator) MyDecoratedMethod() {
	methodInfo := MyServiceMethodInfo{
		methodName: "MyDecoratedMethod",
		typeName:   "MyService",
	}

	var args []any

	proxiedFunc := func(args []any) []any {
		d.original.MyDecoratedMethod()
		return []any{}
	}
	d.advice(methodInfo, args, proxiedFunc)
}

func (d *MyServiceDecorator) MyContextMethod(ctx context.Context) {
	methodInfo := MyServiceMethodInfo{
		methodName: "MyContextMethod",
		typeName:   "MyService",
	}

	var args []any = []any{ctx}

	proxiedFunc := func(args []any) []any {
		d.original.MyContextMethod(args[0].(context.Context))
		return []any{}
	}
	d.advice(methodInfo, args, proxiedFunc)
}

func (d *MyServiceDecorator) MyFuncReturnsError(ctx context.Context, myType myUnexportedType) (string, error) {
	methodInfo := MyServiceMethodInfo{
		methodName: "MyFuncReturnsError",
		typeName:   "MyService",
	}

	var args []any = []any{ctx, myType}

	proxiedFunc := func(args []any) []any {
		result0, result1 := d.original.MyFuncReturnsError(args[0].(context.Context), args[1].(myUnexportedType))
		return []any{result0, result1}
	}
	results := d.advice(methodInfo, args, proxiedFunc)
	return results[0].(string), results[1].(error)
}

func NewMyServiceDecorator(delegate *MyService, advice func(methodInfo MyServiceMethodInfo, args []any, proxiedFunc func(args []any) (retVal []any)) (retVal []any)) *MyServiceDecorator {
	if advice == nil {
		advice = func(info MyServiceMethodInfo, args []any, proxiedFunc func([]any) []any) []any {
			return proxiedFunc(args)
		}
	}

	return &MyServiceDecorator{
		original: delegate,
		advice:   advice,
	}
}
