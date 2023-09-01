//go:build linux || darwin
// +build linux darwin

package woodcutter

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDarwin_MaintainMode(t *testing.T) {
	resetMocks()
	currentTime = fakeTime
	newUUID = fakeUUID
	cwd := t.TempDir()

	filename := logFile(cwd)

	mode := os.FileMode(0o600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	assert.Nil(t, err)
	f.Close()

	l := &Logger{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	newFakeTime()

	err = l.Rotate()
	assert.Nil(t, err)

	filename2 := backupFile(cwd)
	info, err := os.Stat(filename)
	assert.Nil(t, err)
	info2, err := os.Stat(filename2)
	assert.Nil(t, err)
	assert.Equal(t, mode, info.Mode())
	assert.Equal(t, mode, info2.Mode())
}

func TestDarwin_MaintainOwner(t *testing.T) {
	resetMocks()
	fakeFS := newFakeFS()
	newUUID = fakeUUID

	osChown = fakeFS.Chown
	osStat = fakeFS.Stat
	defer func() {
		osChown = os.Chown
		osStat = os.Stat
	}()
	currentTime = fakeTime
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	filename := logFile(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0o644)
	assert.Nil(t, err)
	f.Close()

	l := &Logger{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	newFakeTime()

	err = l.Rotate()
	assert.Nil(t, err)

	assert.Equal(t, 555, fakeFS.files[filename].uid)
	assert.Equal(t, 666, fakeFS.files[filename].gid)
}

func TestDarwin_CompressMaintainMode(t *testing.T) {
	resetMocks()
	currentTime = fakeTime
	newUUID = fakeUUID

	dir := t.TempDir()
	filename := logFile(dir)

	mode := os.FileMode(0o600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	assert.Nil(t, err)
	f.Close()

	l := &Logger{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	newFakeTime()

	err = l.Rotate()
	assert.Nil(t, err)

	// we need to wait a little bit since the files get compressed on a different
	// goroutine.
	<-time.After(20 * time.Millisecond)

	// a compressed version of the log file should now exist with the correct
	// mode.
	filename2 := backupFile(dir)
	info, err := os.Stat(filename)
	assert.Nil(t, err)
	info2, err := os.Stat(filename2 + compressSuffix)
	assert.Nil(t, err)
	assert.Equal(t, mode, info.Mode())
	assert.Equal(t, mode, info2.Mode())
}

func TestDarwin_CompressMaintainOwner(t *testing.T) {
	resetMocks()
	fakeFS := newFakeFS()
	newUUID = fakeUUID

	osChown = fakeFS.Chown
	osStat = fakeFS.Stat
	defer func() {
		osChown = os.Chown
		osStat = os.Stat
	}()
	currentTime = fakeTime
	dir := t.TempDir()

	filename := logFile(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0o644)
	assert.Nil(t, err)
	f.Close()

	l := &Logger{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	newFakeTime()

	err = l.Rotate()
	assert.Nil(t, err)

	// we need to wait a little bit since the files get compressed on a different
	// goroutine.
	<-time.After(10 * time.Millisecond)

	// a compressed version of the log file should now exist with the correct
	// owner.
	filename2 := backupFile(dir)
	assert.Equal(t, 555, fakeFS.files[filename2+compressSuffix].uid)
	assert.Equal(t, 666, fakeFS.files[filename2+compressSuffix].gid)
}

type fakeFile struct {
	uid int
	gid int
}

type fakeFS struct {
	files map[string]fakeFile
}

func newFakeFS() *fakeFS {
	return &fakeFS{files: make(map[string]fakeFile)}
}

func (fs *fakeFS) Chown(name string, uid, gid int) error {
	fs.files[name] = fakeFile{uid: uid, gid: gid}
	return nil
}

func (fs *fakeFS) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("could not get file info for %s", name)
	}
	stat.Uid = 555
	stat.Gid = 666
	return info, nil
}
