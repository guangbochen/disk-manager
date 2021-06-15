package blockdevice

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	diskv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/blockdevice"
	ctldiskv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/util"
)

const (
	blockDeviceHandlerName = "longhorn-block-device-handler"
)

type Controller struct {
	namespace string
	nodeName  string

	blockdevices     ctldiskv1.BlockDeviceController
	blockdeviceCache ctldiskv1.BlockDeviceCache
	blockInfo        *blockdevice.BlockInfo
}

// Register register the block device CRD controller
func Register(ctx context.Context, blockdevices ctldiskv1.BlockDeviceController, namespace string) error {
	blockInfo, err := blockdevice.InitBlockInfo()
	if err != nil {
		return err
	}

	nodeName, err := util.GetNodeName()
	if err != nil {
		return err
	}

	controller := &Controller{
		namespace:        namespace,
		nodeName:         nodeName,
		blockdevices:     blockdevices,
		blockdeviceCache: blockdevices.Cache(),
		blockInfo:        blockInfo,
	}

	if err := controller.RegisterNodeBlockDevices(); err != nil {
		return err
	}

	blockdevices.OnChange(ctx, blockDeviceHandlerName, controller.OnBlockDeviceChange)
	return nil
}

// OnBlockDeviceChange watch the block device CR on change and updates its status accordingly
func (c *Controller) OnBlockDeviceChange(key string, disk *diskv1.BlockDevice) (*diskv1.BlockDevice, error) {
	if disk == nil || disk.DeletionTimestamp != nil {
		return disk, nil
	}

	// update the block device state
	// TODO, update block device state when the block device is unplugged
	diskCopy := disk.DeepCopy()
	diskCopy.Status.State = diskv1.BlockDeviceActive

	// check if the block device is mounted
	for _, partition := range disk.Spec.Partitions {
		if partition.MountPoint != "" {
			diskv1.DeviceMounted.SetStatusBool(diskCopy, true)
		}
	}

	return c.blockdevices.Update(diskCopy)
}

// RegisterNodeBlockDevices will scan all the block devices available on the node, then it will either
// create or update the block device as a BlockDevice CR to the ETCD
func (c *Controller) RegisterNodeBlockDevices() error {
	fmt.Println("debug:", c.blockInfo.Block.JSONString(true))
	logrus.Infof("Register block devices of node: %s", c.nodeName)
	blockDeviceList := make([]diskv1.BlockDevice, 0)

	// list all the block devices
	for _, disk := range c.blockInfo.Block.Disks {
		// ignore block device that is created by the Longhorn
		if util.IsLonghornBlockDevice(disk.BusPath) {
			logrus.Debugf("Skip longhorn disk, %s", disk.String())
			continue
		}

		logrus.Printf("Found a block device %s", disk.Name)
		newBlockDevice := c.getNewBlockDevice(disk)
		blockDeviceList = append(blockDeviceList, newBlockDevice)
	}

	bdList, err := c.blockdevices.List(c.namespace, v1.ListOptions{})
	if err != nil {
		return err
	}

	// either create or update the block device
	for _, bd := range blockDeviceList {
		if err := c.saveBlockDevice(&bd, bdList); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) saveBlockDevice(blockDevice *diskv1.BlockDevice, bdList *diskv1.BlockDeviceList) error {
	for _, existingBD := range bdList.Items {
		if existingBD.Name == blockDevice.Name {
			if !reflect.DeepEqual(existingBD.Spec, blockDevice.Spec) {
				logrus.Infof("Update existing block device %s with device name %s", existingBD.Name, existingBD.Spec.Name)
				toUpdate := existingBD.DeepCopy()
				toUpdate.Spec = blockDevice.Spec
				if _, err := c.blockdevices.Update(toUpdate); err != nil {
					return err
				}
			}
			return nil
		}
	}

	logrus.Infof("Add new block device %s with device name %s", blockDevice.Name, blockDevice.Spec.Name)
	if _, err := c.blockdevices.Create(blockDevice); err != nil {
		return err
	}
	return nil
}

func (c *Controller) getNewBlockDevice(disk *block.Disk) diskv1.BlockDevice {
	bdName := util.GetBlockDeviceName(c.nodeName, disk.Name)
	return diskv1.BlockDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bdName,
			Namespace: c.namespace,
		},
		Spec: diskv1.BlockDeviceSpec{
			NodeName: c.nodeName,
			Name:     disk.Name,
			Capacity: diskv1.DeviceCapcity{
				SizeBytes:              disk.SizeBytes,
				PhysicalBlockSizeBytes: disk.PhysicalBlockSizeBytes,
			},
			Details: diskv1.DeviceDetails{
				DriveType:         disk.DriveType.String(),
				IsRemovable:       disk.IsRemovable,
				StorageController: disk.StorageController.String(),
				BusPath:           disk.BusPath,
				Model:             disk.Model,
				Vendor:            disk.Vendor,
				SerialNumber:      disk.SerialNumber,
				NUMANodeID:        disk.NUMANodeID,
				WWN:               disk.WWN,
			},
			Partitions: getDiskPartitions(disk.Partitions),
		},
	}
}

func getDiskPartitions(partitions []*ghw.Partition) []diskv1.Partition {
	diskPartitions := make([]diskv1.Partition, 0, len(partitions))
	for _, part := range partitions {
		partition := diskv1.Partition{
			Name:       part.Name,
			Label:      part.Label,
			MountPoint: part.MountPoint,
			SizeBytes:  part.SizeBytes,
			Type:       part.Type,
			UUID:       part.UUID,
			IsReadOnly: part.IsReadOnly,
		}
		diskPartitions = append(diskPartitions, partition)
	}
	return diskPartitions
}
