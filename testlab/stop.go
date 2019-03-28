package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func stop(c *cli.Context) {
	if err := testLab.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "tearing down testlab %s", err)
	}
}

var Stop = cli.Command{
	Name:        "stop",
	Description: "Tears down a cluster by stopping all tagged jobs in nomad",
	Action:      stop,
}
