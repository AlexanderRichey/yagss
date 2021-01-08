package main

import (
	"log"

	"github.com/AlexanderRichey/yasst/internal/builder"
)

func main() {
	log.SetFlags(0)

	b, err := builder.New()
	if err != nil {
		log.Fatal(err)
	}

	err = b.Build()
	if err != nil {
		log.Fatal(err)
	}
}
