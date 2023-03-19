package main

import (
	"context"
	"errors"
)

//go:generate go run generator.go --type MyService --exclude-methods MyMethod myservice.go
type MyService struct {
	param1 string
	param2 string
}

func NewMyService(param1 string, param2 string) *MyService {
	return &MyService{param1, param2}
}

func (s *MyService) MyMethod(ctx context.Context) {}

func (s *MyService) MyDecoratedMethod() {}

func (s *MyService) MyContextMethod(ctx context.Context) {}

func (s *MyService) myUnexportedMethod() error {
	return nil
}

func (s *MyService) MyFuncReturnsError(ctx context.Context, myType myType) (string, error) {
	return "", errors.New("grosse erreur")
}

type myType struct{}
