package library

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ralim/switchhost/formats"
	cnmt "github.com/ralim/switchhost/formats/CNMT"
	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

//Metadata workers pickup requests for files, and figure out the file meta info for the file

//

func (lib *Library) fileMetadataWorker() {
	defer lib.waitgroup.Done()
	defer log.Info().Msg("fileMetadataWorker task exiting")
	var status *termui.TaskState
	if lib.ui != nil {
		status = lib.ui.RegisterTask("Metadata")
		defer status.UpdateStatus("Exited")
		status.UpdateStatus("Idle")
	}

	//For now limited to having to use keys to read files, TODO: Regex the deets out of the file name
	if lib.keys == nil {
		log.Error().Msg("No keys are loaded, so file operations can't work.")
		return
	}

	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case event := <-lib.fileMetaScanRequests:
			// If the file can be parsed, update metadata and push along
			// Otherwise handle cleanup
			if status != nil {
				status.UpdateStatus(path.Base(event.path))
			}
			err := lib.setFileMeta(event)
			if err == nil {
				// File parsed well; so sent it to the next stage
				lib.fileValidationScanRequests <- event
			} else {
				//File cant be parsed
				if event.mustCleanupFile {
					os.Remove(event.path)
				}
			}
			if status != nil {
				status.UpdateStatus("Idle")
			}
		}
	}
}

func (lib *Library) setFileMeta(info *fileScanningInfo) error {
	requestedPath, err := filepath.Abs(info.path)
	if err != nil {
		return err
	}
	if !utilities.Exists(requestedPath) {
		return errors.New("not found")
	}
	info.path = requestedPath // store cleaned and checked path

	log.Debug().Str("path", requestedPath).Msg("Starting requested scan")
	fileInfo, err := lib.getFileInfo(requestedPath)
	if err != nil {
		return err
	}
	info.metadata = fileInfo
	return nil
}

// getFileInfo will return the parsed fileInfo if we know how to decode the file
func (lib *Library) getFileInfo(sourceFile string) (*formats.FileInfo, error) {

	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("could not parse file metadata for %s due to error %w when opening file", sourceFile, err)
	}
	info := formats.FileInfo{}

	ext := strings.ToLower(filepath.Ext(sourceFile))

	switch ext {
	case ".nsp":
		fallthrough
	case ".nsz":
		info, err = formats.ParseNSPToMetaData(lib.keys, lib.settings, file)
	case ".xci":
		fallthrough
	case ".xcz":
		info, err = formats.ParseXCIToMetaData(lib.keys, lib.settings, file)
	default:
		return &info, fmt.Errorf("not a valid file type - %s", sourceFile)
	}
	if err != nil {
		return nil, fmt.Errorf("could not parse file metadata for %s due to error %w during file parsing", sourceFile, err)
	}
	if len(info.EmbeddedTitle) == 0 && info.Type != cnmt.DLC {
		log.Warn().Str("file", sourceFile).Msg("Parsing embedded title failed")
	}
	if fileStat, err := file.Stat(); err == nil {
		info.Size = fileStat.Size()
	}
	return &info, nil
}
