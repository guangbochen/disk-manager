package blockdevice

import (
	"fmt"

	diskv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/util"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNewBlockDevices(disk *block.Disk, nodeName, namespace string) []*diskv1.BlockDevice {
	bdList := make([]*diskv1.BlockDevice, 0)
	partitioned := len(disk.Partitions) > 0
	fileSystemInfo := diskv1.FilesystemStatus{
		MountPoint: disk.FileSystemInfo.MountPoint,
		Type:       disk.FileSystemInfo.FsType,
		IsReadOnly: disk.FileSystemInfo.IsReadOnly,
	}
	parent := &diskv1.BlockDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetBlockDeviceName(disk.Name, nodeName),
			Namespace: namespace,
			Labels: map[string]string{
				v1.LabelHostname: nodeName,
			},
		},
		Spec: diskv1.BlockDeviceSpec{
			NodeName: nodeName,
			DevPath:  GetDiskPath(disk.Name),
			FileSystem: diskv1.FilesystemInfo{
				MountPoint: disk.FileSystemInfo.MountPoint,
				Type:       disk.FileSystemInfo.FsType,
			},
		},
		Status: diskv1.BlockDeviceStatus{
			State: diskv1.BlockDeviceActive,
			DeviceStatus: diskv1.DeviceStatus{
				Partitioned: partitioned,
				Capacity: diskv1.DeviceCapcity{
					SizeBytes:              disk.SizeBytes,
					PhysicalBlockSizeBytes: disk.PhysicalBlockSizeBytes,
				},
				Details: diskv1.DeviceDetails{
					DeviceType:        diskv1.DeviceTypeDisk,
					DriveType:         disk.DriveType.String(),
					IsRemovable:       disk.IsRemovable,
					StorageController: disk.StorageController.String(),
					UUID:              disk.UUID,
					PtUUID:            disk.PtUUID,
					BusPath:           disk.BusPath,
					Model:             disk.Model,
					Vendor:            disk.Vendor,
					SerialNumber:      disk.SerialNumber,
					NUMANodeID:        disk.NUMANodeID,
					WWN:               disk.WWN,
				},
				FileSystem: fileSystemInfo,
			},
		},
	}
	bdList = append(bdList, parent)
	bdList = append(bdList, GetPartitionDisks(disk.Partitions, parent.DeepCopy(), nodeName)...)
	return bdList
}

func GetPartitionDisks(partitions []*ghw.Partition, parentDisk *diskv1.BlockDevice, nodeName string) []*diskv1.BlockDevice {
	blockDevices := make([]*diskv1.BlockDevice, 0, len(partitions))
	for _, part := range partitions {
		fmt.Printf("partition: %s, uuid: %s\n", part.String(), part.UUID)
		fileSystemInfo := diskv1.FilesystemStatus{
			Type:       part.FileSystemInfo.FsType,
			MountPoint: part.FileSystemInfo.MountPoint,
			IsReadOnly: part.FileSystemInfo.IsReadOnly,
		}
		diskCpy := parentDisk.DeepCopy()
		diskCpy.Spec.DevPath = GetDiskPath(part.Name)
		diskCpy.Name = util.GetBlockDeviceName(part.Name, nodeName)
		diskCpy.Spec.FileSystem.Type = part.FileSystemInfo.FsType
		diskCpy.Spec.FileSystem.MountPoint = part.FileSystemInfo.MountPoint
		diskCpy.Status.DeviceStatus.Partitioned = false
		diskCpy.Status.DeviceStatus.ParentDevice = parentDisk.Spec.DevPath
		diskCpy.Status.DeviceStatus.Details.DeviceType = diskv1.DeviceTypePart
		diskCpy.Status.DeviceStatus.Capacity.SizeBytes = part.SizeBytes
		diskCpy.Status.DeviceStatus.Details.Label = part.Label
		diskCpy.Status.DeviceStatus.Details.PartUUID = part.UUID
		diskCpy.Status.DeviceStatus.FileSystem = fileSystemInfo
		blockDevices = append(blockDevices, diskCpy)
	}
	return blockDevices
}

func GetDiskPath(shortPath string) string {
	if shortPath == "" {
		return ""
	}
	return fmt.Sprintf("/dev/%s", shortPath)
}
