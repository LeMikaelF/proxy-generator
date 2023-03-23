package tests

// Code generated by Mikaël's proxy generator. DO NOT EDIT.

import (
	"context"
)

type MyServiceProxy struct {
	original          *MyService
	invocationHandler func(method interface {
		Package() string
		Receiver() string
		Name() string
		Invoke(args []any) []any
	}, args []any) []any
}

type _MyServiceMethod struct {
	methodName string
	receiver   string
	method     func([]any) []any
}

func (m *_MyServiceMethod) Name() string { return m.methodName }

func (m *_MyServiceMethod) Receiver() string { return m.receiver }

func (m *_MyServiceMethod) Package() string { return "tests" }

func (m *_MyServiceMethod) Invoke(args []any) []any { return m.method(args) }

func (d *MyServiceProxy) NoArgsMethod() {
	method := _MyServiceMethod{
		methodName: "NoArgsMethod",
		receiver:   "*MyService",
		method: func(args []any) []any {
			d.original.NoArgsMethod()
			return []any{}
		},
	}

	var args []any
	d.invocationHandler(&method, args)
}

func (d *MyServiceProxy) ContextMethod(ctx context.Context) {
	method := _MyServiceMethod{
		methodName: "ContextMethod",
		receiver:   "*MyService",
		method: func(args []any) []any {
			d.original.ContextMethod(args[0].(context.Context))
			return []any{}
		},
	}

	var args []any = []any{ctx}
	d.invocationHandler(&method, args)
}

func (d *MyServiceProxy) OneArgErrorMethod() error {
	method := _MyServiceMethod{
		methodName: "OneArgErrorMethod",
		receiver:   "*MyService",
		method: func(args []any) []any {
			result0 := d.original.OneArgErrorMethod()
			return []any{result0}
		},
	}

	var args []any
	results := d.invocationHandler(&method, args)
	return results[0].(error)
}

func (d *MyServiceProxy) TwoArgsErrorMethod(ctx context.Context, aStruct Struct) (string, error) {
	method := _MyServiceMethod{
		methodName: "TwoArgsErrorMethod",
		receiver:   "*MyService",
		method: func(args []any) []any {
			result0, result1 := d.original.TwoArgsErrorMethod(args[0].(context.Context), args[1].(Struct))
			return []any{result0, result1}
		},
	}

	var args []any = []any{ctx, aStruct}
	results := d.invocationHandler(&method, args)
	return results[0].(string), results[1].(error)
}

func NewMyServiceProxy(delegate *MyService, invocationHandler func(method interface {
	Package() string
	Receiver() string
	Name() string
	Invoke(args []any) []any
}, args []any) (retVals []any)) *MyServiceProxy {
	if invocationHandler == nil {
		invocationHandler = func(method interface {
			Package() string
			Receiver() string
			Name() string
			Invoke(args []any) []any
		}, args []any) []any {
			return method.Invoke(args)
		}
	}

	return &MyServiceProxy{
		original:          delegate,
		invocationHandler: invocationHandler,
	}
}
