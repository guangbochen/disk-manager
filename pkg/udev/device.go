package udev

import (
	"strconv"

	"github.com/longhorn/node-disk-manager/pkg/util"
)

const (
	NDMBlockDevicePrefix = "blockdevice-" // NDMBlockDevicePrefix used as device's uuid prefix
	UDEV_SUBSYSTEM       = "block"        // udev to filter this device type
	UDEV_SYSTEM          = "disk"         // used to filter devices other than disk which udev tracks (eg. CD ROM)
	UDEV_PARTITION       = "partition"    // used to filter out partitions
	BY_ID_LINK           = "by-id"        // by-path devlink contains this string
	BY_PATH_LINK         = "by-path"      // by-path devlink contains this string
	LINK_ID_INDEX        = 4              // this is used to get link index from dev link
	UDEV_FS_TYPE         = "ID_FS_TYPE"   // file system type the partition
	UDEV_FS_UUID         = "ID_FS_UUID"   // UUID of the filesystem present

	UDEV_PATH                 = "DEVPATH"              // udev attribute to get device path
	UDEV_WWN                  = "ID_WWN"               // udev attribute to get device WWN number
	UDEV_SERIAL               = "ID_SERIAL_SHORT"      // udev attribute to get device serial number
	UDEV_SERIAL_FULL          = "ID_SERIAL"            // udev attribute to get - separated vendor, model, serial
	UDEV_BUS                  = "ID_BUS"               // udev attribute to get bus name
	UDEV_MODEL                = "ID_MODEL"             // udev attribute to get device model number
	UDEV_VENDOR               = "ID_VENDOR"            // udev attribute to get device vendor details
	UDEV_ID_PATH              = "ID_PATH"              // udev attribute to get device id path
	UDEV_ID_PATH_TAG          = "ID_PATH_TAG"          // udev attribute to get device id path tag
	UDEV_TYPE                 = "ID_TYPE"              // udev attribute to get device type
	UDEV_MAJOR                = "MAJOR"                // udev attribute to get device major no
	UDEV_MINOR                = "MINOR"                // udev attribute to get device minor no
	UDEV_UUID                 = "UDEV_UUID"            // ndm attribute to get device uuid
	UDEV_SYSPATH              = "UDEV_SYSPATH"         // udev attribute to get device syspath
	UDEV_ACTION               = "UDEV_ACTION"          // udev attribute to get monitor device action
	UDEV_ACTION_ADD           = "add"                  // udev attribute constant for add action
	UDEV_ACTION_REMOVE        = "remove"               // udev attribute constant for remove action
	UDEV_DEVTYPE              = "DEVTYPE"              // udev attribute to get device device type ie - disk or part
	UDEV_SOURCE               = "udev"                 // udev source constant
	UDEV_SYSPATH_PREFIX       = "/sys/dev/block/"      // udev syspath prefix
	UDEV_DEVNAME              = "DEVNAME"              // udev attribute contain disk name given by kernel
	UDEV_DEVLINKS             = "DEVLINKS"             // udev attribute contain devlinks of a disk
	UDEV_PARTITION_TABLE_TYPE = "ID_PART_TABLE_TYPE"   // udev attribute to get partition table type(gpt/dos)
	UDEV_PARTITION_TABLE_UUID = "ID_PART_TABLE_UUID"   // udev attribute to get partition table UUID
	UDEV_PARTITION_NUMBER     = "ID_PART_ENTRY_NUMBER" // udev attribute to get partition number
	UDEV_PARTITION_UUID       = "ID_PART_ENTRY_UUID"   // udev attribute to get partition uuid
	UDEV_PARTITION_TYPE       = "ID_PART_ENTRY_TYPE"   // udev attribute to get partition type
	UDEV_DM_UUID              = "DM_UUID"              // udev attribute to get the device mapper uuid

	// UDEV_DM_NAME is udev attribute to get the name of the dm device. This is used to generate the device mapper path
	UDEV_DM_NAME = "DM_NAME"
)

type UdevEventDetails struct {
	PartTableUUID   string   `json:"ID_PART_TABLE_UUID"`
	RevisionID      string   `json:"ID_REVISION"`
	USBInterfaceNum string   `json:"ID_USB_INTERFACE_NUM"`
	Major           string   `json:"MAJOR"`
	Minor           string   `json:"MINOR"`
	DeviceName      string   `json:"DEVNAME"`
	ModelID         string   `json:"ID_MODEL"`
	USBInterfaceID  string   `json:"ID_USB_INTERFACES"`
	BusID           string   `json:"ID_BUS"`
	InstanceID      string   `json:"ID_INSTANCE"`
	SerialID        string   `json:"ID_SERIAL"`
	DevPath         []string `json:"ID_INSTANCE"`

	WWN            string
	Model          string   // Model is Model of disk.
	Serial         string   // Serial is Serial of a disk.
	Vendor         string   // Vendor is Vendor of a disk.
	Path           string   // Path is Path of a disk.
	ByIdDevLinks   []string // ByIdDevLinks contains by-id devlinks
	ByPathDevLinks []string // ByPathDevLinks contains by-path devlinks
	DiskType       string   // DeviceType can be disk, partition
	// IDType is used for uuid generation using the legacy algorithm
	IDType     string
	FileSystem string // FileSystem on the disk
	// Partitiontype on the disk/device
	PartitionType string
	// PartitionNumber is the partition number, for /dev/sdb1, partition number is 1
	PartitionNumber uint8
	// PartitionTableType is the type of the partition table (dos/gpt)
	PartitionTableType string
	// DMPath is the /dev/mapper path if this is a dm device
	DMPath string
}

type UdevDevice map[string]string

func InitUdevDevice(udev map[string]string) UdevDevice {
	return udev
}

//DiskInfoFromLibudev returns disk attribute extracted using libudev apicalls.
//func (device UdevDevice) DiskInfoFromLibudev() UdevDiskDetails {
//}

//	devLinks := device.GetDevLinks()
//	diskDetails := UdevDiskDetails{
//		WWN:                device.GetPropertyValue(UDEV_WWN),
//		Model:              device.GetPropertyValue(UDEV_MODEL),
//		Serial:             device.GetPropertyValue(UDEV_SERIAL),
//		Vendor:             device.GetPropertyValue(UDEV_VENDOR),
//		Path:               device.GetPropertyValue(UDEV_DEVNAME),
//		ByIdDevLinks:       devLinks[BY_ID_LINK],
//		ByPathDevLinks:     devLinks[BY_PATH_LINK],
//		DiskType:           device.GetDevtype(),
//		IDType:             device.GetPropertyValue(UDEV_TYPE),
//		FileSystem:         device.GetFileSystemInfo(),
//		PartitionType:      device.GetPartitionType(),
//		PartitionNumber:    device.GetPartitionNumber(),
//		PartitionTableType: device.GetPropertyValue(UDEV_PARTITION_TABLE_TYPE),
//	}
//	// get the devicemapper path from the dm name
//	dmName := device.GetPropertyValue(UDEV_DM_NAME)
//	if len(dmName) != 0 {
//		diskDetails.DMPath = "/dev/mapper/" + dmName
//	}
//	return diskDetails
//}

// GetDevLinks returns syspath of a disk using syspath we can fell details
// in diskInfo struct using udev probe
//func (device UdevDevice) GetDevLinks() map[string][]string {
//	devLinkMap := make(map[string][]string)
//	byIdLink := make([]string, 0)
//	byPathLink := make([]string, 0)
//	for _, link := range strings.Split(device.GetPropertyValue(UDEV_DEVLINKS), " ") {
//		/*
//			devlink is like - /dev/disk/by-id/scsi-0Google_PersistentDisk_demo-disk
//			parts = ["", "dev", "disk", "by-id", "scsi-0Google_PersistentDisk_demo-disk"]
//			parts[4] contains link index like model or wwn or sysPath (wwn-0x5000c5009e3a8d2b) (ata-ST500LM021-1KJ152_W6HFGR)
//		*/
//		parts := strings.Split(link, "/")
//		if util.Contains(parts, BY_ID_LINK) {
//			/*
//				A default by-id link is observed to be created for all types of disks (physical, virtual and cloud).
//				This link has the format - bus, vendor, model, serial - all appended in the same order. Keeping this
//				link as the first element of array for consistency purposes.
//			*/
//			if strings.HasPrefix(parts[LINK_ID_INDEX], device.GetPropertyValue(UDEV_BUS)) && strings.HasSuffix(parts[LINK_ID_INDEX], device.GetPropertyValue(UDEV_SERIAL_FULL)) {
//				byIdLink = append([]string{link}, byIdLink...)
//			} else {
//				byIdLink = append(byIdLink, link)
//			}
//		}
//		if util.Contains(parts, BY_PATH_LINK) {
//			byPathLink = append(byPathLink, link)
//		}
//	}
//	devLinkMap[BY_ID_LINK] = byIdLink
//	devLinkMap[BY_PATH_LINK] = byPathLink
//	return devLinkMap
//}

// IsDisk check if device is a disk
func (device UdevDevice) IsDisk() bool {
	return device[UDEV_TYPE] == UDEV_SYSTEM
}

// IsPartition check if device is a partition
func (device UdevDevice) IsPartition() bool {
	return device[UDEV_TYPE] == UDEV_PARTITION
}

// GetFileSystemInfo returns filesystem type on disk/partition if it exists.
func (device UdevDevice) GetFileSystemInfo() string {
	fileSystem := device[UDEV_FS_TYPE]
	return fileSystem
}

// GetPartitionType returns the partition type of the partition, like DOS, lvm2 etc
func (device UdevDevice) GetPartitionType() string {
	partitionType := device[UDEV_PARTITION_TYPE]
	return partitionType
}

// GetPartitionNumber returns the partition number of the device, if the device is partition
// eg: /dev/sdb2 -> 2
func (device UdevDevice) GetPartitionNumber() uint8 {
	partNo, err := strconv.Atoi(device[UDEV_PARTITION_NUMBER])
	if err != nil {
		return 0
	}
	return uint8(partNo)
}

// GetSysPath returns syspath of a disk using syspath we can fell details
// in diskInfo struct using udev probe
func (device UdevDevice) GetSysPath() string {
	major := device[UDEV_MAJOR]
	minor := device[UDEV_MINOR]
	return UDEV_SYSPATH_PREFIX + major + ":" + minor
}

// GetPath returns the path of device in /dev directory
func (device UdevDevice) GetPath() string {
	return device[UDEV_DEVNAME]
}

func (device UdevDevice) IsLonghornBlockDevice() bool {
	return util.IsLonghornBlockDevice(device[UDEV_ID_PATH])
}
