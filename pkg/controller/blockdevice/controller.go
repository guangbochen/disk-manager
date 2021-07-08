package blockdevice

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	lhutil "github.com/longhorn/longhorn-manager/util"
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

	if device.Spec.FileSystem.MountPoint == "" {
		return device, nil
	}

	deviceCpy := device.DeepCopy()
	fs := deviceCpy.Spec.FileSystem
	fsStatus := deviceCpy.Status.DeviceStatus.FileSystem

	// check whether need to performing disk operation
	if _, valid := isValidFileSystem(fs, fsStatus); !valid {
		logrus.Infof("performing disk operation of disk %s, mount path %s", device.Spec.DevPath, fs.MountPoint)
		if fs.ForceFormatted && fsStatus.LastFormattedAt == nil {
			//TODO, perform filesystem formatting to ext4
			deviceCpy.Status.DeviceStatus.FileSystem.LastFormattedAt = &metav1.Time{Time: time.Now()}
		}

		if err := mountDevice(deviceCpy.Spec.DevPath, fs.MountPoint); err != nil {
			diskv1.DeviceMounted.SetStatusBool(deviceCpy, false)
			diskv1.DeviceMounted.SetError(deviceCpy, "", fmt.Errorf("failed to mount the device %s to path %s, error:%s",
				device.Spec.DevPath, device.Spec.FileSystem.MountPoint, err.Error()))
			return c.Blockdevices.Update(deviceCpy)
		}

		disk := c.BlockInfo.GetDiskByName(deviceCpy.Spec.DevPath)
		deviceCpy.Status.DeviceStatus.FileSystem.Type = disk.FileSystemInfo.FsType
		deviceCpy.Status.DeviceStatus.FileSystem.MountPoint = disk.FileSystemInfo.MountPoint
	}

	err, validFs := isValidFileSystem(deviceCpy.Spec.FileSystem, deviceCpy.Status.DeviceStatus.FileSystem)
	mounted := validFs && deviceCpy.Status.DeviceStatus.FileSystem.MountPoint != ""
	diskv1.DeviceMounted.SetStatusBool(deviceCpy, mounted)
	diskv1.DeviceMounted.SetError(deviceCpy, "", err)
	if err != nil {
		diskv1.DeviceMounted.Message(deviceCpy, err.Error())
	}

	if !reflect.DeepEqual(device, deviceCpy) {
		if _, err := c.Blockdevices.Update(deviceCpy); err != nil {
			return device, err
		}
	}

	return nil, nil
}

func mountDevice(devPath, mountPoint string) error {
	_, err := os.Stat(mountPoint)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		if err := os.Mkdir(mountPoint, os.ModeDir); err != nil {
			return err
		}
	}

	return block.MountExt4(devPath, mountPoint, false)
}

func isValidFileSystem(fs diskv1.FilesystemInfo, fsStatus diskv1.FilesystemStatus) (error, bool) {
	if len(fs.MountPoint) > 1 {
		fs.MountPoint = strings.TrimSuffix(fs.MountPoint, "/")
	}

	if fs.MountPoint != fsStatus.MountPoint {
		return fmt.Errorf("current mountPoint %s does not match the specified path: %s", fsStatus.MountPoint, fs.MountPoint), false
	}

	if !lhutil.IsSupportedFileSystem(fsStatus.Type) {
		return fmt.Errorf("unsupported filesystem type %s", fsStatus.Type), false
	}

	return nil, true
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
