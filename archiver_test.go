package debos_test

import (
	"bytes"
	"fmt"
	"github.com/go-debos/debos"
	"github.com/pkg/xattr"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/user"
	_ "reflect"
	_ "strings"
	"testing"
)

func TestBase(t *testing.T) {

	// New archive
	// Expect Tar by default
	_, err := debos.NewArchive("test.base", 0)
	assert.EqualError(t, err, "Unsupported archive 'test.base'")

	// Test base
	archive := debos.ArchiveBase{}
	arcType := archive.Type()
	assert.Equal(t, 0, int(arcType))

	// Add  option
	err = archive.AddOption("someoption", "somevalue")
	assert.Empty(t, err)

	err = archive.Unpack("/tmp/test")
	assert.EqualError(t, err, "Unpack is not supported for ''")
	err = archive.RelaxedUnpack("/tmp/test")
	assert.EqualError(t, err, "Unpack is not supported for ''")
}

func TestTar_default(t *testing.T) {

	// New archive
	// Expect Tar by default
	archive, err := debos.NewArchive("doesnotexists.tar.gz")
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)

	// Type must be Tar by default
	arcType := archive.Type()
	assert.Equal(t, debos.Tar, arcType)

	// Test unpack
	err = archive.Unpack("/tmp/test")
	// Expect unpack failure
	assert.EqualError(t, err, "exit status 2")

	// Expect failure for RelaxedUnpack
	err = archive.RelaxedUnpack("/tmp/test")
	assert.EqualError(t, err, "exit status 2")

	// Check options
	err = archive.AddOption("taroptions", []string{"--option1"})
	assert.Empty(t, err)
	err = archive.Unpack("/tmp/test")
	assert.EqualError(t, err, "exit status 64")
	err = archive.Unpack("/proc/debostest")
	assert.EqualError(t, err, "mkdir /proc/debostest: no such file or directory")
	err = archive.RelaxedUnpack("/tmp/test")
	assert.EqualError(t, err, "exit status 64")

	// Add wrong option
	err = archive.AddOption("someoption", "somevalue")
	assert.EqualError(t, err, "Option 'someoption' is not supported for tar archive type")
}

func checkXAttrActuallyWork(t *testing.T, tmpDir string) error {
	// ensure that tmpDir points to a filesystem where xattr are supported
	f, err := ioutil.TempFile(tmpDir, "")
	if err != nil {
		return fmt.Errorf("unable to create temporary file: %v", err)
	}
	defer os.Remove(f.Name())
	assert.Empty(t, err)
	// this should roughly match the `cap_sys_pacct+p` capabilities set on
	// `test-xattr` in `testdata/test.tar.gz`
	xattrName := "security.capability"
	xattrValue := []byte{0, 0, 0, 2, 0, 0, 16, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	err = xattr.Set(f.Name(), xattrName, xattrValue)
	if err != nil {
		return err
	}
	readbackValue, err := xattr.Get(f.Name(), xattrName)
	if err != nil {
		return err
	}
	if !bytes.Equal(xattrValue, readbackValue) {
		return fmt.Errorf("unable to read back the configured attribute")
	}
	return nil
}

func TestTar_extraction_xattr(t *testing.T) {
	u, err := user.Current()
	assert.Empty(t, err)
	if u.Uid != "0" {
		t.Skip("skipping test; needs to be run as UID 0")
	}

	tmpDir := ""
	// check that the default temporary directory supports setting xattrs
	err = checkXAttrActuallyWork(t, tmpDir)
	if err != nil {
		t.Skip("skipping test; xattr support unavailable:", err)
	}

	archive, err := debos.NewArchive("testdata/test.tar.gz")
	dir, err := ioutil.TempDir(tmpDir, "debos-test-")
	assert.Empty(t, err)
	err = archive.Unpack(dir)
	assert.Empty(t, err)
	path := dir + "/test-xattr"
	var data []byte
	data, err = xattr.Get(path, "security.capability")
	assert.Empty(t, err)
	assert.NotNil(t, data)
}

// Check supported compression types
func TestTar_compression(t *testing.T) {
	compressions := map[string]string{
		"gz":    "tar -C test -x -z -f doesnotexists.tar.gz",
		"bzip2": "tar -C test -x -j -f doesnotexists.tar.gz",
		"xz":    "tar -C test -x -J -f doesnotexists.tar.gz",
	}

	// Force type
	archive, err := debos.NewArchive("doesnotexists.tar.gz", debos.Tar)
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)
	// Type must be Tar
	arcType := archive.Type()
	assert.Equal(t, debos.Tar, arcType)

	for compression, _ := range compressions {
		err = archive.AddOption("tarcompression", compression)
		assert.Empty(t, err)
		err := archive.Unpack("test")
		assert.EqualError(t, err, "exit status 2")
	}
	// Check of unsupported compression type
	err = archive.AddOption("tarcompression", "fake")
	assert.EqualError(t, err, "Compression 'fake' is not supported")

	// Pass incorrect type
	err = archive.AddOption("taroptions", nil)
	assert.EqualError(t, err, "Wrong type for value")
	err = archive.AddOption("tarcompression", nil)
	assert.EqualError(t, err, "Wrong type for value")
}

func TestDeb_notexisting(t *testing.T) {

	// Guess Deb
	archive, err := debos.NewArchive("doesnotexists.deb")
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)

	// Type must be guessed as Deb
	arcType := archive.Type()
	assert.Equal(t, debos.Deb, arcType)

	// Force Deb type
	archive, err = debos.NewArchive("doesnotexists.deb", debos.Deb)
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)

	// Type must be Deb
	arcType = archive.Type()
	assert.Equal(t, debos.Deb, arcType)

	// Expect unpack failure
	err = archive.Unpack("/tmp/test")
	assert.EqualError(t, err, "exit status 2")
	err = archive.Unpack("/proc/debostest")
	assert.EqualError(t, err, "mkdir /proc/debostest: no such file or directory")
	err = archive.RelaxedUnpack("/tmp/test")
	assert.EqualError(t, err, "exit status 2")
}

func TestDeb(t *testing.T) {
	u, err := user.Current()
	if u.Uid != "0" {
		t.Skip("skipping test; needs to be run as UID 0")
	}
	archive, err := debos.NewArchive("testdata/hello_2.10-2_amd64.deb")
	dir, err := ioutil.TempDir("", "debos-test-")
	assert.Empty(t, err)
	err = archive.Unpack(dir)
	assert.Empty(t, err)
}

func TestZip_notexisting(t *testing.T) {
	// Guess zip
	archive, err := debos.NewArchive("doesnotexists.ZiP")
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)
	// Type must be guessed as Zip
	arcType := archive.Type()
	assert.Equal(t, debos.Zip, arcType)

	// Force Zip type
	archive, err = debos.NewArchive("doesnotexists.zip", debos.Zip)
	assert.NotEmpty(t, archive)
	assert.Empty(t, err)

	// Type must be Zip
	arcType = archive.Type()
	assert.Equal(t, debos.Zip, arcType)

	// Expect unpack failure
	err = archive.Unpack("/tmp/test")
	assert.EqualError(t, err, "exit status 9")
	err = archive.Unpack("/proc/debostest")
	assert.EqualError(t, err, "mkdir /proc/debostest: no such file or directory")
	err = archive.RelaxedUnpack("/tmp/test")
	assert.EqualError(t, err, "exit status 9")
}
