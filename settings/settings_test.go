package settings_test

import (
	"encoding/json"
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
	defer func() { _ = os.Remove(tempFile.Name()) }()
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
	defer func() { _ = os.Remove(tempFile.Name()) }()
	newSettings := settings.NewSettings(tempFile.Name())
	demoStr := "{\"cacheFolder\":\"testessetsteset\"}"
	reader := strings.NewReader(demoStr)
	newSettings.LoadFrom(reader)
	if newSettings.CacheFolder != "testessetsteset" {
		t.Error("Should setup cache folder as demo overwrite")
	}

}

func TestNSZCommandLineDefaultIsValid(t *testing.T) {
	tempFile, err := os.CreateTemp("", "settings_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	newSettings := settings.NewSettings(tempFile.Name())
	for _, part := range strings.Fields(newSettings.NSZCommandLine) {
		if part == "-p" || part == "--parseCnmt" {
			t.Errorf("default NSZCommandLine contains flag removed in nsz 5: %q", newSettings.NSZCommandLine)
		}
	}
}

func TestNSZCommandLineRemovedFlagsStripped(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"old default", "nsz --verify -w -C -p -t 4 --rm-source ", "nsz --verify -w -C -t 4 --rm-source"},
		{"long form", "nsz --parseCnmt -C", "nsz -C"},
		{"full path", "/usr/local/bin/nsz -p -C", "/usr/local/bin/nsz -C"},
		{"windows exe", "nsz.exe -C -p", "nsz.exe -C"},
		{"already clean", "nsz -C -t 4 ", "nsz -C -t 4 "},
		{"other command untouched", "othertool -p -C", "othertool -p -C"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &settings.Settings{}
			payload, err := json.Marshal(map[string]string{"NSZCommandLine": tt.in})
			if err != nil {
				t.Fatal(err)
			}
			s.LoadFrom(strings.NewReader(string(payload)))
			if s.NSZCommandLine != tt.want {
				t.Errorf("got %q, want %q", s.NSZCommandLine, tt.want)
			}
		})
	}
}

func TestNSZCommandLineCleanedOnLoad(t *testing.T) {
	tempFile, err := os.CreateTemp("", "settings_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	if _, err := tempFile.WriteString(`{"NSZCommandLine":"nsz --verify -w -C -p -t 4 --rm-source "}`); err != nil {
		t.Fatal(err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatal(err)
	}
	newSettings := settings.NewSettings(tempFile.Name())
	if want := "nsz --verify -w -C -t 4 --rm-source"; newSettings.NSZCommandLine != want {
		t.Errorf("got %q, want %q", newSettings.NSZCommandLine, want)
	}
}
