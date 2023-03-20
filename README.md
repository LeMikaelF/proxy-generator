# Go Proxy Generator
This is a proxy generator for golang. It can be used similarly to java's `Proxy::newInstance` to augment existing structs with cross-cutting concerns such as tracing, metrics, logging, security, etc.

Below is an example of implementing a proxy over an existing type:

```go
myService := NewMyService("field1", "field2")
advice := func(methodInfo MyServiceMethodInfo, args []any, proxiedFunc func(args []any) (retVals []any)) (retVals []any) {
	// here you can inspect and modify arguments before calling proxiedFunc
	returnValues := proxiedFunc(args)
	// here you can inspect and modify return values
	return returnValues
}
proxy := NewMyServiceProxy(myService, advice)
```
The proxy has the same method set as the proxied type, and can therefore be used interchangeably with it.

The type of first argument that the proxy receives has a method set that clients can use to implement an interface in their proxies. For example:

```go
type MethodInfo interface {
	MethodName() string
	TypeName() string
}
```

## Caveat
If you don't want to invoke the proxied method, you still have to return a slice containing values of the expected types.

## TODO
- [ ] Write to a file instead of stdout
- [ ] Add tests

## License
This project is available under the MIT license.
