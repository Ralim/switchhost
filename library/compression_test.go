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
		settings:                &sett,
		fileCompressionRequests: make(chan string),
		fileScanRequests:        make(chan *scanRequest),
		waitgroup:               &sync.WaitGroup{},
	}
	lib.waitgroup.Add(1)

	defer close(lib.fileCompressionRequests)
	defer close(lib.fileScanRequests)

	go lib.compressionWorker()

	lib.fileCompressionRequests <- tempFile.Name()

	result := <-lib.fileScanRequests
	if result == nil {
		t.Error("bad result")
	} else {
		if !result.fileRemoved {
			t.Error("should report file removed")
		}
		if result.path != tempFile.Name() {
			t.Error("should report file path")
		}
	}
	//Test the made new file path
	lib.settings.NSZCommandLine = "touch"
	lib.fileCompressionRequests <- "/tmp/testingFiles.nsz"

	result = <-lib.fileScanRequests
	if result == nil {
		t.Error("bad result")
	} else {
		if result.fileRemoved {
			t.Error("should report file created")
		}
		if result.path != "/tmp/testingFiles.nsz" {
			t.Error("should report file path")
		}
	}
}
