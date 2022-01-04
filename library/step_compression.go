package library

import (
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

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
							}
							lib.fileOrganisationRequests <- event
						}
						if utilities.Exists(newpath) {
							//New file exists, put it through the scanner
							event := &fileScanningInfo{
								path:        newpath,
								isInLibrary: !lib.settings.ValidateCompressedFiles,
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
	cmd := exec.Command(cleanedParts[0], cleanedParts[1:]...)
	byteData, err := cmd.CombinedOutput()
	if err != nil {
		outputLog := string(byteData)
		log.Error().Err(err).Str("output", outputLog).Msg("NSZ compression failed")
		return err
	}
	return nil

}
