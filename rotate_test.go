package lumberjack

import (
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example of how to rotate in response to SIGHUP.
func TestRotate_RotateOnSigHup(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	cwd := t.TempDir()
	logfilepath := logFile(cwd)
	l := &Logger{
		Filename: logfilepath,
	}
	log.SetOutput(l)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	doneRotating := make(chan bool, 1)
	go func() {
		for {
			<-c
			err := l.Rotate()
			assert.Nil(t, err)
			doneRotating <- true
		}
	}()

	logger := log.Default()
	content := "this is a test for rotate on sighup"
	logger.Printf(content)

	fileContainsContent(t, logfilepath, []byte(content))

	c <- syscall.SIGHUP

	for {
		if <-doneRotating {
			break
		}
	}

	var logfiles []string
	walkErr := filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		assert.Nil(t, err)

		if entry.Type().IsDir() {
			return nil
		}

		logfiles = append(logfiles, path)
		return nil
	})

	assert.Nil(t, walkErr)
	assert.Equal(t, 2, len(logfiles)) // the main log file and the rotated log file

	rotatedLogfile := backupFile(cwd)

	assert.FileExists(t, rotatedLogfile)
	fileContainsContent(t, rotatedLogfile, []byte(content))
}
