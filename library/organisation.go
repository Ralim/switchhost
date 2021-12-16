package library

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/formats"
	"github.com/ralim/switchhost/utilities"
)

const (
	FormatNameSub       = "{TitleName}"
	FormatTitleIDSub    = "{TitleID}"
	FormatVersionSub    = "{Version}"
	FormatVersionDecSub = "{VersionDec}"
	FormatTypeSub       = "{Type}"
)

func (lib *Library) fileScanningWorker() {
	// This worker thread listens on the channel for notification of any files that should be checked
	// Single threaded to prevent any race issues

	// Theory of operation
	// 1. Parse file to find new name
	// 2. If name is different, move to new name
	// 3. Update our in ram db to the file existance :)
	// 4. If the containing folder is now empty, remove it

	for requestedPath := range lib.fileScanRequests {
		if requestedPath, err := filepath.Abs(requestedPath); err == nil {

			fmt.Printf("Starting requested scan of %s\r\n", requestedPath)
			info, err := lib.getFileInfo(requestedPath)
			if err != nil {
				fmt.Printf("could not determine sorted path for %s due to error %v during file parsing", requestedPath, err)
			} else {
				fileResultingPath := lib.sortFileIfApplicable(info, requestedPath)

				//Add to our repo, moved or not
				record := &FileOnDiskRecord{
					Path:    fileResultingPath,
					TitleID: info.TitleID,
					Version: info.Version,
					Name:    info.EmbeddedTitle,
					Size:    info.Size,
				}
				gameTitle, err := lib.QueryGameTitleFromTitleID(info.TitleID)
				if err == nil {
					record.Name = gameTitle
				}

				lib.AddFileRecord(record)
			}
			fmt.Printf("Finished scan of %s\r\n", requestedPath)
		}
	}
}

// sortFileIfApplicable; if sorting is on, attempts to sort the file to the new path if its different.
// If sorting is turned off, or if the sorting fails for one reason or another, just returns the source path
// If the file is moved, it returns the updated path
// If the file is moved, it will also notify the cleanup handler to go scan if the folder needs cleanup
func (lib *Library) sortFileIfApplicable(infoInfo *formats.FileInfo, currentPath string) string {

	// If sorting is off, no-op
	if !lib.settings.EnableSorting {
		return currentPath
	}
	newPath, err := lib.determineIdealFilePath(infoInfo, currentPath)
	if err != nil {
		fmt.Printf("Cant sort file %s due to error %v\n", currentPath, err)
		return currentPath
	}
	if err == nil {
		if newPath != currentPath {
			fmt.Printf("Moving file %s to %s\n", currentPath, newPath)
			err := os.MkdirAll(path.Dir(newPath), 0755)
			if err != nil {
				fmt.Printf("Could not move %s to %s, due to err %v\n", currentPath, newPath, err)
			} else {
				err = utilities.RenameFile(currentPath, newPath)
				if err != nil {
					fmt.Printf("Could not move %s to %s, due to err %v\n", currentPath, newPath, err)
				} else {
					fmt.Println("Done")
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
	}
	if err != nil {
		return nil, fmt.Errorf("could not determine sorted path for %s due to error %w during file parsing", sourceFile, err)
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
		outputName = strings.ReplaceAll(outputName, FormatNameSub, gameTitle)
	}
	extension := filepath.Ext(sourceFile)
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
