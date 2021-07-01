package main

import (
	"os"

	diskv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
)

func main() {
	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/longhorn/node-disk-manager/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"longhorn.io": {
				Types: []interface{}{
					diskv1.BlockDevice{},
					diskv1.Node{},
				},
				GenerateTypes:   true,
				GenerateClients: false,
			},
		},
	})
}
