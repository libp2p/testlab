package main

import (
	"os"

	"github.com/libp2p/testlab"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var testLab *testlab.TestLab

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0-alpha"
	app.Usage = "A cluster-friendly testlab for deploying and measuring the perfomance of p2p applications"
	app.Commands = []cli.Command{
		Stop,
		Start,
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "The directory where testlab stores its state",
			EnvVar: "TESTLAB_ROOT",
			Value:  "/tmp/testlab",
		},
	}
	app.Before = func(c *cli.Context) error {
		path := c.String("root")

		var err error
		testLab, err = testlab.NewTestlab(path)
		return err
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Error(err)
	}
}
