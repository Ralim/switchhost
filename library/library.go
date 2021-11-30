package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ralim/switchhost/keystore"
	"github.com/ralim/switchhost/settings"
	titledb "github.com/ralim/switchhost/titledb"
)

// Library manages the representation of the game files on disk + their metadata

type Library struct {
	//Privates
	keys       *keystore.Keystore
	settings   *settings.Settings
	filesKnown map[uint64]TitleOnDiskCollection
	//Organisation
	titledb *titledb.TitlesDB

	fileScanRequests      chan string
	folderCleanupRequests chan string
}

func NewLibrary(titledb *titledb.TitlesDB, settings *settings.Settings) *Library {
	library := &Library{

		fileScanRequests:      make(chan string, 128),
		folderCleanupRequests: make(chan string, 128),
		titledb:               titledb,
		settings:              settings,
		filesKnown:            make(map[uint64]TitleOnDiskCollection),
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
				fmt.Fprintf(os.Stderr, "Couldn't create storage folder. Sorting will fail, so disabling\n")
				lib.settings.EnableSorting = false
				lib.settings.Save()
			}
		}

	}
	//Start worker thread for handling file parsing
	go lib.fileScanningWorker()
	go lib.RunScan()
	go lib.cleanupFolderWorker()
	return nil
}

// RunScan runs a scan of all "normal" scan folders
func (lib *Library) RunScan() {
	for _, folder := range lib.settings.GetAllScanFolders() {
		if err := lib.ScanFolder(folder); err == nil {
			lib.folderCleanupRequests <- folder
		}
	}
}

//ScanFolder recursively scans the provied folder and feeds it to the organisation queue
func (lib *Library) ScanFolder(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil {

			if !info.IsDir() {
				//This is a file, so push it to the queue
				fmt.Printf("File scan requested for %s\r\n", path)
				lib.fileScanRequests <- path
			}
		}
		return nil
	})
}
