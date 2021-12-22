package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

type scanRequest struct {
	path             string // The path to inspect
	isEndOfStartScan bool   // Marker just used so that we can print the nice text to confirm inital scanning is done
	isNotifierBased  bool   // did this come from the notifier or from a scan
	fileRemoved      bool   // If file event is because file was removed
	mustCleanupFile  bool   // If this file must be cleaned up by either sorting or delete (aka its an incoming file)
}

// Library manages the representation of the game files on disk + their metadata
type Library struct {
	//Privates
	keys     *keystore.Keystore
	settings *settings.Settings
	titledb  *titledb.TitlesDB

	filesKnown map[uint64]TitleOnDiskCollection

	waitgroup               *sync.WaitGroup
	fileScanRequests        chan *scanRequest
	folderCleanupRequests   chan string
	fileCompressionRequests chan string
	exit                    chan bool
}

func NewLibrary(titledb *titledb.TitlesDB, settings *settings.Settings) *Library {
	library := &Library{
		titledb:  titledb,
		settings: settings,
		// Channels
		fileScanRequests:        make(chan *scanRequest, 256),
		folderCleanupRequests:   make(chan string, 256),
		fileCompressionRequests: make(chan string, 256),
		exit:                    make(chan bool, 10),
		keys:                    nil,
		filesKnown:              make(map[uint64]TitleOnDiskCollection),
		// Internal objects
		waitgroup: &sync.WaitGroup{},
	}

	return library

}

func (lib *Library) LoadKeys(keysDBReader io.Reader) error {
	if keysDBReader != nil {
		//Attempt to load the keys db
		//Do this blocking as its fast and required for all other steps
		store, err := keystore.NewKeystore(keysDBReader)
		if err != nil {
			return err
		}
		lib.keys = store
	}
	return nil
}

//Start spawns internal workers and performs any non-trivial setup time tasks
func (lib *Library) Start() error {
	//Check output folder exists if sorting enabled
	if lib.settings.EnableSorting {
		if _, err := os.Stat(lib.settings.StorageFolder); os.IsNotExist(err) {
			if err := os.Mkdir(lib.settings.StorageFolder, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't create storage folder. Sorting will fail, so disabling")
				lib.settings.EnableSorting = false
				lib.settings.Save()
			}
		}

	}
	// Start worker thread for handling file parsing
	lib.waitgroup.Add(1)
	go lib.fileScanningWorker()
	// Run first file scan in background
	lib.waitgroup.Add(1)
	go lib.RunScan()
	// Start worker for cleaning up empty folders
	lib.waitgroup.Add(1)
	go lib.cleanupFolderWorker()
	// Start worker for nsz compression
	lib.waitgroup.Add(1)
	go lib.compressionWorker()

	return nil
}

func (lib *Library) Stop() {
	log.Info().Msg("Library closing")
	lib.exit <- true
	lib.waitgroup.Wait()
}

// RunScan runs a scan of all "normal" scan folders
func (lib *Library) RunScan() {
	defer lib.waitgroup.Done()
	for _, folder := range lib.settings.GetAllScanFolders() {
		if err := lib.ScanFolder(folder); err == nil {
			lib.folderCleanupRequests <- folder
		}

	}
	//end marker to allow indication to users that scan is done
	event := &scanRequest{
		path:             "",
		isEndOfStartScan: true,
		isNotifierBased:  false,
	}
	lib.fileScanRequests <- event
}

func (lib *Library) NotifyIncomingFile(path string) {
	log.Info().Str("path", path).Msg("Notified of uploaded file")
	event := &scanRequest{
		path:             path,
		isEndOfStartScan: false,
		isNotifierBased:  false,
		fileRemoved:      false,
		mustCleanupFile:  true,
	}
	lib.fileScanRequests <- event
}

//ScanFolder recursively scans the provied folder and feeds it to the organisation queue
func (lib *Library) ScanFolder(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil {

			if !info.IsDir() {

				ext := filepath.Ext(path)
				ext = strings.ToUpper(ext)
				shouldScan := false
				switch ext {
				case ".NSP":
					shouldScan = true
				case ".NSZ":
					shouldScan = true
				case ".XCI":
					shouldScan = true
				case ".XCZ":
					shouldScan = true
				}
				if shouldScan {
					//This is a file, so push it to the queue
					log.Debug().Str("path", path).Msg("File scan requested")
					event := &scanRequest{
						path:             path,
						isEndOfStartScan: false,
						isNotifierBased:  false,
					}
					lib.fileScanRequests <- event
				}
			}
		}
		return nil
	})
}
