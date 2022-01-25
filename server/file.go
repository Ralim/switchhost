package server

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ralim/switchhost/library/index"
	"github.com/ralim/switchhost/utilities"
)

var ErrInvalidPath error = errors.New("invalid path")

// Since we dont want to expose the file system to users, we provide virtual paths for all known tracked files

// Since all tracked files (should) have a unique TitleID/Version, this is used to generate the virtual path to the file
// These can then be reversed back into the real file path

// Given valid input, these two functions are each others inverse

func (server *Server) GenerateVirtualFilePath(file index.FileOnDiskRecord, hostNameToUse string, useHTTPS bool) string {
	ext := path.Ext(file.Path)
	ext = strings.ToLower(ext)
	fileFinalName := fmt.Sprintf("%s [%016X][v%d]%s", utilities.CleanName(file.Name), file.TitleID, file.Version, ext)
	base := fmt.Sprintf("/vfile/%d/%d/data.bin#%s", file.TitleID, file.Version, fileFinalName)
	if useHTTPS {
		base = "https://" + hostNameToUse + base
	} else {
		base = "http://" + hostNameToUse + base
	}
	return base
}
func (server *Server) LookupVirtualFilePath(path string) (uint64, uint32, error) {
	splits := strings.Split(path, "/")
	if len(splits) != 4 {
		return 0, 0, ErrInvalidPath
	}
	//Split out the two numbers
	titleID, err := strconv.ParseUint(splits[1], 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidPath
	}
	version, err := strconv.ParseUint(splits[2], 10, 32)
	if err != nil {
		return 0, 0, ErrInvalidPath
	}
	return titleID, uint32(version), nil
}

func (server *Server) getFileFromVirtualPath(path string) (io.ReadSeekCloser, string, int64, error) {
	titleID, version, err := server.LookupVirtualFilePath(path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("couldn't interpret path %s - %w", path, err)
	}
	//Otherwise we can now look up the actual on disk path to said file
	info, ok := server.library.FileIndex.GetFileRecord(titleID, version)
	if !ok {
		return nil, "", 0, fmt.Errorf("couldn't lookup path %s", path)
	}
	file, err := os.Open(info.Path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("couldn't lookup path %s", path)
	}
	_, filename := filepath.Split(info.Path)
	finfo, err := file.Stat()
	if err != nil {
		return nil, "", 0, fmt.Errorf("couldn't lookup path %s", path)
	}
	return file, filename, finfo.Size(), err
}
