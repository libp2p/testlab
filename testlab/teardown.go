package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func teardown(c *cli.Context) {
	if err := testLab.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "tearing down testlab %s", err)
	}
}

var Teardown = cli.Command{
	Name:        "teardown",
	Description: "Tears down a cluster by stopping all tagged jobs in nomad",
	Action:      teardown,
}
