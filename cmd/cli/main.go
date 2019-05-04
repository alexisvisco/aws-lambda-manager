package main

import (
	"aws-test/pkg/commands"
	"os"
)

func main() {
	if err := commands.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
