//go:build !linux
// +build !linux

package baseapp

import "github.com/tendermint/tendermint/libs/log"

func logIoStats(logger log.Logger) {
}
