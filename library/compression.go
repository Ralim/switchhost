package library

import (
	"os/exec"
	"strings"

	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

// Compression handles compressing files using the existingnsz tooling
// It runs a single file compression at a time in the background

func (lib *Library) compressionWorker() {
	//Dequeue any requests off the queue and run the compression

	for request := range lib.fileCompressionRequests {
		//For each requested file, run it through NSZ and check output
		err := lib.NSZCompressFile(request)
		if err != nil {
			log.Err(err).Msg("NSZ compression failed")
		} else {
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
	return cmd.Run()

}