package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/radovskyb/watcher"
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
	waitgroupOrganiser      *sync.WaitGroup
	fileScanRequests        chan *scanRequest
	folderCleanupRequests   chan string
	fileCompressionRequests chan string
	exitRequest             chan bool
	fileWatcher             *watcher.Watcher
}

func NewLibrary(titledb *titledb.TitlesDB, settings *settings.Settings) *Library {
	library := &Library{
		titledb:  titledb,
		settings: settings,
		// Channels
		fileScanRequests:        make(chan *scanRequest, 32),
		folderCleanupRequests:   make(chan string, 128),
		fileCompressionRequests: make(chan string, 128),
		exitRequest:             make(chan bool),
		filesKnown:              make(map[uint64]TitleOnDiskCollection),
		// Internal objects
		fileWatcher:        watcher.New(),
		waitgroup:          &sync.WaitGroup{},
		waitgroupOrganiser: &sync.WaitGroup{},
	}

	library.fileWatcher.FilterOps(watcher.Create, watcher.Move, watcher.Remove, watcher.Rename)

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
	lib.waitgroupOrganiser.Add(1)
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
	// Start worker that manages files being deleted
	lib.waitgroup.Add(1)
	go func() {
		defer lib.waitgroup.Done()
		if err := lib.fileWatcher.Start(time.Minute); err != nil {
			log.Warn().Err(err).Msg("File watcher could not be started")
		}

	}()
	//Trivial map from watcher into the pendings list
	lib.waitgroup.Add(1)
	go lib.fileWatcherWorker()
	return nil
}

func (lib *Library) Stop() {
	log.Info().Msg("Library closing")
	//Order matters here a bit since we have a mild circular loop around the central organiser
	// We want to stop (a) All scanning and (b) compression and cleanup _first_
	// Then wind down the main organiser thread

	lib.fileWatcher.Close()            // Stops any new file changes coming in, and makes its two goroutines exit
	close(lib.fileCompressionRequests) // causes compression to pack up
	close(lib.folderCleanupRequests)   // Will cause cleanup to exit

	//Compression _may_ send results back to the organiser, so we want to wait for compression to finish _before_ we stop organiser
	lib.waitgroup.Wait()

	close(lib.fileScanRequests)
	lib.waitgroupOrganiser.Wait()
}
func (lib *Library) fileWatcherWorker() {
	defer lib.waitgroup.Done()
	for {
		select {
		case change := <-lib.fileWatcher.Event:
			log.Info().Str("path", change.Path).Str("event", change.Name()).Msg("Watcher event")
			if change.IsDir() {
				if change.Op == watcher.Remove || change.Op == watcher.Move {
					lib.folderCleanupRequests <- change.Path
					event := &scanRequest{
						path:             change.Path,
						isEndOfStartScan: false,
						isNotifierBased:  true,
						fileRemoved:      true,
					}
					lib.fileScanRequests <- event

				} else {
					if err := lib.ScanFolder(change.Path); err == nil {
						lib.folderCleanupRequests <- change.Path
					}
				}
			} else {
				event := &scanRequest{
					path:             change.Path,
					isEndOfStartScan: false,
					isNotifierBased:  true,
					fileRemoved:      false,
				}
				switch change.Op {
				case watcher.Create:
				case watcher.Move:
				case watcher.Remove:
					event.fileRemoved = true
				}
				lib.fileScanRequests <- event
			}
		case err := <-lib.fileWatcher.Error:
			log.Error().Err(err)
		case <-lib.fileWatcher.Closed:
			return
		}
	}
}

// RunScan runs a scan of all "normal" scan folders
func (lib *Library) RunScan() {
	defer lib.waitgroup.Done()
	for _, folder := range lib.settings.GetAllScanFolders() {
		if err := lib.ScanFolder(folder); err == nil {
			lib.folderCleanupRequests <- folder
		}
		// Setup watch on folder for new files
		if err := lib.fileWatcher.AddRecursive(folder); err != nil {
			log.Error().Err(err).Str("folder", folder).Msg("Could not install watcher")
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
