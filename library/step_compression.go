package library

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

var ErrCompressionTimeout = errors.New("Compression timed out")

// Compression handles compressing files using the existingnsz tooling
// It runs a single file compression at a time in the background

func (lib *Library) compressionWorker() {
	//Dequeue any requests off the queue and run the compression
	defer lib.waitgroup.Done()
	defer log.Info().Msg("Compression task exiting")
	var status *termui.TaskState
	if lib.ui != nil {
		status = lib.ui.RegisterTask("Compression")
		defer status.UpdateStatus("Exited")
		status.UpdateStatus("Idle")
	}
	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case request := <-lib.fileCompressionRequests:
			//For each requested file, run it through NSZ and check output
			if request != nil && utilities.Exists(request.path) {
				if len(request.path) > 3 {
					if status != nil {
						status.UpdateStatus(path.Base(request.path))
					}
					newpath := request.path[0:len(request.path)-1] + "z"
					log.Info().Str("path", request.path).Msg("Starting compression")
					err := lib.NSZCompressFile(request.path)
					if err != nil {
						log.Err(err).Msg("NSZ compression failed")
						//Cleanup output if it made one
						if utilities.Exists(newpath) {
							if err := os.Remove(newpath); err != nil {
								log.Error().Err(err).Msg("cleanup output from failing NSZ failed")
							}
						}
					} else {
						log.Info().Str("path", request.path).Msg("Compression complete")
						// Check if the source file has been deleted
						// Check if the expected output file is made

						if !utilities.Exists(request.path) {
							//Source file has been deleted, notify library
							event := &fileScanningInfo{
								path:           request.path,
								fileWasDeleted: true,
								metadata:       request.metadata,
							}
							lib.fileOrganisationRequests <- event
						}
						if utilities.Exists(newpath) {
							//New file exists, put it through the scanner
							event := &fileScanningInfo{
								path:        newpath,
								isInLibrary: !lib.settings.ValidateCompressedFiles,
								metadata:    request.metadata,
							}
							lib.fileMetaScanRequests <- event
						}
					}
				}
			}
			if status != nil {
				status.UpdateStatus("Idle")
			}
		}
	}
}
func (lib *Library) NSZCompressFile(path string) error {
	//Call out to external tool using the user provided base string
	parts := strings.Split(lib.settings.NSZCommandLine, " ")
	cleanedParts := []string{}
	for _, p := range parts {
		if len(p) > 0 {
			cleanedParts = append(cleanedParts, p)
		}
	}
	cleanedParts = append(cleanedParts, path)

	timeoutValue := lib.settings.CompressionTimeoutMins
	if timeoutValue == 0 {
		timeoutValue = 60 // If not set, default to an hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutValue)*time.Minute)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	cmd := exec.CommandContext(ctx, cleanedParts[0], cleanedParts[1:]...)

	if ctx.Err() == context.DeadlineExceeded {
		log.Error().Msg("Compression timed out and was terminated.")
		return ErrCompressionTimeout
	}

	byteData, err := cmd.CombinedOutput()
	if err != nil {
		outputLog := string(byteData)
		log.Error().Err(err).Str("output", outputLog).Msg("NSZ compression failed")
		return err
	}
	return nil

}
