package slogrotate_test

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/natefinch/lumberjack.v2/slogrotate"

	"github.com/stretchr/testify/assert"
)

func TestCreationOfLogFile(t *testing.T) {
	cwd := t.TempDir()
	logfile := filepath.Join(cwd, "test.log")
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		LocalTime:  false,
		Compress:   false,
	}
	logger := slogrotate.NewSlogRotateLogger(lumberjackLogger)

	logger.Warn("this is a test", "test_key", "test_value")

	assert.FileExists(t, logfile)
}

func TestRotation(t *testing.T) {
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := slogrotate.NewLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := slogrotate.NewSlogRotateLogger(lumberjackLogger)

	logger.Warn("this is a test", "test_key", "test_value")

	assert.FileExists(t, logfile)

	err := lumberjackLogger.Rotate()
	assert.Nil(t, err)

	filesInLogDir, readErr := os.ReadDir(cwd)
	assert.Nil(t, readErr)

	assert.FileExists(t, logfile)
	assert.Equal(t, len(filesInLogDir), 2) // rotated file and current file

	entries := []string{}
	for _, dirEntry := range filesInLogDir {
		entries = append(entries, dirEntry.Name())
	}

	assert.Contains(t, entries, filename)
}

func TestConcurrentLogging(t *testing.T) {
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := slogrotate.NewLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := slogrotate.NewSlogRotateLogger(lumberjackLogger)

	numRoutines := 5
	done := make(chan bool, numRoutines)
	for i := 0; i < numRoutines; i++ {
		testNum := i
		go func() {
			logger.Warn("concurrency test", "test_num", testNum)
			done <- true
		}()
	}

	for i := 0; i < numRoutines; i++ {
		<-done
	}

	createdLogfile, err := os.Open(logfile)
	defer func() {
		_ = createdLogfile.Close()
	}()

	assert.Nil(t, err)
	createdLogfileScanner := bufio.NewScanner(bufio.NewReader(createdLogfile))
	var numLines int
	for createdLogfileScanner.Scan() {
		numLines++
	}

	assert.Equal(t, numRoutines, numLines)
}

func TestRotateInConcurrent(t *testing.T) {
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := slogrotate.NewLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := slogrotate.NewSlogRotateLogger(lumberjackLogger)

	numRoutines := 5
	done := make(chan bool, numRoutines)
	for i := 0; i < numRoutines; i++ {
		testNum := i
		go func() {
			logger.Warn("concurrency test", "test_num", testNum)
			if testNum%2 == 0 {
				rotateErr := lumberjackLogger.Rotate()
				assert.Nil(t, rotateErr)
			}
			done <- true
		}()
	}

	var numDone int
	for numDone < numRoutines {
		if <-done {
			numDone++
		}
	}

	close(done)

	logfiles := []string{}
	walkErr := filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		assert.Nil(t, err)

		if entry.Type().IsDir() {
			return nil
		}

		logfiles = append(logfiles, path)

		return nil
	})
	assert.Nil(t, walkErr)

	var numLines int
	for _, logfile := range logfiles {
		createdLogfile, err := os.Open(logfile)
		defer func() {
			closeErr := createdLogfile.Close()
			assert.Nil(t, closeErr)
		}()

		assert.Nil(t, err)

		logfileScanner := bufio.NewScanner(bufio.NewReader(createdLogfile))
		for logfileScanner.Scan() {
			numLines++
		}
	}

	assert.Equal(t, numRoutines, numLines)
}
