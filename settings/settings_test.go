package settings_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ralim/switchhost/settings"
)

func TestNewSettings(t *testing.T) {
	//Test that settings will init

	tempFile, err := os.CreateTemp("", "settings_test_*")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tempFile.Name())
	newSettings := settings.NewSettings(tempFile.Name())
	if newSettings.CacheFolder != "/tmp/" {
		t.Error("Should setup cache folder as demo overwrite")
	}

}

func TestLoadFrom(t *testing.T) {
	//Test that settings will init
	tempFile, err := os.CreateTemp("", "settings_test_*")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tempFile.Name())
	newSettings := settings.NewSettings(tempFile.Name())
	demoStr := "{\"cacheFolder\":\"testessetsteset\"}"
	reader := strings.NewReader(demoStr)
	newSettings.LoadFrom(reader)
	if newSettings.CacheFolder != "testessetsteset" {
		t.Error("Should setup cache folder as demo overwrite")
	}

}
