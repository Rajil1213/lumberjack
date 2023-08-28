package lumberjack_test

import (
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	testifyAssert "github.com/stretchr/testify/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Example of how to rotate in response to SIGHUP.
func TestRotateOnSigHup(t *testing.T) {
	cwd := t.TempDir()
	logfilepath := filepath.Join(cwd, "test.log")
	l := &lumberjack.Logger{
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
			testifyAssert.Nil(t, err)
			doneRotating <- true
		}
	}()

	logger := log.Default()
	content := "this is a test for rotate on sighup"
	logger.Printf(content)

	fileContainsContent(t, logfilepath, content)

	c <- syscall.SIGHUP

	for {
		if <-doneRotating {
			break
		}
	}

	var logfiles []string
	walkErr := filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		testifyAssert.Nil(t, err)

		if entry.Type().IsDir() {
			return nil
		}

		logfiles = append(logfiles, path)
		return nil
	})

	testifyAssert.Nil(t, walkErr)
	testifyAssert.Equal(t, 2, len(logfiles)) // the main log file and the rotated log file

	var rotatedLogfile string
	for _, logfile := range logfiles {
		if logfile != logfilepath {
			rotatedLogfile = logfile
			break
		}
	}

	testifyAssert.Greater(t, len(rotatedLogfile), 0) // the rotated log file has some non-zero length name
	fileContainsContent(t, rotatedLogfile, content)
}

func fileContainsContent(t *testing.T, logfilepath string, expectedContent string) {
	testifyAssert.FileExists(t, logfilepath)
	bytesInFile, err := os.ReadFile(logfilepath)
	testifyAssert.Nil(t, err)
	testifyAssert.Contains(t, string(bytesInFile), expectedContent)
}
