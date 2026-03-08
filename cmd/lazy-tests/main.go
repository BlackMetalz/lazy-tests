package main

import (
	"os"

	"github.com/kienlt/lazy-tests/internal/cli"
)

func main() {
	app := cli.New(os.Stdout, os.Stderr)
	code := app.Run(os.Args[1:])
	os.Exit(code)
}
