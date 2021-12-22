package virtualftp

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
	ftpserver "goftp.io/server/v2"
)

var ErrNotAllowed error = errors.New("not allowed")

type FTPServer struct {
	server *ftpserver.Server
}

func CreateVirtualFTP(lib *library.Library, settings *settings.Settings) *FTPServer {
	driver := NewDriver(lib, settings)
	perm := ftpserver.NewSimplePerm("switch", "switch")
	opt := &ftpserver.Options{
		Name:           "switchhost",
		Driver:         driver,
		Port:           settings.FTPPort,
		Auth:           driver,
		Perm:           perm,
		WelcomeMessage: settings.ServerMOTD,
	}
	// start ftp server
	ftpServer, err := ftpserver.NewServer(opt)
	if err != nil {
		log.Error().Err(err).Msg("FTP server creation failed")
	}
	return &FTPServer{server: ftpServer}

}

func (ftp *FTPServer) Start() {
	err := ftp.server.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("FTP server start failed")
	}
}
func (ftp *FTPServer) Stop() {
	if ftp.server != nil {
		_ = ftp.server.Shutdown()
	}
}

// Driver for the ftp lib to remap the virtual index
type FTPDriver struct {
	library  *library.Library
	settings *settings.Settings
}

// NewDriver creates a new FTPDriver for the virtual FTP hosting
func NewDriver(lib *library.Library, settings *settings.Settings) *FTPDriver {
	return &FTPDriver{library: lib, settings: settings}
}

/*

Virtual FTP server
This does _NOT_ host the actual game folders directly
Instead this presents a "virtual" listing, akin to the way HTTP serves files

This will create folder paths of
	/<title name> [GameID]/...Files...
Which is relatively easy to browse

*/

func (driver *FTPDriver) dirPathToTitleID(path string) (uint64, error) {
	reg := regexp.MustCompile(`^/.+\[(\d+)\]$`)
	match := reg.FindStringSubmatch(path)
	if len(match) > 1 {
		titleID, err := strconv.ParseUint(match[1], 10, 64)
		return titleID, err
	}
	return 0, errors.New("couldnt parse")
}

func (driver *FTPDriver) getFakeFolderFileInfo(titleInfo library.FileOnDiskRecord) os.FileInfo {
	virtualpath := fmt.Sprintf("%s [%d]", utilities.CleanName(titleInfo.Name), titleInfo.TitleID)
	info := NewFakeFolder(virtualpath)
	return &info
}

// ListDir implements Driver
func (driver *FTPDriver) ListDir(ctx *ftpserver.Context, path string, callback func(os.FileInfo) error) error {
	if path == "/" {
		//Returning virtual folder of titles
		for _, titleInfo := range driver.library.ListTitleFiles() {
			//Generate title virtual path
			_ = callback(driver.getFakeFolderFileInfo(titleInfo))
		}
	} else {
		//Most likely a path to a folder of files
		if titleID, err := driver.dirPathToTitleID(path); err == nil {
			val, ok := driver.library.GetFilesForTitleID(titleID)
			if ok {
				//Now need to yield os info's for all of the underlying files
				for _, file := range val.GetFiles() {
					info, err := os.Stat(file.Path)
					if err == nil {
						fakefile := NewFakeFile(driver.getfakepathForRealFile(file), info)
						_ = callback(&fakefile)
					}
				}
			}
		}
	}
	return nil
}
func (driver *FTPDriver) getfakepathForRealFile(file library.FileOnDiskRecord) string {
	ext := path.Ext(file.Path)
	fileTitle := fmt.Sprintf("%s - [%d][%d]%s", file.Name, file.TitleID, file.Version, ext)
	return path.Join(fileTitle)
}
func (driver *FTPDriver) getRealFilePathFromVirtual(path string) (string, bool) {

	//Lookup the titleID to check against
	//Then just match paths :shrug:
	reg := regexp.MustCompile(`/.+\[(\d+)\]\[(\d+)\]\..+$`)
	match := reg.FindStringSubmatch(path)
	if len(match) != 3 {
		return "", false
	}
	titleID, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		return "", false
	}
	version, err := strconv.ParseUint(match[2], 10, 32)
	if err != nil {
		return "", false
	}
	value, ok := driver.library.GetFileRecord(titleID, uint32(version))
	if !ok {
		return "", false
	}
	return value.Path, true
}

func (driver *FTPDriver) Stat(ctx *ftpserver.Context, path string) (os.FileInfo, error) {
	if path == "/" {
		info := NewFakeFolder(path)
		return &info, nil
	}
	if titleid, err := driver.dirPathToTitleID(path); err == nil {
		//This is a file folder, generate faux info
		if titleInfo, ok := driver.library.GetFilesForTitleID(titleid); ok {
			files := titleInfo.GetFiles()
			if len(files) > 0 {
				return driver.getFakeFolderFileInfo(files[0]), err
			}
		}
	}
	realPath, ok := driver.getRealFilePathFromVirtual(path)
	if !ok {
		return nil, errors.New("cant find it")
	}
	fileInfo, err := os.Stat(realPath)
	if err != nil {
		return fileInfo, err
	}
	fakeFile := NewFakeFile(path, fileInfo)
	return &fakeFile, nil
}

func (driver *FTPDriver) GetFile(ctx *ftpserver.Context, path string, offset int64) (int64, io.ReadCloser, error) {
	realPath, ok := driver.getRealFilePathFromVirtual(path)
	if !ok {
		return 0, nil, errors.New("cant find file")
	}
	f, err := os.Open(realPath)
	if err != nil {
		return 0, nil, fmt.Errorf("reading file from offset failed open - %w", err)
	}

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf("reading file from offset failed stat - %w", err)
	}

	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, nil, fmt.Errorf("reading file from offset failed seek - %w", err)
	}
	username, ok := ctx.Sess.Data["username"].(string)
	if !ok {
		username = "unknown"
	}
	log.Info().Str("user", username).Str("path", path).Msg("Started FTP stream")
	return info.Size() - offset, f, nil
}

func (driver *FTPDriver) PutFile(ctx *ftpserver.Context, destPath string, data io.Reader, offset int64) (int64, error) {
	if !driver.settings.UploadingAllowed {
		return 0, ErrNotAllowed
	}
	if allowed, ok := ctx.Sess.Data["uploadAllowed"]; !ok || !(allowed.(bool)) {
		return 0, ErrNotAllowed
	}

	//Only allow uploads to resume at 0 or no resume at all
	if !((offset == 0) || (offset == -1)) {
		return 0, errors.New("no partial uploads")
	}
	//File uploads are filtered by file extension, and anything that isnt a NS? or XC? is rejected
	extension := strings.ToLower(path.Ext(destPath))
	switch extension {
	case ".nsp":
	case ".nsz":
	case ".xci":
	case ".xcz":
	default:
		return 0, errors.New("bad file type")
	}
	// We upload the file to a location in tmp during the upload and then sort or delete
	tmpFile, err := ioutil.TempFile(os.TempDir(), "switchhost-upload-*"+extension)
	if err != nil {
		log.Error().Err(err).Msg("Failed creating temp file for upload")
	}
	// Now drain all the ftp upload into the temp file
	bytesSaved, err := io.Copy(tmpFile, data)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Warn().Err(err).Msg("Error during copying data in FTP upload")
		return 0, err
	}
	//Notify the library code to scan this file and sort it or delete it
	tmpFile.Close()

	driver.library.NotifyIncomingFile(tmpFile.Name())

	return bytesSaved, nil
}

func (driver *FTPDriver) DeleteDir(ctx *ftpserver.Context, path string) error {
	return ErrNotAllowed
}

func (driver *FTPDriver) DeleteFile(ctx *ftpserver.Context, path string) error {
	return ErrNotAllowed
}

func (driver *FTPDriver) Rename(ctx *ftpserver.Context, fromPath string, toPath string) error {
	return ErrNotAllowed
}

func (driver *FTPDriver) MakeDir(ctx *ftpserver.Context, path string) error {
	//Ignored as these are uploads that we are gonna strip the path from
	return nil
}

func (driver *FTPDriver) CheckPasswd(ctx *ftpserver.Context, username string, password string) (bool, error) {
	ctx.Sess.Data["uploadAllowed"] = false
	match := false
	for _, user := range driver.settings.Users {
		if subtle.ConstantTimeCompare([]byte(user.Username), []byte(username)) == 1 && subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) == 1 {
			if user.AllowFTP {
				ctx.Sess.Data["uploadAllowed"] = user.AllowUpload
				ctx.Sess.Data["username"] = username
				match = true
			}
		}
	}
	// If anon is enabled, anyone can download, but upload is controlled by user accounts
	if driver.settings.AllowAnonFTP {
		return true, nil
	}

	return match, nil
}
