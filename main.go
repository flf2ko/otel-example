package main

import (
	"os"

	"github.com/flf2ko/otel-example/command"
)

func main() {
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
