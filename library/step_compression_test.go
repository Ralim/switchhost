package library

import (
	"os"
	"sync"
	"testing"

	"github.com/ralim/switchhost/settings"
)

func TestNSZCompressFile(t *testing.T) {
	t.Parallel()
	tempFile, err := os.CreateTemp("", "TestNSZCompressFile-*")
	if err != nil {
		t.Error(err)
	}
	tempFile.WriteString("Test")
	tempFile.Close()
	defer os.Remove(tempFile.Name())
	sett := settings.Settings{
		NSZCommandLine: "cat ",
	}
	lib := Library{
		settings: &sett,
	}
	err = lib.NSZCompressFile(tempFile.Name())
	if err != nil {
		t.Error("Should run normal commands fine", err)
	}
	sett.NSZCommandLine = "kfkjhksjhfksjhfksjdhfksjdfhksjhfdksdjhfksjhfksjdhfksdhfksjfd"
	err = lib.NSZCompressFile(tempFile.Name())
	if err == nil {
		t.Error("Should throw error on bad ")
	}
	sett.NSZCommandLine = "cat //"
	err = lib.NSZCompressFile(tempFile.Name())
	if err == nil {
		t.Error("Should throw error on bad ")
	}
}

func TestCompressionWorker(t *testing.T) {
	t.Parallel()
	tempFile, err := os.CreateTemp("", "TestNSZCompressFile-*")
	if err != nil {
		t.Error(err)
	}
	tempFile.WriteString("Test")
	tempFile.Close()
	defer os.Remove(tempFile.Name())
	sett := settings.Settings{
		NSZCommandLine: "rm",
	}
	lib := Library{
		settings:                 &sett,
		fileCompressionRequests:  make(chan *fileScanningInfo, 10),
		fileMetaScanRequests:     make(chan *fileScanningInfo, 10),
		fileOrganisationRequests: make(chan *fileScanningInfo, 10),
		waitgroup:                &sync.WaitGroup{},
	}
	lib.waitgroup.Add(1)

	defer close(lib.fileCompressionRequests)
	defer close(lib.fileMetaScanRequests)
	defer close(lib.fileOrganisationRequests)

	go lib.compressionWorker()

	lib.fileCompressionRequests <- &fileScanningInfo{
		path: tempFile.Name(),
	}

	result := <-lib.fileOrganisationRequests

	if !result.fileWasDeleted {
		t.Error("should report file removed")
	}
	if result.path != tempFile.Name() {
		t.Error("should report file path")
	}
	tmpfile, err := os.CreateTemp("", "test_comp*.nsz")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmpfile.Name())
	//Test the made new file path
	lib.settings.NSZCommandLine = "touch"
	lib.fileCompressionRequests <- &fileScanningInfo{
		path: tmpfile.Name(),
	}

	result = <-lib.fileMetaScanRequests

	if result.fileWasDeleted {
		t.Error("should report file created")
	}
	if result.path != tmpfile.Name() {
		t.Error("should report file path")
	}
}
