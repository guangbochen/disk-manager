package blockdevice

import (
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
)

type BlockInfo struct {
	Block *block.Info
}

func InitBlockInfo() (*BlockInfo, error) {
	block, err := ghw.Block()
	if err != nil {
		return nil, fmt.Errorf("error getting block storage info: %v", err)
	}
	return &BlockInfo{
		Block: block,
	}, err
}
