package tests

import (
	"context"
	"errors"
)

//go:generate go run ../generator.go --type MyService --exclude-methods ExcludedMethod myservice.go
type MyService struct {
	param1 string
	param2 string
}

func NewMyService(param1 string, param2 string) *MyService {
	return &MyService{param1, param2}
}

func (s *MyService) NoArgsMethod() {}

func (s *MyService) ContextMethod(ctx context.Context) {}

func (s *MyService) unexportedMethod() {
}

func (s *MyService) ExcludedMethod() error {
	return nil
}

func (s *MyService) OneArgErrorMethod() error {
	return nil
}

func (s *MyService) TwoArgsErrorMethod(ctx context.Context, aStruct Struct) (string, error) {
	return "", errors.New("grosse erreur")
}

type Struct struct{}
