package main

import "context"

//go:generate go run generator.go --type MyService --exclude-methods MyMethod myservice.go
type MyService struct {
	param1 string
	param2 string
}

func NewService(param1 string, param2 string) *MyService {
	return &MyService{param1, param2}
}

func (s *MyService) MyMethod(ctx context.Context) {}

func (s *MyService) MyDecoratedMethod() {}
