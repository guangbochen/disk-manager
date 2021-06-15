package util

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	LonghornBusPathSubstring = "longhorn"
)

func GetNodeName() (string, error) {
	nodeName, ok := os.LookupEnv("NODE_NAME")
	if !ok {
		return "", errors.New("error getting node name")
	}
	return nodeName, nil
}

func GetBlockDeviceName(deviceName, nodeName string) string {
	return fmt.Sprintf("%s-%s", nodeName, deviceName)
}

func IsLonghornBlockDevice(path string) bool {
	return strings.Contains(path, LonghornBusPathSubstring)
}
