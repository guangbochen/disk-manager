package node

import (
	"context"

	"github.com/jaypipes/ghw"
	longhornv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	ctllonghornv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/option"
)

type Controller struct {
	namespace string
	nodeName  string

	BlockDevices ctllonghornv1.BlockDeviceController
	Nodes        ctllonghornv1.NodeController
	BlockInfo    *ghw.BlockInfo
}

const (
	blockDeviceNodeHandlerName = "longhorn-block-device-node-handler"
)

// Register register the block device CRD controller
func Register(ctx context.Context, nodes ctllonghornv1.NodeController, bds ctllonghornv1.BlockDeviceController,
	block *ghw.BlockInfo, opt *option.Option) error {

	controller := &Controller{
		namespace:    opt.Namespace,
		nodeName:     opt.NodeName,
		Nodes:        nodes,
		BlockDevices: bds,
		BlockInfo:    block,
	}

	nodes.OnChange(ctx, blockDeviceNodeHandlerName, controller.OnNodeChange)
	nodes.OnRemove(ctx, blockDeviceNodeHandlerName, controller.OnNodeDelete)
	return nil
}

// OnNodeChange watch the node CR on change and updates the associated block devices
func (c *Controller) OnNodeChange(key string, node *longhornv1.Node) (*longhornv1.Node, error) {
	return nil, nil
}

// OnNodeDelete watch the node CR on remove and delete node related block devices
func (c *Controller) OnNodeDelete(key string, node *longhornv1.Node) (*longhornv1.Node, error) {
	return nil, nil
}
