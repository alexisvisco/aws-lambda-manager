package main

import (
	"os"

	"aws-test/pkg/commands"
)

func main() {
	if err := commands.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
