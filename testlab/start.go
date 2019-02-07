package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/libp2p/testlab"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func readTopology(path string) (*testlab.Topology, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading topology configuration: %s", err)
	}

	var topology testlab.Topology
	err = json.NewDecoder(file).Decode(&topology)
	if err != nil {
		return nil, err
	}

	return &topology, nil
}

func start(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("expected 1 argument, got %d", c.NArg())
	}
	logrus.Info(c.Args())
	topology, err := readTopology(c.Args().Get(0))
	if err != nil {
		return err
	}

	return testLab.Start(topology)
}

var Start = cli.Command{
	Name:        "start",
	Description: "Start a cluster with a given configuration",
	Action:      start,
	ArgsUsage:   "[testlab configuration]",
}
