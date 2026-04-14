package main

import (
	"os"

	"github.com/truestamp/truestamp-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
