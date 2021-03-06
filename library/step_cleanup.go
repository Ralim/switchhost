package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

// Cleanup is notified anytime a folder has a file removed from it, which causes the parent search folder for that file to be scanned to remove empty folders
// This is messy, and open to ideas

func (lib *Library) cleanupFolderWorker() {
	defer lib.waitgroup.Done()
	defer log.Info().Msg("Cleanup task exiting")
	var status *termui.TaskState
	if lib.ui != nil {
		status = lib.ui.RegisterTask("Cleanup")
		defer status.UpdateStatus("Exited")
		status.UpdateStatus("Idle")
	}
	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case cleanupPath := <-lib.folderCleanupRequests:
			if lib.settings.CleanupEmptyFolders {
				//need to check that this folder is inside one of the search folders && its not _the_ search folder
				if cleanupPath, err := filepath.Abs(cleanupPath); err == nil {
					ok := false
					parent := ""
					for _, baseFolder := range lib.settings.GetAllScanFolders() {
						if folderAbs, err := filepath.Abs(baseFolder); err == nil {
							if strings.HasPrefix(cleanupPath, folderAbs) && cleanupPath != folderAbs {
								ok = true
								parent = folderAbs
							}
						}
					}
					if ok {
						if status != nil {
							status.UpdateStatus(parent)
						}
						recursivelyCheckForEmptyFolders(parent)
						if status != nil {
							status.UpdateStatus("Idle")
						}
					}
				}
			}
		}
	}

}

func recursivelyCheckForEmptyFolders(pathin string) {
	err := filepath.Walk(pathin,
		func(path string, info os.FileInfo, err error) error {
			if pathin != path {
				if err != nil {
					return err
				}
				if info.IsDir() {
					recursivelyCheckForEmptyFolders(path)
					if empty, err := utilities.IsEmpty(path); err == nil && empty {
						log.Info().Str("path", path).Msg("Removing empty path")
						os.Remove(path)
					}
				}
			}
			return nil
		})
	if err != nil {
		log.Debug().Err(err).Str("path", pathin).Msg("Cant clean folder")
	}
}
