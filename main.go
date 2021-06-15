//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go
//go:generate /bin/bash scripts/generate-manifest

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/longhorn/node-disk-manager/pkg/version"

	blockdevicev1 "github.com/longhorn/node-disk-manager/pkg/controller/blockdevice"
	ctlndmv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io"
)

type Option struct {
	KubeConfig  string
	Namespace   string
	Threadiness int
}

var opt Option

func main() {
	app := cli.NewApp()
	app.Name = "node-disk-manager"
	app.Version = version.FriendlyVersion()
	app.Usage = "node-disk-manager help to manage node disks, implementing block device partition and file system formatting."
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "kubeconfig",
			EnvVars:     []string{"KUBECONFIG"},
			Destination: &opt.KubeConfig,
		},
		&cli.StringFlag{
			Name:        "namespace",
			DefaultText: "longhorn-system",
			EnvVars:     []string{"LONGHORN_NAMESPACE"},
			Destination: &opt.Namespace,
		},
		&cli.IntFlag{
			Name:        "threadiness",
			DefaultText: "2",
			Destination: &opt.Threadiness,
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	flag.Parse()

	logrus.Info("Starting node disk manager controller")
	ctx := signals.SetupSignalHandler(context.Background())

	kubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(opt.KubeConfig).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to find kubeconfig: %v", err)
	}

	bds, err := ctlndmv1.NewFactoryFromConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("error building node-disk-manager controllers: %s", err.Error())
	}

	err = blockdevicev1.Register(ctx, bds.Longhorn().V1beta1().BlockDevice(), opt.Namespace)
	if err != nil {
		return err
	}

	if err := start.All(ctx, opt.Threadiness, bds); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
