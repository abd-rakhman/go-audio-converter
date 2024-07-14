package main

import (
	"log"

	"github.com/abd-rakhman/go-audio-converter/internal/app/apiserver"
)

func main() {
	config := apiserver.GetConfig()

	if err := apiserver.Start(config); err != nil {
		log.Fatal(err)
	}
}
