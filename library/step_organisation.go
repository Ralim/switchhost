package library

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/formats"
	"github.com/ralim/switchhost/index"
	"github.com/ralim/switchhost/termui"
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
	var status *termui.TaskState
	if lib.ui != nil {
		status = lib.ui.RegisterTask("Organisation")
		defer status.UpdateStatus("Exited")
		status.UpdateStatus("Idle")
	}

	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case event := <-lib.fileOrganisationRequests:
			lib.organisationEventHandler(event, status)
			lib.updateTotals()
		}
	}
}

func (lib *Library) updateTotals() {
	if lib.ui != nil {
		stats := lib.FileIndex.GetStats()
		lib.ui.Statistics = &stats
	}
}

func (lib *Library) organisationEventHandler(event *fileScanningInfo, status *termui.TaskState) {
	if event.metadata == nil {
		log.Error().Str("path", event.path).Msg("BUG: nil metadata in organisation")
		return
	}
	lib.organisationLocking.Lock(event.metadata.TitleID)
	defer lib.organisationLocking.Unlock(event.metadata.TitleID)

	fileShortName := path.Base(event.path)
	if event.fileWasDeleted {
		if status != nil {
			status.UpdateStatus(fmt.Sprintf("Handling Delete of %s", fileShortName))
		}
		lib.FileIndex.RemoveFile(event.path)
	} else {
		info := event.metadata
		if status != nil {
			status.UpdateStatus(fmt.Sprintf("Sorting %s (%s)", fileShortName, info.EmbeddedTitle))
		}
		fileResultingPath := lib.sortFileIfApplicable(info, event.path, event.mustCleanupFile)
		if status != nil {
			status.UpdateStatus(fmt.Sprintf("Processing %s", fileShortName))
		}
		//Add to our repo, moved or not
		record := &index.FileOnDiskRecord{
			Path:    fileResultingPath,
			TitleID: info.TitleID,
			Version: info.Version,
			Name:    info.EmbeddedTitle,
			Size:    info.Size,
		}
		if gameTitle, err := lib.QueryGameTitleFromTitleID(info.TitleID); err == nil {
			record.Name = gameTitle
		}
		if lib.ui != nil {
			defer lib.ui.Statistics.Redraw()
		}
		lib.FileIndex.AddFileRecord(record)
		event.path = fileResultingPath
		lib.postFileAddToLibraryHooks(event)
		if status != nil {
			status.UpdateStatus("Idle")
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
