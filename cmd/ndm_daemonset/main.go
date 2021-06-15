package main

import (
	"os"

	"github.com/jaypipes/ghw"
	"github.com/longhorn/node-disk-manager/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "ndm"
	app.Version = version.FriendlyVersion()
	app.Usage = "ndm daemonset."
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "kubeconfig",
			EnvVars: []string{"KUBECONFIG"},
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(ctx *cli.Context) error {
	block, err := ghw.Block()
	if err != nil {
		logrus.Fatalf("Error getting block storage info: %v", err)
	}

	logrus.Printf("block: %v\n", block)

	for _, disk := range block.Disks {
		logrus.Printf(" disk: %v\n", disk)
		for _, part := range disk.Partitions {
			logrus.Printf("	part: %v\n", part)
		}
	}
	return nil
}
