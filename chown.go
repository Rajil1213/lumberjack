//go:build !linux && !darwin
// +build !linux,!darwin

package woodcutter

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
