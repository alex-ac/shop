package main

import (
	"os"

	"github.com/alex-ac/shop/cli"
)

func main() {
	ctx, cancel := cli.CliContext()

	code := int(cli.Run(ctx, os.Args))
	cancel()

	os.Exit(code)
}
