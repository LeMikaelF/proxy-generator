package main

import (
	"github.com/LeMikaelF/proxy-generator/generator"
	"log"
)

func main() {
	gen, err := generator.New(nil)
	if err != nil {
		log.Fatalf("could not create generator: %v\n", err)
	}

	err = gen.Run()
	if err != nil {
		log.Fatalf("could not not generator: %v\n", err)
	}
}
