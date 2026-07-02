package main

import (
	"fmt"
	"os"

	"github.com/crandallnet/openstack-go/internal/cli"
)

func main() {
	if err := cli.RenderPrettyOSColorTest(os.Stdout, &cli.Options{Format: "pretty"}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
