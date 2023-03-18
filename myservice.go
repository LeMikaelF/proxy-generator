package main

import "context"

//go:generate go run generator.go --type myService --exclude-methods MyMethod myservice.go
type myService struct {
	param1 string
}

func NewService(param1 string) *myService {
	return &myService{param1}
}

func (s *myService) MyMethod(ctx context.Context) {}

func (s *myService) MyDecoratedMethod() {}
