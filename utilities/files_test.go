package utilities_test

import (
	"os"
	"testing"

	"github.com/ralim/switchhost/utilities"
)

func TestExists(t *testing.T) {
	t.Parallel()
	tempFile, err := os.CreateTemp("", "TestNSZCompressFile-*")
	if err != nil {
		t.Error(err)
	}
	_, err = tempFile.WriteString("Test")
	if err != nil {
		t.Error(err)
	}
	_ = tempFile.Close()

	exists := utilities.Exists(tempFile.Name())
	if !exists {
		t.Error("should work for known exising files")
	}
	exists = utilities.Exists("/")
	if !exists {
		t.Error("should work for known exising folder")
	}
	if err := os.Remove(tempFile.Name()); err != nil {
		t.Fatal(err)
	}
	exists = utilities.Exists(tempFile.Name())
	if exists {
		t.Error("should work for known not-exising files")
	}
}
