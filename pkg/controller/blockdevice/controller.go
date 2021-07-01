package blockdevice

import (
	"context"
	"reflect"

	"github.com/jaypipes/ghw"
	diskv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/blockdevice"
	ctldiskv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/option"
	"github.com/longhorn/node-disk-manager/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	blockDeviceHandlerName = "longhorn-block-device-handler"
)

type Controller struct {
	namespace string
	nodeName  string

	Blockdevices     ctldiskv1.BlockDeviceController
	BlockdeviceCache ctldiskv1.BlockDeviceCache
	BlockInfo        *ghw.BlockInfo
}

// Register register the block device CRD controller
func Register(ctx context.Context, blockdevices ctldiskv1.BlockDeviceController, block *ghw.BlockInfo, opt *option.Option) error {
	controller := &Controller{
		namespace:        opt.Namespace,
		nodeName:         opt.NodeName,
		Blockdevices:     blockdevices,
		BlockdeviceCache: blockdevices.Cache(),
		BlockInfo:        block,
	}

	if err := controller.RegisterNodeBlockDevices(); err != nil {
		return err
	}

	blockdevices.OnChange(ctx, blockDeviceHandlerName, controller.OnBlockDeviceChange)
	blockdevices.OnRemove(ctx, blockDeviceHandlerName, controller.OnBlockDeviceDelete)
	return nil
}

// RegisterNodeBlockDevices will scan the block devices on the node, and it will either create or update the block device
func (c *Controller) RegisterNodeBlockDevices() error {
	logrus.Infof("Register block devices of node: %s", c.nodeName)
	bds := make([]*diskv1.BlockDevice, 0)

	// list all the block devices
	for _, disk := range c.BlockInfo.Disks {
		// ignore block device that is created by the Longhorn
		if util.IsLonghornBlockDevice(disk.BusPath) {
			logrus.Debugf("Skip longhorn disk, %s", disk.String())
			continue
		}

		logrus.Infof("Found a block device %s", disk.Name)
		newBlockDevices := blockdevice.GetNewBlockDevices(disk, c.nodeName, c.namespace)
		bds = append(bds, newBlockDevices...)
	}

	bdList, err := c.Blockdevices.List(c.namespace, v1.ListOptions{})
	if err != nil {
		return err
	}

	// either create or update the block device
	for _, bd := range bds {
		if err := c.SaveBlockDeviceByList(bd, bdList); err != nil {
			return err
		}
	}
	return nil
}

// OnBlockDeviceChange watch the block device CR on change and updates its status accordingly
func (c *Controller) OnBlockDeviceChange(key string, device *diskv1.BlockDevice) (*diskv1.BlockDevice, error) {
	if device == nil || device.DeletionTimestamp != nil {
		return device, nil
	}

	mounted := device.Status.DeviceStatus.FileSystem.MountPoint != ""
	deviceCpy := device.DeepCopy()
	if !reflect.DeepEqual(device.Status, deviceCpy.Status) {
		diskv1.DeviceMounted.SetStatusBool(deviceCpy, mounted)
		if _, err := c.Blockdevices.Update(deviceCpy); err != nil {
			return device, err
		}
	}

	return device, nil
}

func (c *Controller) SaveBlockDevice(blockDevice *diskv1.BlockDevice, bds []*diskv1.BlockDevice) error {
	for _, existingBD := range bds {
		if existingBD.Name == blockDevice.Name {
			if !reflect.DeepEqual(existingBD.Spec, blockDevice.Spec) {
				logrus.Infof("Update existing block device %s with device: %s", existingBD.Name, existingBD.Spec.DevPath)
				toUpdate := existingBD.DeepCopy()
				toUpdate.Spec = blockDevice.Spec
				if _, err := c.Blockdevices.Update(toUpdate); err != nil {
					return err
				}
			}
			return nil
		}
	}

	logrus.Infof("Add new block device %s with device: %s", blockDevice.Name, blockDevice.Spec.DevPath)
	if _, err := c.Blockdevices.Create(blockDevice); err != nil {
		return err
	}
	return nil
}

func (c *Controller) SaveBlockDeviceByList(blockDevice *diskv1.BlockDevice, bdList *diskv1.BlockDeviceList) error {
	for _, existingBD := range bdList.Items {
		if existingBD.Name == blockDevice.Name {
			if !reflect.DeepEqual(existingBD.Spec, blockDevice.Spec) {
				logrus.Infof("Update existing block device %s with device: %s", existingBD.Name, existingBD.Spec.DevPath)
				toUpdate := existingBD.DeepCopy()
				toUpdate.Spec = blockDevice.Spec
				if _, err := c.Blockdevices.Update(toUpdate); err != nil {
					return err
				}
			}
			return nil
		}
	}

	logrus.Infof("Add new block device %s with device: %s", blockDevice.Name, blockDevice.Spec.DevPath)
	if _, err := c.Blockdevices.Create(blockDevice); err != nil {
		return err
	}
	return nil
}

func (c *Controller) OnBlockDeviceDelete(key string, device *diskv1.BlockDevice) (*diskv1.BlockDevice, error) {

	if device == nil {
		return nil, nil
	}

	bds, err := c.BlockdeviceCache.List(c.namespace, labels.SelectorFromSet(map[string]string{
		blockdevice.ParentDeviceLabel: device.Name,
	}))

	if err != nil {
		return device, err
	}

	for _, bd := range bds {
		if err := c.Blockdevices.Delete(c.namespace, bd.Name, &metav1.DeleteOptions{}); err != nil {
			return device, err
		}
	}
	return nil, nil
}
