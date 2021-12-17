package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ralim/switchhost/utilities"
)

// Cleanup is notified anytime a folder has a file removed from it, which causes the parent search folder for that file to be scanned to remove empty folders
// This is messy, and open to ideas

func (lib *Library) cleanupFolderWorker() {
	for cleanupPath := range lib.folderCleanupRequests {
		if lib.settings.CleanupEmptyFolders {
			//need to check that this folder is inside one of the search folders && its not _the_ search folder
			if cleanupPath, err := filepath.Abs(cleanupPath); err == nil {
				ok := false
				parent := ""
				for _, baseFolder := range lib.settings.GetAllScanFolders() {
					if folderAbs, err := filepath.Abs(baseFolder); err == nil {
						if strings.HasPrefix(cleanupPath, folderAbs) {
							ok = true
							parent = folderAbs
						}
					}
				}
				if ok {
					go recursivelyCheckForEmptyFolders(parent)
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
						log.Info().Msgf("Removing %s as its empty", path)
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
