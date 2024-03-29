package library

import (
	"fmt"
	"github.com/ralim/switchhost/versionsdb"
	"io"
	"os"
	"sync"

	"github.com/ralim/switchhost/formats"
	"github.com/ralim/switchhost/index"
	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

// This struct is used for all of the file ingest scanning path.
// The data in the struct is slowly filled in
type fileScanningInfo struct {
	path string // The path to inspect
	// If this file must be cleaned up by either sorting or delete (aka its an incoming file)
	// This means, if this fails at any step it must be deleted rather than ignored
	mustCleanupFile bool
	// Metadata parsed out of the raw file
	metadata *formats.FileInfo
	//Sent to notify of a deleted file that must be cleand up from the lib
	fileWasDeleted bool
	// Did this file come from the library folder (else, its upload + startup scan)
	isInLibrary bool
}

// Library manages the representation of the game files on disk + their metadata
type Library struct {
	FileIndex *index.Index

	//Privates
	keys      *keystore.Keystore
	settings  *settings.Settings
	titledb   *titledb.TitlesDB
	versiondb *versionsdb.VersionDB

	waitgroup *sync.WaitGroup
	//These channels are used for decoupling the workers for each state of the file import pipeline
	// `Scanner` -> `Metadata parser` -> `Validator` -> `Organiser` -> `Cleanup` -> `Compression`

	// 1. Scan requests to figure out metadata (or unparsable)
	fileMetaScanRequests chan *fileScanningInfo
	// 2. Once metadata is scanned, files are pushed to the validation queue, which can validate file hashes if desired (short-circuits if not)
	fileValidationScanRequests chan *fileScanningInfo
	// 3. Organiser, once a file is valid; it is orgnisationally checked to ensure correct fs location, and placed into the library
	fileOrganisationRequests chan *fileScanningInfo
	// 4. Now that the file has been organised; if it moved its old folder is scanned for cleanup
	folderCleanupRequests chan string
	// 5. Additionally, once a file is in the library, compression may be desired and thus it is passed here
	fileCompressionRequests chan *fileScanningInfo
	exit                    chan bool
	ui                      *termui.TermUI

	organisationLocking organisationLocks
}

func NewLibrary(titledb *titledb.TitlesDB, settings *settings.Settings, ui *termui.TermUI, versions *versionsdb.VersionDB) *Library {
	library := &Library{
		titledb:   titledb,
		settings:  settings,
		versiondb: versions,
		ui:        ui,
		keys:      nil,
		// Channels
		fileMetaScanRequests:       make(chan *fileScanningInfo, settings.QueueLength),
		fileValidationScanRequests: make(chan *fileScanningInfo, settings.QueueLength),
		fileOrganisationRequests:   make(chan *fileScanningInfo, settings.QueueLength),
		fileCompressionRequests:    make(chan *fileScanningInfo, settings.QueueLength),
		folderCleanupRequests:      make(chan string, settings.QueueLength),
		exit:                       make(chan bool, 10),
		FileIndex:                  index.NewIndex(titledb, settings),
		waitgroup:                  &sync.WaitGroup{},
		organisationLocking:        organisationLocks{},
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

// Start spawns internal workers and performs any non-trivial setup time tasks
func (lib *Library) Start() {
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

	// Internal states of the chain (except organisation) run multiple workers to utilise more cores
	// Process up to CPU count steps at once for each type
	// This will overschedule the tasks to run usually, but we are _super_ IO blocked so its usually OK
	for i := 0; i < lib.settings.GetCPUCount(); i++ {
		lib.waitgroup.Add(1)
		go lib.fileMetadataWorker()

		lib.waitgroup.Add(1)
		go lib.fileValidationWorker()

		// Start worker thread for handling file parsing
		lib.waitgroup.Add(1)
		go lib.fileorganisationWorker()
	}

	// Start worker for cleaning up empty folders
	lib.waitgroup.Add(1)
	go lib.cleanupFolderWorker()

	// Start worker for nsz compression
	lib.waitgroup.Add(1)
	go lib.compressionWorker()

	// Run first file scan in background
	lib.waitgroup.Add(1)
	go lib.RunScan()

}

func (lib *Library) Stop() {
	log.Info().Msg("Library closing")
	lib.exit <- true
	log.Info().Msg("Waiting")

	lib.waitgroup.Wait()
}

func (lib *Library) NotifyIncomingFile(path string) {
	log.Info().Str("path", path).Msg("Notified of uploaded file")
	event := &fileScanningInfo{
		path:            path,
		mustCleanupFile: true,
	}
	lib.fileMetaScanRequests <- event
}
