package blockdevice

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	diskv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/block"
	ctldiskv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/option"
	"github.com/longhorn/node-disk-manager/pkg/util"
)

const (
	blockDeviceHandlerName = "longhorn-block-device-handler"
)

type Controller struct {
	namespace string
	nodeName  string

	Blockdevices     ctldiskv1.BlockDeviceController
	BlockdeviceCache ctldiskv1.BlockDeviceCache
	BlockInfo        *block.Info
}

// Register register the block device CRD controller
func Register(ctx context.Context, blockdevices ctldiskv1.BlockDeviceController, block *block.Info, opt *option.Option) error {
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
			logrus.Debugf("Skip longhorn disk, %s", disk.Name)
			continue
		}

		logrus.Infof("Found a block device %s", disk.Name)
		blockDevices := GetNewBlockDevices(disk, c.nodeName, c.namespace)
		bds = append(bds, blockDevices...)
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

// OnBlockDeviceChange watch the block device CR on change and performing disk operations
// like mounting the disks to a desired folder via ext4
func (c *Controller) OnBlockDeviceChange(key string, device *diskv1.BlockDevice) (*diskv1.BlockDevice, error) {
	if device == nil || device.DeletionTimestamp != nil {
		return device, nil
	}

	fs := device.Spec.FileSystem
	fsStatus := device.Status.DeviceStatus.FileSystem
	deviceCpy := device.DeepCopy()

	// check whether need to performing disk operation
	if _, valid := isValidMountPath(fs, fsStatus); !valid {
		if fs.ForceFormatted && fsStatus.LastFormattedAt == nil {
			logrus.Infof("performing disk operation disk: %s, fs type:%s, mount point: %s",
				device.Spec.DevPath, fs.Type, fs.MountPoint)
			//disk, err := diskfs.Open(device.Spec.DevPath)
			//if err != nil {
			//	return device, err
			//}

			//filesystem := diskfstype.FilesystemSpec{
			//	Partition:   0,
			//	FSType:      filesystem.TypeFat32,
			//	VolumeLabel: fmt.Sprintf("%s-%s", device.Namespace, device.Name),
			//}
			//if _, err := disk.CreateFilesystem(filesystem); err != nil {
			//	return device, err
			//}

			//fs, err := disk.GetFilesystem(0) // assuming it is the whole disk, so partition = 0
			//if err != nil {
			//	return device, err
			//}
			deviceCpy.Status.DeviceStatus.FileSystem = diskv1.FilesystemStatus{
				Type:            fs.Type,
				MountPoint:      fs.MountPoint,
				LastFormattedAt: &metav1.Time{Time: time.Now()},
			}
		}
	}

	message, validPath := isValidMountPath(deviceCpy.Spec.FileSystem, deviceCpy.Status.DeviceStatus.FileSystem)
	mounted := validPath && deviceCpy.Status.DeviceStatus.FileSystem.MountPoint != ""
	diskv1.DeviceMounted.SetStatusBool(deviceCpy, mounted)
	diskv1.DeviceMounted.Message(deviceCpy, message)
	if !reflect.DeepEqual(device, deviceCpy) {
		if _, err := c.Blockdevices.Update(deviceCpy); err != nil {
			return device, err
		}
	}

	return nil, nil
}

func isValidMountPath(fs diskv1.FilesystemInfo, fsStatus diskv1.FilesystemStatus) (string, bool) {
	if fs.MountPoint != fsStatus.MountPoint {
		return fmt.Sprintf("current mountPoint %s does not match the specified path: %s", fs.MountPoint, fsStatus.MountPoint), false
	}

	if fs.Type != fsStatus.Type {
		return fmt.Sprintf("current filesystem type %s does not match the specified type: %s", fs.MountPoint, fsStatus.MountPoint), false
	}

	return "", true
}

func (c *Controller) SaveBlockDevice(blockDevice *diskv1.BlockDevice, bds []*diskv1.BlockDevice) error {
	for _, existingBD := range bds {
		if existingBD.Name == blockDevice.Name {
			if !reflect.DeepEqual(existingBD, blockDevice) {
				logrus.Infof("Update existing block device %s with devPath: %s", existingBD.Name, existingBD.Spec.DevPath)
				toUpdate := existingBD.DeepCopy()
				toUpdate.Spec = blockDevice.Spec
				toUpdate.Status.DeviceStatus = blockDevice.Status.DeviceStatus
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
			if !reflect.DeepEqual(existingBD, blockDevice) {
				logrus.Infof("Update existing block device %s with device: %s", existingBD.Name, existingBD.Spec.DevPath)
				toUpdate := existingBD.DeepCopy()
				toUpdate.Spec = blockDevice.Spec
				toUpdate.Status.DeviceStatus = blockDevice.Status.DeviceStatus
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

// OnBlockDeviceDelete will delete the block devices that belongs to the same parent device
func (c *Controller) OnBlockDeviceDelete(key string, device *diskv1.BlockDevice) (*diskv1.BlockDevice, error) {
	if device == nil {
		return nil, nil
	}

	bds, err := c.BlockdeviceCache.List(c.namespace, labels.SelectorFromSet(map[string]string{
		ParentDeviceLabel: device.Name,
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
