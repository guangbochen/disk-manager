package blockdevice

import (
	"fmt"

	longhornv1 "github.com/longhorn/node-disk-manager/pkg/apis/longhorn.io/v1beta1"
	"github.com/longhorn/node-disk-manager/pkg/util"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ParentDeviceLabel = "blockdevice.longhorn.io/parent-device"
)

func GetNewBlockDevices(disk *block.Disk, nodeName, namespace string) []*longhornv1.BlockDevice {
	bdList := make([]*longhornv1.BlockDevice, 0)
	partitioned := len(disk.Partitions) > 0
	fileSystemInfo := longhornv1.FilesystemStatus{
		MountPoint: disk.FileSystemInfo.MountPoint,
		Type:       disk.FileSystemInfo.FsType,
		IsReadOnly: disk.FileSystemInfo.IsReadOnly,
	}
	parent := &longhornv1.BlockDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetBlockDeviceName(disk.Name, nodeName),
			Namespace: namespace,
			Labels: map[string]string{
				v1.LabelHostname: nodeName,
			},
		},
		Spec: longhornv1.BlockDeviceSpec{
			NodeName: nodeName,
			DevPath:  GetDiskPath(disk.Name),
			FileSystem: longhornv1.FilesystemInfo{
				MountPoint: disk.FileSystemInfo.MountPoint,
				Type:       disk.FileSystemInfo.FsType,
			},
		},
		Status: longhornv1.BlockDeviceStatus{
			State: longhornv1.BlockDeviceActive,
			DeviceStatus: longhornv1.DeviceStatus{
				Partitioned: partitioned,
				Capacity: longhornv1.DeviceCapcity{
					SizeBytes:              disk.SizeBytes,
					PhysicalBlockSizeBytes: disk.PhysicalBlockSizeBytes,
				},
				Details: longhornv1.DeviceDetails{
					DeviceType:        longhornv1.DeviceTypeDisk,
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
	bdList = append(bdList, GetPartitionDisks(disk.Partitions, parent, nodeName)...)
	return bdList
}

func GetPartitionDisks(partitions []*ghw.Partition, parentDisk *longhornv1.BlockDevice, nodeName string) []*longhornv1.BlockDevice {
	blockDevices := make([]*longhornv1.BlockDevice, 0, len(partitions))
	for _, part := range partitions {
		fmt.Printf("partition: %s, uuid: %s\n", part.String(), part.UUID)
		fileSystemInfo := longhornv1.FilesystemStatus{
			Type:       part.FileSystemInfo.FsType,
			MountPoint: part.FileSystemInfo.MountPoint,
			IsReadOnly: part.FileSystemInfo.IsReadOnly,
		}
		diskCpy := parentDisk.DeepCopy()
		diskCpy.Labels[ParentDeviceLabel] = parentDisk.Name
		diskCpy.Spec.DevPath = GetDiskPath(part.Name)
		diskCpy.Name = util.GetBlockDeviceName(part.Name, nodeName)
		diskCpy.Spec.FileSystem.Type = part.FileSystemInfo.FsType
		diskCpy.Spec.FileSystem.MountPoint = part.FileSystemInfo.MountPoint
		diskCpy.Status.DeviceStatus.Partitioned = false
		diskCpy.Status.DeviceStatus.ParentDevice = parentDisk.Spec.DevPath
		diskCpy.Status.DeviceStatus.Details.DeviceType = longhornv1.DeviceTypePart
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
