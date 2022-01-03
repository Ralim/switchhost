package library

import (
	"os"
	"path"
	"strings"

	"github.com/ralim/switchhost/formats"
	"github.com/ralim/switchhost/termui"
	"github.com/rs/zerolog/log"
)

func (lib *Library) fileValidationWorker() {
	defer lib.waitgroup.Done()
	defer log.Info().Msg("fileValidationWorker task exiting")
	var status *termui.TaskState
	if lib.ui != nil {
		status = lib.ui.RegisterTask("Validation")
		defer status.UpdateStatus("Exited")
		status.UpdateStatus("Idle")
	}

	if lib.keys == nil {
		log.Error().Msg("No keys are loaded, so file validations can't work.")
		return
	}

	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case event := <-lib.fileValidationScanRequests:
			requestedPath := event.path
			if status != nil {
				status.UpdateStatus(path.Base(event.path))
			}

			// This file has had its metadata parsed, so we want to validate integrity if desired
			// If it parses validation send it on, if not.. handle it
			shouldValidate := (lib.settings.ValidateLibrary && event.isInLibrary) || (lib.settings.ValidateNewFiles && !event.isInLibrary)

			if !shouldValidate || lib.validateFile(requestedPath) {
				//Validated, send onwards
				lib.fileOrganisationRequests <- event
			} else {
				if lib.settings.DeleteValidationFails || event.mustCleanupFile {
					log.Warn().Str("path", requestedPath).Msg("File failed valiation, deleting file")
					if err := os.Remove(requestedPath); err != nil {
						log.Error().Str("path", requestedPath).Msg("File failed valiation, tried deleting file, but it failed")
					}
				} else {
					log.Warn().Str("path", requestedPath).Str("name", event.metadata.Name).Str("title", event.metadata.EmbeddedTitle).Msg("File failed valiation, not putting in library")
				}
			}
			if status != nil {
				status.UpdateStatus("Idle")
			}

		}
	}
}

func (lib *Library) validateFile(filepath string) bool {
	//Returns false if file fails validation, true if good or uncertain

	ext := strings.ToLower(path.Ext(filepath))
	if len(ext) == 4 {

		if ext[0:3] == ".ns" {
			file, err := os.Open(filepath)
			if err != nil {
				return true
			}
			defer file.Close()
			if err := formats.ValidateNSPHash(lib.keys, lib.settings, file); err != nil {
				log.Warn().Str("path", filepath).Err(err).Msg("Failed validation")
				return false
			}
		} else if ext[0:3] == ".xc" {
			file, err := os.Open(filepath)
			if err != nil {
				return true
			}
			defer file.Close()
			if err := formats.ValidateXCIHash(lib.keys, lib.settings, file); err != nil {
				log.Warn().Str("path", filepath).Err(err).Msg("Failed validation")
				return false
			}

		} else {
			return true // can't validate
		}

	}
	return true

}
