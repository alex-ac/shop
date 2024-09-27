package main

import (
	"os"

	"github.com/alex-ac/shop/cli"
)

func main() {
	ctx, cancel := cli.CliContext()

	code := cli.ErrorToExitCode(cli.Run(ctx, os.Args))
	cancel()

	os.Exit(code)
}
