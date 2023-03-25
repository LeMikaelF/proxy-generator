# Go Proxy Generator

This is a proxy generator for golang. It can be used similarly to java's `Proxy::newInstance` to
augment existing structs with cross-cutting concerns such as tracing, metrics, logging, security,
caching, etc.

Below is an example of implementing a proxy over an existing type:

```go
package main

func main() {

	// declare an invocation handler 
	invocationHandler := func(method interface {
		Package() string
		Receiver() string
		Name() string
		Invoke(args []any) []any
	}, args []any) (retVals []any) {

		// here you can inspect and modify arguments before calling Invoke()
		returnValues := method.Invoke(args)
		// here you can inspect and modify return values

		return returnValues
	}

	// configure the proxy
	myService := NewMyService("field1", "field2")
	proxy := NewMyServiceProxy(myService, invocationHandler)
}

```

The proxy implements all the exported methods from the proxied type. All method invocations will be
delegated to the provided invocation handler, similar to an `@Around` aspect in AspectJ, or the
invocationHandler of `Proxy::newInstance`.

Why the anonymous interface? This prevents the need to depend on types from the generated code, or
to implement adapter functions for every type of `NewXxxProxy`.

## TODO

- [ ] Add tests
- [ ] Test or disallow usage on interfaces
- [ ] Proxy all methods, exported and unexported; client can use method.Name() to decide.

## License

This project is available under the MIT license.
