package udev

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	option "github.com/longhorn/node-disk-manager/pkg/option"
	"github.com/longhorn/node-disk-manager/pkg/util"

	"github.com/jaypipes/ghw"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	ctldiskv1 "github.com/longhorn/node-disk-manager/pkg/generated/controllers/longhorn.io/v1beta1"
)

const (
	defaultDuration time.Duration = 1
)

type Udev struct {
	namespace        string
	nodeName         string
	blockInfo        *ghw.BlockInfo
	startOnce        sync.Once
	blockdevices     ctldiskv1.BlockDeviceController
	blockdeviceCache ctldiskv1.BlockDeviceCache
}

func NewUdev(info *ghw.BlockInfo, blockdevices ctldiskv1.BlockDeviceController, opt *option.Option) *Udev {
	return &Udev{
		startOnce:        sync.Once{},
		namespace:        opt.Namespace,
		nodeName:         opt.NodeName,
		blockInfo:        info,
		blockdevices:     blockdevices,
		blockdeviceCache: blockdevices.Cache(),
	}
}

func (u *Udev) Monitor(ctx context.Context) {
	u.startOnce.Do(func() {
		u.monitor(ctx)
	})
}

func (u *Udev) monitor(ctx context.Context) {
	logrus.Infoln("Start monitoring udev processed events")

	matcher, err := getOptionalMatcher(nil)
	if err != nil {
		logrus.Fatalf("failed to get udev config, error: %s", err.Error())
	}

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		logrus.Fatalf("unable to connect to Netlink Kobject UEvent socket, error: %s", err.Error())
	}
	defer conn.Close()

	uqueue := make(chan netlink.UEvent)
	errors := make(chan error)
	quit := conn.Monitor(uqueue, errors, matcher)

	// Handling message from udev queue
	for {
		select {
		case uevent := <-uqueue:
			u.ActionHandler(uevent)
		case err := <-errors:
			logrus.Errorf("failed to parse udev event, error: %s", err.Error())
		case <-ctx.Done():
			close(quit)
			return
		}
	}
}

func (u *Udev) ActionHandler(uevent netlink.UEvent) {
	udevDevice := InitUdevDevice(uevent.Env)
	if util.IsLonghornBlockDevice(udevDevice.GetIDPath()) {
		return
	}

	//if udevDevice.IsDisk() || udevDevice.IsPartition() {
	if udevDevice.IsDisk() {
		log.Println("Handle", pretty.Sprint(uevent))
		switch uevent.Action {
		case UDEV_ACTION_ADD:
			u.AddBlockDevice(udevDevice, defaultDuration)
		case UDEV_ACTION_REMOVE:
			u.RemoveBlockDevice(uevent, udevDevice)
		}
	}
}

func (u *Udev) AddBlockDevice(udevDevice UdevDevice, duration time.Duration) error {
	if duration > defaultDuration {
		time.Sleep(duration)
	}

	bdList, err := u.blockdeviceCache.List(u.namespace, labels.Everything())
	if err != nil {
		logrus.Errorf("Failed to add block device via udev event, error: %s, retry in %s", err.Error(), duration.String())
		return u.AddBlockDevice(udevDevice, 2*duration)
	}

	for _, bd := range bdList {
		// TODO, update the block device state if it is already exist
		if bd.Name == util.GetBlockDeviceName(udevDevice.GetPath(), u.nodeName) {
			return nil
		}
	}

	// TODO, create the block device if not found

	return nil
}

// RemoveBlockDevice will set the existing block device to detached state
func (u *Udev) RemoveBlockDevice(uevent netlink.UEvent, device UdevDevice) {
}

// getOptionalMatcher Parse and load config file which contains rules for matching
func getOptionalMatcher(filePath *string) (matcher netlink.Matcher, err error) {
	if filePath == nil || *filePath == "" {
		return nil, nil
	}

	stream, err := ioutil.ReadFile(*filePath)
	if err != nil {
		return nil, err
	}

	if stream == nil {
		return nil, fmt.Errorf("empty, no rules provided in \"%s\", err: %w", *filePath, err)
	}

	var rules netlink.RuleDefinitions
	if err := json.Unmarshal(stream, &rules); err != nil {
		return nil, fmt.Errorf("wrong rule syntax, err: %w", err)
	}

	return &rules, nil
}
