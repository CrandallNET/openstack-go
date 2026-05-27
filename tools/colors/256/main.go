package main

import (
	"fmt"
	"os"

	"github.com/crandallnet/golang-osc/internal/cli"
)

func main() {
	if err := cli.RenderPretty256ColorTestGrid(os.Stdout, &cli.Options{Format: "pretty"}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
