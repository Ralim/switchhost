package library

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

// Compression handles compressing files using the existingnsz tooling
// It runs a single file compression at a time in the background

func (lib *Library) compressionWorker() {
	//Dequeue any requests off the queue and run the compression
	defer lib.waitgroup.Done()
	defer log.Info().Msg("Compression task exiting")
	for {
		select {
		case <-lib.exit:
			lib.exit <- true
			return
		case request := <-lib.fileCompressionRequests:
			//For each requested file, run it through NSZ and check output
			if utilities.Exists(request) {
				log.Info().Str("path", request).Msg("Starting compression")
				err := lib.NSZCompressFile(request)
				if err != nil {
					log.Err(err).Msg("NSZ compression failed")
					//Cleanup output if it made one
					if len(request) > 3 {
						newpath := request[0:len(request)-1] + "z"
						if utilities.Exists(newpath) {
							if err := os.Remove(newpath); err != nil {
								log.Error().Err(err).Msg("cleanup output from failing NSZ failed")
							}
						}
					}
				} else {
					log.Info().Str("path", request).Msg("Compression complete")
					// Check if the source file has been deleted
					// Check if the expected output file is made

					if !utilities.Exists(request) {
						//Source file has been deleted, notify library
						event := &scanRequest{
							path:             request,
							isEndOfStartScan: false,
							isNotifierBased:  true,
							fileRemoved:      true,
						}
						lib.fileScanRequests <- event
					}
					//Figure out the output file path
					newpath := request[0:len(request)-1] + "z"
					if utilities.Exists(newpath) {
						//Source file has been deleted, notify library
						event := &scanRequest{
							path:             newpath,
							isEndOfStartScan: false,
							isNotifierBased:  true,
							fileRemoved:      false,
						}
						lib.fileScanRequests <- event
					}
				}
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
