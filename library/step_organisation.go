package library

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/formats"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

//Organisation takes incoming files and sorts their file paths and adds them to the library

const (
	FormatNameSub       = "{TitleName}"
	FormatTitleIDSub    = "{TitleID}"
	FormatVersionSub    = "{Version}"
	FormatVersionDecSub = "{VersionDec}"
	FormatTypeSub       = "{Type}"
)

func (lib *Library) fileorganisationWorker() {
	defer lib.waitgroup.Done()
	defer log.Info().Msg("fileorganisationWorker task exiting")
	status := lib.ui.RegisterTask("Organisation")
	defer status.UpdateStatus("Exited")

	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case event := <-lib.fileOrganisationRequests:
			fileShortName := path.Base(event.path)
			if event.fileWasDeleted {
				status.UpdateStatus(fmt.Sprintf("Handling Delete of %s", fileShortName))
				lib.sortFileHandleRemoved(event)
			} else {
				info := event.metadata
				status.UpdateStatus(fmt.Sprintf("Sorting %s", fileShortName))
				fileResultingPath := lib.sortFileIfApplicable(info, event.path, event.mustCleanupFile)
				status.UpdateStatus(fmt.Sprintf("Processing %s", fileShortName))
				//Add to our repo, moved or not
				record := &FileOnDiskRecord{
					Path:    fileResultingPath,
					TitleID: info.TitleID,
					Version: info.Version,
					Name:    info.EmbeddedTitle,
					Size:    info.Size,
				}
				if gameTitle, err := lib.QueryGameTitleFromTitleID(info.TitleID); err == nil {
					record.Name = gameTitle
				}

				lib.AddFileRecord(record)
				lib.postFileAddToLibraryHooks(event)
				status.UpdateStatus("Idle")
			}
		}
	}
}

func (lib *Library) postFileAddToLibraryHooks(event *fileScanningInfo) {
	//Dispatch any post hooks
	if lib.settings.CompressionEnabled {
		extension := strings.ToLower(path.Ext(event.path))
		if len(extension) == 4 {
			if extension[3] != 'z' {
				//File might be compressable, send it off
				log.Info().Str("path", event.path).Msg("Adding to compression list")
				lib.fileCompressionRequests <- event
			}
		}
	}
}

// sortFileIfApplicable; if sorting is on, attempts to sort the file to the new path if its different.
// If sorting is turned off, or if the sorting fails for one reason or another, just returns the source path
// If the file is moved, it returns the updated path
// If the file is moved, it will also notify the cleanup handler to go scan if the folder needs cleanup
func (lib *Library) sortFileIfApplicable(infoInfo *formats.FileInfo, currentPath string, isIncomingFile bool) string {
	shouldSort := lib.settings.EnableSorting
	if isIncomingFile {
		shouldSort = true // Have to sort incoming files
	}
	// If sorting is off, no-op
	if !shouldSort || lib.keys == nil {
		return currentPath
	}
	newPath, err := lib.determineIdealFilePath(infoInfo, currentPath)
	if err != nil {
		log.Warn().Err(err).Str("path", currentPath).Msg("Determining ideal path failed")
		return currentPath
	}
	if err == nil {
		if newPath != currentPath {
			log.Debug().Str("oldPath", currentPath).Str("newPath", newPath).Msg("Attempting move")
			err := os.MkdirAll(path.Dir(newPath), 0755)
			if err != nil {
				log.Warn().Str("oldPath", currentPath).Str("newPath", newPath).Err(err).Msg("Moving file raised error")
			} else {
				//Check if file exists already, if it does then only overwrite if dedupe is on
				if _, err := os.Stat(newPath); err == nil {
					// File exists, so abort if not allowed to overwrite
					if !lib.settings.Deduplicate {
						log.Debug().Str("oldPath", currentPath).Str("newPath", newPath).Msg("Not moving file as deduplication is disabled")
						return currentPath
					}
				}
				err = utilities.RenameFile(currentPath, newPath)
				if err != nil {
					log.Warn().Str("oldPath", currentPath).Str("newPath", newPath).Err(err).Msg("Moving file raised error")
				} else {
					log.Info().Str("oldPath", currentPath).Str("newPath", newPath).Msg("Done moving")
					//Push the folder to the cleanup path
					lib.folderCleanupRequests <- filepath.Dir(currentPath)
					return newPath
				}
			}
		}
	}
	return currentPath
}

// determineIdealFilePath is used for sorting files into the managed folder structure
func (lib *Library) determineIdealFilePath(info *formats.FileInfo, sourceFile string) (string, error) {
	//Using the template we want to create the new file path
	//Since go doesnt really do named args; using string replacements for now
	outputName := lib.settings.OrganisationFormat

	outputName = strings.ReplaceAll(outputName, FormatTitleIDSub, FormatTitleIDToString(info.TitleID))
	outputName = strings.ReplaceAll(outputName, FormatVersionSub, FormatVersionToString(info.Version))
	outputName = strings.ReplaceAll(outputName, FormatVersionDecSub, FormatVersionToHumanString(info.Version))
	outputName = strings.ReplaceAll(outputName, FormatTypeSub, info.Type.String())

	if strings.Contains(outputName, FormatNameSub) {
		//Have to lookup the name in the db
		gameTitle, err := lib.QueryGameTitleFromTitleID(info.TitleID)
		if err != nil {
			//Try and load the file name directly
			gameTitle = info.EmbeddedTitle
			if len(gameTitle) == 0 {
				//Check if its a DLC and we can ignore it
				return "", fmt.Errorf("unable to determine path as title lookup failed with - >%w< and the embedded Title was empty", err)
			}
		}
		gameTitle = utilities.CleanName(gameTitle)
		outputName = strings.ReplaceAll(outputName, FormatNameSub, gameTitle)
	}
	extension := filepath.Ext(sourceFile)
	extension = strings.ToLower(extension)
	outputName += extension
	outputName = path.Join(lib.settings.StorageFolder, outputName)
	outputName, err := filepath.Abs(outputName)
	return outputName, err

}

func (lib *Library) sortFileHandleRemoved(event *fileScanningInfo) {

	// Scan the list of known files and check if the path matches
	if oldPath, err := filepath.Abs(event.path); err == nil {
		log.Info().Str("path", oldPath).Msg("Delete event")
		for key, item := range lib.filesKnown {
			items := item.GetFiles()
			match := false
			for _, item := range items {
				if item.Path == oldPath {
					//This one is a match
					match = true
				} else if strings.HasPrefix(item.Path, oldPath) {
					match = true
				}
			}
			if match {
				//Dump the old record, requeue all files
				log.Info().Str("path", oldPath).Msg("Deleted path matched, rescanning")
				delete(lib.filesKnown, key)
				for _, item := range items {
					event := &fileScanningInfo{
						path: item.Path,
					}
					lib.fileMetaScanRequests <- event
				}
				return
			}
		}
	}
}
