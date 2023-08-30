//nolint:testpackage // use same name as package to access variables to mock
package lumberjack

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	testifyAssert "github.com/stretchr/testify/assert"
)

// !!!NOTE!!!
//
// Running these tests in parallel will almost certainly cause sporadic (or even
// regular) failures, because they're all messing with the same global variable
// that controls the logic's mocked time.Now.  So... don't do that.

// Since all the tests uses the time to determine filenames etc, we need to
// control the wall clock as much as possible, which means having a wall clock
// that doesn't change unless we want it to. The same goes for random UUID.
//
//nolint:gochecknoglobals // need global time as we need to mock it across all tests
var fakeCurrentTime = time.Now()

func fakeTime() time.Time {
	return fakeCurrentTime
}

//nolint:gochecknoglobals // need global random UUID as we need to mock it across all tests
var fakeRandomUUID = uuid.New()

func fakeUUID() uuid.UUID {
	return fakeRandomUUID
}

func TestMain_NewFile(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID

	dir := t.TempDir()
	l := &Logger{
		Filename: logFile(dir),
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)
	fileContainsContent(t, logFile(dir), b)
	fileCount(dir, 1, t)
}

func TestMain_OpenExisting(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	dir := t.TempDir()

	filename := logFile(dir)
	data := []byte("foo!")
	err := os.WriteFile(filename, data, 0o644)
	testifyAssert.Nil(t, err)
	fileContainsContent(t, filename, data)

	l := &Logger{
		Filename: filename,
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	// make sure the file got appended
	fileContainsContent(t, filename, append(data, b...))

	// make sure no other files were created
	fileCount(dir, 1, t)
}

func TestMain_WriteTooLong(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1
	dir := t.TempDir()
	l := &Logger{
		Filename: logFile(dir),
		MaxSize:  5,
	}
	defer l.Close()
	b := []byte("booooooooooooooo!")
	n, err := l.Write(b)
	testifyAssert.NotNil(t, err)
	testifyAssert.Equal(t, 0, n)
	testifyAssert.Equal(t,
		err.Error(),
		fmt.Sprintf("write length %d exceeds maximum file size %d", len(b), l.MaxSize),
	)
	testifyAssert.NoFileExists(t, logFile(dir))
}

func TestMain_MakeLogDir(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	cwd := t.TempDir()
	dir := time.Now().Format("TestMain_MakeLogDir" + backupTimeFormat)
	dir = filepath.Join(cwd, dir)
	filename := logFile(dir)
	l := &Logger{
		Filename: filename,
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)
	fileContainsContent(t, logFile(dir), b)
	fileCount(dir, 1, t)
}

func TestMain_DefaultFilename(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	// use `os` instead of `t` to fit implementation of `Logger.filename()`
	dir := os.TempDir()
	filename := filepath.Join(dir, filepath.Base(os.Args[0])+"-lumberjack.log")
	defer os.Remove(filename)

	l := &Logger{}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)

	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)
	fileContainsContent(t, filename, b)
}

func TestMain_AutoRotate(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Filename: filename,
		MaxSize:  10,
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)

	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)
	fileContainsContent(t, filename, b)
	fileCount(dir, 1, t)

	newFakeTime()

	b2 := []byte("foooooo!")
	n, err = l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)

	// the old logfile should be moved aside and the main logfile should have
	// only the last write in it.
	fileContainsContent(t, filename, b2)

	// the backup file will use the current fake time and have the old contents.
	fileContainsContent(t, backupFile(dir), b)

	fileCount(dir, 2, t)
}

func TestMain_FirstWriteRotate(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1
	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Filename: filename,
		MaxSize:  10,
	}
	defer l.Close()

	start := []byte("boooooo!")
	err := os.WriteFile(filename, start, 0o600)
	testifyAssert.Nil(t, err)

	newFakeTime()

	// this would make us rotate
	b := []byte("fooo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	fileContainsContent(t, filename, b)
	fileContainsContent(t, backupFile(dir), start)

	fileCount(dir, 2, t)
}

func TestMain_MaxBackups(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1
	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 1,
	}
	defer l.Close()
	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	fileContainsContent(t, filename, b)
	fileCount(dir, 1, t)

	newFakeTime()

	// this will put us over the max
	b2 := []byte("foooooo!")
	n, err = l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)

	// this will use the new fake time
	secondFilename := backupFile(dir)
	fileContainsContent(t, secondFilename, b)

	// make sure the old file still exists with the same content.
	fileContainsContent(t, filename, b2)

	fileCount(dir, 2, t)

	newFakeTime()

	// this will make us rotate again
	b3 := []byte("baaaaaar!")
	n, err = l.Write(b3)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b3), n)

	// this will use the new fake time
	thirdFilename := backupFile(dir)
	fileContainsContent(t, thirdFilename, b2)

	fileContainsContent(t, filename, b3)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(time.Millisecond * 10)

	// should only have two files in the dir still
	fileCount(dir, 2, t)

	// second file name should still exist
	fileContainsContent(t, thirdFilename, b2)

	// should have deleted the first backup
	testifyAssert.NoFileExists(t, secondFilename)

	// now test that we don't delete directories or non-logfile files

	newFakeTime()

	// create a file that is close to but different from the logfile name.
	// It shouldn't get caught by our deletion filters.
	notlogfile := logFile(dir) + ".foo"
	err = os.WriteFile(notlogfile, []byte("data"), 0o644)
	testifyAssert.Nil(t, err)

	// Make a directory that exactly matches our log file filters... it still
	// shouldn't get caught by the deletion filter since it's a directory.
	notlogfiledir := backupFile(dir)
	err = os.Mkdir(notlogfiledir, 0o700)
	testifyAssert.Nil(t, err)

	newFakeTime()

	// this will use the new fake time
	fourthFilename := backupFile(dir)

	// Create a log file that is/was being compressed - this should
	// not be counted since both the compressed and the uncompressed
	// log files still exist.
	compLogFile := fourthFilename + compressSuffix
	err = os.WriteFile(compLogFile, []byte("compress"), 0o644)
	testifyAssert.Nil(t, err)

	// this will make us rotate again
	b4 := []byte("baaaaaaz!")
	n, err = l.Write(b4)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b4), n)

	fileContainsContent(t, fourthFilename, b3)
	fileContainsContent(t, fourthFilename+compressSuffix, []byte("compress"))

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(time.Millisecond * 10)

	// We should have four things in the directory now - the 2 log files, the
	// not log file, and the directory
	fileCount(dir, 5, t)

	// third file name should still exist
	fileContainsContent(t, filename, b4)

	fileContainsContent(t, fourthFilename, b3)

	// should have deleted the first filename
	testifyAssert.NoFileExists(t, thirdFilename)

	// the not-a-logfile should still exist
	testifyAssert.FileExists(t, notlogfile)

	// the directory
	testifyAssert.DirExists(t, notlogfiledir)
}

func TestMain_CleanupExistingBackups(t *testing.T) {
	// test that if we start with more backup files than we're supposed to have
	// in total, that extra ones get cleaned up when we rotate.

	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	// make 3 backup files

	data := []byte("data")
	backup := backupFile(dir)
	err := os.WriteFile(backup, data, 0o644)
	testifyAssert.Nil(t, err)

	newFakeTime()

	backup = backupFile(dir)
	err = os.WriteFile(backup+compressSuffix, data, 0o644)
	testifyAssert.Nil(t, err)

	newFakeTime()

	backup = backupFile(dir)
	err = os.WriteFile(backup, data, 0o644)
	testifyAssert.Nil(t, err)

	// now create a primary log file with some data
	filename := logFile(dir)
	err = os.WriteFile(filename, data, 0o644)
	testifyAssert.Nil(t, err)

	l := &Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 1,
	}
	defer l.Close()

	newFakeTime()

	b2 := []byte("foooooo!")
	n, err := l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(time.Millisecond * 10)

	// now we should only have 2 files left - the primary and one backup
	fileCount(dir, 2, t)
}

func TestMain_MaxAge(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Filename: filename,
		MaxSize:  10,
		MaxAge:   1,
	}
	defer l.Close()

	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	fileContainsContent(t, filename, b)
	fileCount(dir, 1, t)

	// two days later
	newFakeTime()

	b2 := []byte("foooooo!")
	n, err = l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)
	fileContainsContent(t, backupFile(dir), b)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(10 * time.Millisecond)

	// We should still have 2 log files, since the most recent backup was just
	// created.
	fileCount(dir, 2, t)

	fileContainsContent(t, filename, b2)

	// we should have deleted the old file due to being too old
	fileContainsContent(t, backupFile(dir), b)

	// two days later
	newFakeTime()

	b3 := []byte("baaaaar!")
	n, err = l.Write(b3)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b3), n)
	fileContainsContent(t, backupFile(dir), b2)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(10 * time.Millisecond)

	// We should have 2 log files - the main log file, and the most recent
	// backup.  The earlier backup is past the cutoff and should be gone.
	fileCount(dir, 2, t)

	fileContainsContent(t, filename, b3)

	// we should have deleted the old file due to being too old
	fileContainsContent(t, backupFile(dir), b2)
}

func TestMain_OldLogFiles(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	filename := logFile(dir)
	data := []byte("data")
	err := os.WriteFile(filename, data, 0o7)
	testifyAssert.Nil(t, err)

	// This gives us a time with the same precision as the time we get from the
	// timestamp in the name.
	t1, err := time.Parse(backupTimeFormat, fakeTime().UTC().Format(backupTimeFormat))
	testifyAssert.Nil(t, err)

	backup := backupFile(dir)
	err = os.WriteFile(backup, data, 0o7)
	testifyAssert.Nil(t, err)

	newFakeTime()

	t2, err := time.Parse(backupTimeFormat, fakeTime().UTC().Format(backupTimeFormat))
	testifyAssert.Nil(t, err)

	backup2 := backupFile(dir)
	err = os.WriteFile(backup2, data, 0o7)
	testifyAssert.Nil(t, err)

	l := &Logger{Filename: filename}
	files, err := l.oldLogFiles()
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, 2, len(files))

	// should be sorted by newest file first, which would be t2
	testifyAssert.Equal(t, t2, files[0].timestamp)
	testifyAssert.Equal(t, t1, files[1].timestamp)
}

func TestMain_TimeFromName(t *testing.T) {
	l := &Logger{Filename: "/var/log/myfoo/foo.log"}
	prefix, ext := l.prefixAndExt()

	tests := []struct {
		filename string
		want     time.Time
		wantErr  bool
	}{
		{"foo-2014-05-04T14-44-33.555-12345678.log", time.Date(2014, 5, 4, 14, 44, 33, 555000000, time.UTC), false},
		{"foo-2014-05-04T14-44-33.555-12345678", time.Time{}, true},
		{"2014-05-04T14-44-33.555-12345678.log", time.Time{}, true},
		{"foo.log", time.Time{}, true},
	}

	for _, test := range tests {
		got, err := l.timeFromName(test.filename, prefix, ext)
		testifyAssert.Equal(t, test.want, got)
		testifyAssert.Equal(t, err != nil, test.wantErr)
	}
}

func TestMain_LocalTime(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	l := &Logger{
		Filename:  logFile(dir),
		MaxSize:   10,
		LocalTime: true,
	}
	defer l.Close()

	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	b2 := []byte("fooooooo!")
	n2, err := l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n2)

	fileContainsContent(t, logFile(dir), b2)
	fileContainsContent(t, backupFileLocal(dir), b)
}

func TestMain_Rotate(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	dir := t.TempDir()

	filename := logFile(dir)

	l := &Logger{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer l.Close()

	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	fileContainsContent(t, filename, b)
	fileCount(dir, 1, t)

	newFakeTime()

	err = l.Rotate()
	testifyAssert.Nil(t, err)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(10 * time.Millisecond)

	filename2 := backupFile(dir)
	fileContainsContent(t, filename2, b)
	fileContainsContent(t, filename, []byte{})
	fileCount(dir, 2, t)
	newFakeTime()

	err = l.Rotate()
	testifyAssert.Nil(t, err)

	// we need to wait a little bit since the files get deleted on a different
	// goroutine.
	<-time.After(10 * time.Millisecond)

	filename3 := backupFile(dir)
	fileContainsContent(t, filename3, []byte{})
	fileContainsContent(t, filename, []byte{})
	fileCount(dir, 2, t)

	b2 := []byte("foooooo!")
	n, err = l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)

	// this will use the new fake time
	fileContainsContent(t, filename, b2)
}

func TestMain_CompressOnRotate(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Compress: true,
		Filename: filename,
		MaxSize:  10,
	}
	defer l.Close()

	b := []byte("boo!")
	n, err := l.Write(b)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b), n)

	fileContainsContent(t, filename, b)
	fileCount(dir, 1, t)

	newFakeTime()

	err = l.Rotate()
	testifyAssert.Nil(t, err)

	// the old logfile should be moved aside and the main logfile should have
	// nothing in it.
	fileContainsContent(t, filename, []byte{})

	// we need to wait a little bit since the files get compressed on a different
	// goroutine.
	<-time.After(300 * time.Millisecond)

	// a compressed version of the log file should now exist and the original
	// should have been removed.
	bc := new(bytes.Buffer)
	gz := gzip.NewWriter(bc)
	_, err = gz.Write(b)
	testifyAssert.Nil(t, err)
	err = gz.Close()
	testifyAssert.Nil(t, err)
	fileContainsContent(t, backupFile(dir)+compressSuffix, bc.Bytes())
	testifyAssert.NoFileExists(t, backupFile(dir))

	fileCount(dir, 2, t)
}

func TestMain_CompressOnResume(t *testing.T) {
	currentTime = fakeTime
	newUUID = fakeUUID
	megabyte = 1

	dir := t.TempDir()

	filename := logFile(dir)
	l := &Logger{
		Compress: true,
		Filename: filename,
		MaxSize:  10,
	}
	defer l.Close()

	// Create a backup file and empty "compressed" file.
	filename2 := backupFile(dir)
	b := []byte("foo!")
	err := os.WriteFile(filename2, b, 0o644)
	testifyAssert.Nil(t, err)
	err = os.WriteFile(filename2+compressSuffix, []byte{}, 0o644)
	testifyAssert.Nil(t, err)

	newFakeTime()

	b2 := []byte("boo!")
	n, err := l.Write(b2)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, len(b2), n)
	fileContainsContent(t, filename, b2)

	// we need to wait a little bit since the files get compressed on a different
	// goroutine.
	<-time.After(300 * time.Millisecond)

	// The write should have started the compression - a compressed version of
	// the log file should now exist and the original should have been removed.
	bc := new(bytes.Buffer)
	gz := gzip.NewWriter(bc)
	_, err = gz.Write(b)
	testifyAssert.Nil(t, err)
	err = gz.Close()
	testifyAssert.Nil(t, err)
	fileContainsContent(t, filename2+compressSuffix, bc.Bytes())
	testifyAssert.NoFileExists(t, filename2)

	fileCount(dir, 2, t)
}

func TestMain_Json(t *testing.T) {
	data := []byte(`
{
	"filename": "foo",
	"maxsize": 5,
	"maxage": 10,
	"maxbackups": 3,
	"localtime": true,
	"compress": true
}`[1:])

	l := Logger{}
	err := json.Unmarshal(data, &l)
	testifyAssert.Nil(t, err)
	testifyAssert.Equal(t, "foo", l.Filename)
	testifyAssert.Equal(t, 5, l.MaxSize)
	testifyAssert.Equal(t, 10, l.MaxAge)
	testifyAssert.Equal(t, 3, l.MaxBackups)
	testifyAssert.Equal(t, true, l.LocalTime)
	testifyAssert.Equal(t, true, l.Compress)
}

// logFile returns the log file name in the given directory for the current fake
// time.
func logFile(dir string) string {
	return filepath.Join(dir, "foobar.log")
}

func backupFile(dir string) string {
	return filepath.Join(
		dir, "foobar-"+
			fakeTime().UTC().Format(backupTimeFormat)+
			"-"+newUUID().String()[:randomSuffixLen]+
			".log")
}

func backupFileLocal(dir string) string {
	return filepath.Join(dir,
		"foobar-"+
			fakeTime().Format(backupTimeFormat)+
			"-"+newUUID().String()[:randomSuffixLen]+
			".log")
}

// fileCount checks that the number of files in the directory is exp.
func fileCount(dir string, exp int, t testing.TB) {
	files, err := os.ReadDir(dir)
	testifyAssert.Nil(t, err)
	// Make sure no other files were created.
	testifyAssert.Equal(t, exp, len(files))
}

// newFakeTime sets the fake "current time" to two days later.
func newFakeTime() {
	fakeCurrentTime = fakeCurrentTime.Add(time.Hour * 24 * 2)
}

// resetMocks resets mocks set in the above tests.
func resetMocks() {
	currentTime = time.Now
	newUUID = uuid.New
	osStat = os.Stat
	megabyte = 1024 * 1024
}

// fileContainsContent checks if the bytes in `logfilepath` contains the expected content string.
func fileContainsContent(t *testing.T, logfilepath string, expectedContent []byte) {
	testifyAssert.FileExists(t, logfilepath)
	bytesInFile, err := os.ReadFile(logfilepath)
	testifyAssert.Nil(t, err)
	testifyAssert.Contains(t, string(bytesInFile), string(expectedContent))
}
