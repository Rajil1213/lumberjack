//nolint:testpackage // use same name as package to access variables to mock
package lumberjack

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// To use lumberjack with the standard library's log package, just pass it into
// the SetOutput function when your application starts.
func TestExample_UsageWithStandardLogger(t *testing.T) {
	cwd := t.TempDir()
	logfilename := filepath.Join(cwd, "foo.log")
	log.SetOutput(&Logger{
		Filename:   logfilename,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   // days
		Compress:   true, // disabled by default
	})

	logger := log.Default()

	content := "test logging with standard logger"
	logger.Printf(content)

	assert.FileExists(t, logfilename)
	fileContainsContent(t, logfilename, []byte(content))
}
