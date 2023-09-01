package woodcutter

import (
	"bufio"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newLumberjackLogger(
	logfilepath string,
	maxBackups, maxAge, maxSize int,
	localtime, compress bool,
) *Logger {
	return &Logger{
		Filename:   logfilepath,
		LocalTime:  localtime,
		Compress:   compress,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
	}
}

func newSlogRotateLogger(lumberjackLogger *Logger) *slog.Logger {
	return slog.New(slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{}))
}

func TestSlog_CreationOfLogFile(t *testing.T) {
	resetMocks()
	cwd := t.TempDir()
	logfile := filepath.Join(cwd, "test.log")
	lumberjackLogger := newLumberjackLogger(logfile, 1, 1, 1, true, true)
	logger := newSlogRotateLogger(lumberjackLogger)

	logger.Warn("this is a test", "test_key", "test_value")

	assert.FileExists(t, logfile)
}

func TestSlog_Rotation(t *testing.T) {
	resetMocks()
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := newLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := newSlogRotateLogger(lumberjackLogger)

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

func TestSlog_ConcurrentLogging(t *testing.T) {
	resetMocks()
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := newLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := newSlogRotateLogger(lumberjackLogger)

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

func TestSlog_RotateInConcurrent(t *testing.T) {
	resetMocks()
	cwd := t.TempDir()
	filename := "test.log"
	logfile := filepath.Join(cwd, filename)
	lumberjackLogger := newLumberjackLogger(logfile, 0, 0, 0, false, false)
	logger := newSlogRotateLogger(lumberjackLogger)

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
