package main

import (
	"github.com/soerenschneider/flac-mate/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
