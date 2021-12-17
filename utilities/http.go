package utilities

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/rs/zerolog/log"
)

func DownloadFileWithVersioning(fileURL, folder string) (string, error) {

	//Look for <filename>.etag for the etag
	_, fileName := path.Split(fileURL)
	outputFile := path.Join(folder, fileName)
	outputETagFile := outputFile + ".etag"
	existingETag := ""
	if content, err := ioutil.ReadFile(outputETagFile); err == nil {
		existingETag = string(content)
	}

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("cant request %s, newrequest threw -> %w", fileURL, err)
	}
	if len(existingETag) > 0 {
		req.Header.Add("If-None-Match", existingETag)
	}
	client := http.DefaultClient
	response, err := client.Do(req)

	if err != nil {
		log.Warn().Msgf("Request for file %s failed with %v, continuing anyway", fileURL, err)
	}
	if response.StatusCode == 304 {
		//Not modified, no-op
		return outputFile, nil
	} else if response.StatusCode != 200 {
		return "", fmt.Errorf("couldn't download file %s -> %d", fileURL, response.StatusCode)
	}
	//Otherwise 200, so save etag and the file
	etag := response.Header.Get("ETag")
	err = os.WriteFile(outputETagFile, []byte(etag), 0666)
	//We dont bubble up etag errors as non-essential
	if err != nil {
		log.Warn().Msgf("Saving ETag for file %s failed with %v, continuing anyway", fileURL, err)
	}

	//Create output file, truncates existing
	f, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("couldn't download file, writing to file %s failed; url: %s -> %w", outputFile, fileURL, err)
	}
	defer f.Close()
	_, err = io.Copy(f, response.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't download file, writing to file %s failed; url: %s -> %w", outputFile, fileURL, err)
	}
	return outputFile, nil
}
