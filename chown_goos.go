//go:build linux || darwin
// +build linux darwin

package lumberjack

import (
	"fmt"
	"os"
	"syscall"
)

// osChown is a var so we can mock it out during tests.
//
//nolint:gochecknoglobals // global variable for testing
var osChown = os.Chown

func chown(name string, info os.FileInfo) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	f.Close()
	sys := info.Sys()
	stat, ok := sys.(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("could not change log permissions for %s", name)
	}
	return osChown(name, int(stat.Uid), int(stat.Gid))
}
