package tests

import (
	"context"
	"encoding/xml"
	"errors"
	"go/build/constraint"
	alias "net/http/httptest"
)

//go:generate go run ../main.go --type MyService --exclude-methods PassthroughMethod myservice.go
type MyService struct {
	param1 string
	param2 string
}

func NewMyService(param1 string, param2 string) *MyService {
	return &MyService{param1, param2}
}

func (s *MyService) NoArgsMethod() {}

func (s *MyService) ContextMethod(ctx context.Context) {}

func (s *MyService) unexportedMethod() {}

func (s *MyService) PassthroughMethod() error {
	return nil
}

func (s *MyService) OneArgErrorMethod() error {
	return nil
}

func (s *MyService) TwoArgsErrorMethod(ctx context.Context, aStruct Struct) (string, error) {
	return "", errors.New("grosse erreur")
}

func (s *MyService) ArgsWithComplexImportPathsAndAlias(a xml.CharData, b constraint.Expr, server alias.ResponseRecorder) {
}

type Struct struct{}
