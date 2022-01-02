package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Scanner performs startup scans of incoming folders + library to populate the db

// RunScan runs a scan of all "normal" scan folders
func (lib *Library) RunScan() {
	defer lib.waitgroup.Done()
	for _, folder := range lib.settings.GetAllScanFolders() {
		if err := lib.ScanFolder(folder); err == nil {
			lib.folderCleanupRequests <- folder
		}

	}
	//end marker to allow indication to users that scan is done
	event := &scanRequest{
		path:             "",
		isEndOfStartScan: true,
		isNotifierBased:  false,
	}
	lib.fileScanRequests <- event
}

//ScanFolder recursively scans the provied folder and feeds it to the organisation queue
func (lib *Library) ScanFolder(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil {

			if !info.IsDir() {

				ext := filepath.Ext(path)
				ext = strings.ToUpper(ext)
				shouldScan := false
				switch ext {
				case ".NSP":
					shouldScan = true
				case ".NSZ":
					shouldScan = true
				case ".XCI":
					shouldScan = true
				case ".XCZ":
					shouldScan = true
				}
				if shouldScan {
					//This is a file, so push it to the queue
					log.Debug().Str("path", path).Msg("File scan requested")
					event := &scanRequest{
						path:             path,
						isEndOfStartScan: false,
						isNotifierBased:  false,
					}
					lib.fileScanRequests <- event
				}
			}
		}
		return nil
	})
}