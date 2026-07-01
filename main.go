package main

import (
	"os"

	"github.com/Surfe/surfer/cmd"
)

func main() {
	if cmd.Execute() != nil {
		os.Exit(1)
	}
}
