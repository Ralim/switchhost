package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

type scanRequest struct {
	path             string
	isEndOfStartScan bool
	isNotifierBased  bool
}

// Library manages the representation of the game files on disk + their metadata
type Library struct {
	//Privates
	keys       *keystore.Keystore
	settings   *settings.Settings
	filesKnown map[uint64]TitleOnDiskCollection
	//Organisation
	titledb *titledb.TitlesDB

	fileScanRequests      chan *scanRequest
	folderCleanupRequests chan string
	fileWatcher           *watcher.Watcher
}

func NewLibrary(titledb *titledb.TitlesDB, settings *settings.Settings) *Library {
	library := &Library{

		fileScanRequests:      make(chan *scanRequest, 32),
		folderCleanupRequests: make(chan string, 128),
		titledb:               titledb,
		settings:              settings,
		filesKnown:            make(map[uint64]TitleOnDiskCollection),
		fileWatcher:           watcher.New(),
	}

	library.fileWatcher.FilterOps(watcher.Create)
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
	//Start worker thread for handling file parsing
	go lib.fileScanningWorker()
	go lib.RunScan()
	go lib.cleanupFolderWorker()
	go lib.fileWatcher.Start(time.Minute)

	//Trivial map fro mwatcher into the pendings list
	go func() {
		for change := range lib.fileWatcher.Event {
			event := &scanRequest{
				path:             change.Path,
				isEndOfStartScan: false,
				isNotifierBased:  true,
			}
			lib.fileScanRequests <- event
		}
	}()
	return nil
}

// RunScan runs a scan of all "normal" scan folders
func (lib *Library) RunScan() {
	for _, folder := range lib.settings.GetAllScanFolders() {
		if err := lib.ScanFolder(folder); err == nil {
			lib.folderCleanupRequests <- folder
		}
		// Setup watch on folder for new files
		if err := lib.fileWatcher.AddRecursive(folder); err != nil {
			log.Error().Err(err).Msgf("Could not install watcher for folder %s", folder)
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
					log.Debug().Msgf("File scan requested for %s", path)
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
