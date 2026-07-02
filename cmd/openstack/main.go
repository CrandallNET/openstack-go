package main

import (
	"fmt"
	"os"

	"github.com/crandallnet/openstack-go/internal/cli"
)

func main() {
	if err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if cli.ShouldPrintError(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(cli.ExitCode(err))
	}
}
