# Go Proxy Generator

This is a proxy generator for golang. It can be used similarly to java's `Proxy::newInstance` to
augment existing structs with cross-cutting concerns such as tracing, metrics, logging, security,
caching, etc.

Below is an example of implementing a proxy over an existing type:

```go
package main

// declare an invocation handler
func main() {
	invocationHandler := func(method interface {
		TypeName() string
		Name() string
		Invoke(args []any) []any
	}, args []any) (retVals []any) {

		// here you can inspect and modify arguments before calling proxiedFunc
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

- [ ] Write to a file instead of stdout
- [ ] Add tests
- [ ] Rename `TypeName` to `Receiver`
- [ ] Test or disallow usage on interfaces

## License

This project is available under the MIT license.
