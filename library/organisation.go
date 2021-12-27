package library

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/formats"
	cnmt "github.com/ralim/switchhost/formats/CNMT"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

const (
	FormatNameSub       = "{TitleName}"
	FormatTitleIDSub    = "{TitleID}"
	FormatVersionSub    = "{Version}"
	FormatVersionDecSub = "{VersionDec}"
	FormatTypeSub       = "{Type}"
)

func (lib *Library) fileScanningWorker() {
	defer lib.waitgroup.Done()
	defer log.Info().Msg("Organisation task exiting")
	// This worker thread listens on the channel for notification of any files that should be checked
	// Single threaded to prevent any race issues

	// Theory of operation
	// 1. Parse file to find new name
	// 2. If name is different, move to new name
	// 3. Update our in ram db to the file existance :)
	// 4. If the containing folder is now empty, remove it

	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case event := <-lib.fileScanRequests:
			if event.isEndOfStartScan {
				log.Info().Msg("Initial startup scan is complete")
			} else if event.fileRemoved {
				lib.sortFileHandleRemoved(event)
			} else {
				lib.sortFileHandleScan(event)
			}

		}
	}

}
func (lib *Library) sortFileHandleRemoved(event *scanRequest) {

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
					event := &scanRequest{
						path:             item.Path,
						isEndOfStartScan: false,
						isNotifierBased:  true,
					}
					lib.fileScanRequests <- event
				}
				return
			}
		}
	}
}

func (lib *Library) sortFileHandleScan(event *scanRequest) {
	log.Debug().Str("path", event.path).Bool("isNotifier", event.isNotifierBased).Msg("Scan request")
	if event.mustCleanupFile {
		//We dont care if this fails because file doesnt exist, that just means it was cleaned up
		defer os.Remove(event.path)
	}
	if requestedPath, err := filepath.Abs(event.path); err == nil {
		//Dont bother wasting time on files that no longer exist
		if !utilities.Exists(requestedPath) {
			return
		}
		if !lib.validateFile(requestedPath) {

			if lib.settings.DeleteValidationFails {
				log.Warn().Str("path", requestedPath).Msg("File failed valiation, deleting file")
				if err := os.Remove(requestedPath); err != nil {
					log.Error().Str("path", requestedPath).Msg("File failed valiation, deleting file, but it failed")
				}
			} else {
				log.Warn().Str("path", requestedPath).Msg("File failed valiation, not putting in library")
			}
			return
		}
		//For now limited to having to use keys to read files, TODO: Regex the deets out of the file name
		if lib.keys != nil {
			log.Debug().Str("path", requestedPath).Msg("Starting requested scan")
			info, err := lib.getFileInfo(requestedPath)
			if err != nil {
				log.Warn().Err(err).Str("path", requestedPath).Msg("could not determine sorted path")
			} else {
				fileResultingPath := lib.sortFileIfApplicable(info, requestedPath, event.mustCleanupFile)

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
				lib.postFileAddToLibraryHooks(record)
			}
			log.Debug().Str("path", requestedPath).Msg("Finished scan")
		}
	}
}

func (lib *Library) postFileAddToLibraryHooks(file *FileOnDiskRecord) {
	//Dispatch any post hooks
	if lib.settings.CompressionEnabled {
		extension := strings.ToLower(path.Ext(file.Path))
		if len(extension) == 4 {
			if extension[3] != 'z' {
				//File might be compressable, send it off
				log.Info().Str("path", file.Path).Msg("Adding to compression list")
				lib.fileCompressionRequests <- file.Path
			}
		}
	}
}

func (lib *Library) validateFile(filepath string) bool {
	//Returns false if file fails validation, true if good or uncertain

	if len(lib.settings.HactoolPath) == 0 {
		return true // cant check
	}
	ext := strings.ToLower(path.Ext(filepath))
	if len(ext) == 4 {

		args := []string{"-t"}
		if ext[0:3] == ".ns" {
			args = append(args, "pfs0")
		} else if ext[0:3] == ".xc" {
			args = append(args, "xci")
		} else {
			return true // can't validate
		}
		args = append(args, filepath)
		cmd := exec.Command(lib.settings.HactoolPath, args...)
		byteData, err := cmd.CombinedOutput()
		if err != nil {
			outputLog := string(byteData)
			log.Error().Err(err).Str("path", filepath).Str("output", outputLog).Msg("File validation failed")
			return false
		}
		log.Debug().Str("path", filepath).Msg("File validation ok")
		return true

	}
	return true

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

// getFileInfo will return the parsed fileInfo if we know how to decode the file
func (lib *Library) getFileInfo(sourceFile string) (*formats.FileInfo, error) {

	if lib.keys == nil {
		return nil, errors.New("can't sort files without a loaded key file")
	}
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("could not determine sorted path for %s due to error %w when opening file", sourceFile, err)
	}
	info := formats.FileInfo{}

	ext := filepath.Ext(sourceFile)
	ext = strings.ToUpper(ext)

	switch ext {
	case ".NSP":
		info, err = formats.ParseNSPToMetaData(lib.keys, lib.settings, file)
	case ".NSZ":
		info, err = formats.ParseNSPToMetaData(lib.keys, lib.settings, file)
	case ".XCI":
		info, err = formats.ParseXCIToMetaData(lib.keys, lib.settings, file)
	case ".XCZ":
		info, err = formats.ParseXCIToMetaData(lib.keys, lib.settings, file)
	default:
		return &info, fmt.Errorf("not a valid file type - %s", sourceFile)
	}
	if err != nil {
		return nil, fmt.Errorf("could not determine sorted path for %s due to error %w during file parsing", sourceFile, err)
	}
	if len(info.EmbeddedTitle) == 0 && info.Type != cnmt.DLC {
		log.Warn().Str("file", sourceFile).Msg("Parsing embedded title failed")
	}
	fileStat, err := file.Stat()
	if err == nil {
		info.Size = fileStat.Size()
	}
	return &info, nil
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

/*************** Below are small formatting helpers ***************/
func FormatTitleIDToString(titleID uint64) string {
	//Format titleID out as a fixed with hex string
	return fmt.Sprintf("%016X", titleID)
}

func FormatVersionToString(version uint32) string {
	//Format as decimal with a v prefix
	return fmt.Sprintf("v%d", version)
}

func FormatVersionToHumanString(version uint32) string {
	if version == 0 {
		return "" //This is base game, no point
	}
	/*
		https://switchbrew.org/wiki/Title_list
			Decimal versions use the format:

			Bit31-Bit26: Major
			Bit25-Bit20: Minor
			Bit19-Bit16: Micro
			Bit15-Bit0: Bugfix

		Dont know if games use this exact format, but ok for now :shrug:
		Using leading zero suppression to make things a little easier to read
	*/
	major := version >> 26
	minor := version >> 20 & 0b111111
	micro := version >> 16 & 0b1111
	bugfix := version & 0xFFFF
	if major != 0 {
		return fmt.Sprintf("v%d.%d.%d.%d", major, minor, micro, bugfix)
	} else if minor != 0 {
		return fmt.Sprintf("v%d.%d.%d", minor, micro, bugfix)
	} else if micro != 0 {
		return fmt.Sprintf("v%d.%d", micro, bugfix)
	}
	return fmt.Sprintf("v%d", bugfix)
}
