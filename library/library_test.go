package library

import (
	"fmt"
	"testing"
	"time"

	"github.com/ralim/switchhost/settings"
)

func TestStopStart(t *testing.T) {
	// Test that starting and stopping library with requests outstanding works
	// Not the best test in the world.. but seems to work for the point, which is that we should wait out the sleep before closing shop
	// Also making sure we dont race on channels etc

	//This works using sleep, and the fact that sleep adds args (sleeps total of args)
	t.Parallel()
	sett := settings.Settings{
		NSZCommandLine: "sleep 0.1",
	}
	lib := NewLibrary(nil, &sett)

	//Inject some pending requests
	lib.fileCompressionRequests <- fileScanningInfo{
		path: "0.02",
	}
	startime := time.Now()
	lib.Start()
	//Yield to let our sleep be selected before the close
	time.Sleep(time.Millisecond * 100)
	fmt.Println(".........")
	lib.Stop()
	duration := time.Since(startime)

	if duration.Milliseconds() < 100 {
		t.Error("Didnt wait for the sleep")
	}
}
