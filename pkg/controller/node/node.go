package node

import (
	"context"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/longhorn/node-disk-manager/pkg/block"

	longhornv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	ctllonghornv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/option"
)

type Controller struct {
	namespace string
	nodeName  string

	BlockDevices     ctllonghornv1.BlockDeviceController
	BlockDeviceCache ctllonghornv1.BlockDeviceCache
	Nodes            ctllonghornv1.NodeController
	BlockInfo        *block.Info
}

const (
	blockDeviceNodeHandlerName = "longhorn-ndm-node-handler"
)

// Register register the block device CRD controller
func Register(ctx context.Context, nodes ctllonghornv1.NodeController, bds ctllonghornv1.BlockDeviceController,
	block *block.Info, opt *option.Option) error {

	c := &Controller{
		namespace:        opt.Namespace,
		nodeName:         opt.NodeName,
		Nodes:            nodes,
		BlockDevices:     bds,
		BlockDeviceCache: bds.Cache(),
		BlockInfo:        block,
	}

	//nodes.OnChange(ctx, blockDeviceNodeHandlerName, c.OnNodeChange)
	nodes.OnRemove(ctx, blockDeviceNodeHandlerName, c.OnNodeDelete)
	return nil
}

// OnNodeChange watch the node CR on change and updates the associated block devices
//func (c *Controller) OnNodeChange(key string, node *longhornv1.Node) (*longhornv1.Node, error) {
//	if node == nil || node.DeletionTimestamp != nil {
//		return node, nil
//	}
//
//
//	return nil, nil
//}

// OnNodeDelete watch the node CR on remove and delete node related block devices
func (c *Controller) OnNodeDelete(key string, node *longhornv1.Node) (*longhornv1.Node, error) {
	if node == nil {
		return nil, nil
	}

	bds, err := c.BlockDeviceCache.List(c.namespace, labels.SelectorFromSet(map[string]string{
		v1.LabelHostname: node.Name,
	}))
	if err != nil {
		return node, err
	}

	for _, bd := range bds {
		if err := c.BlockDevices.Delete(c.namespace, bd.Name, &metav1.DeleteOptions{}); err != nil {
			return node, err
		}
	}
	return nil, nil
}
