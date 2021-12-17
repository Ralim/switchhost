package titledb

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"

	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
)

/*

This is not actualy a _DB_ but its a close enough name for its intended use case.

For data storage there are a few things desired

* Lookup details from the titledb file
* Index of all files on disk

To keep things fairly fast these are split into:
* List of all id's on disk and their paths
* Key-Value store of titleID -> description structure

*/

// This is the stored set of data from the titledb json files
// We dont parse and store everything
type TitleDBEntry struct {
	StringID       string   `json:"id"`
	Name           string   `json:"name"`
	ReleaseDate    int      `json:"releaseDate"`
	NumPlayers     int      `json:"numberOfPlayers"`
	IconURL        string   `json:"iconUrl"`
	BannerURL      string   `json:"bannerUrl"`
	ScreenshotURLs []string `json:"screenshots"`
}

type TitlesDB struct {
	//Public

	//Private
	entriesLock sync.RWMutex
	entries     map[uint64]TitleDBEntry
	settings    *settings.Settings
}

func CreateTitlesDB(settings *settings.Settings) *TitlesDB {
	return &TitlesDB{
		entries:  make(map[uint64]TitleDBEntry),
		settings: settings,
	}
}

// UpdateTitlesDB will sync latest titlesdb, then update the internal memory state
func (db *TitlesDB) UpdateTitlesDB() {
	_ = os.MkdirAll(db.settings.CacheFolder, 0755)

	// Download the latest titlesdb to the current folder
	for _, fileURL := range db.settings.TitlesDBURLs {
		path, err := utilities.DownloadFileWithVersioning(fileURL, db.settings.CacheFolder)
		if err != nil {
			log.Warn().Msgf("Downloading latest TitlesDB failed, will continue using cached - %v", err)
		}
		if err := db.injestTitleDBFile(path); err != nil {
			log.Error().Err(err).Str("url", fileURL).Msg("TitleDB couldn't parse downloaded data")
		} else {
			log.Info().Str("url", fileURL).Msg("Loaded TitleDB")
		}
	}
}

func (db *TitlesDB) injestTitleDBFile(path string) error {
	//Load json from the titlesDB file
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to load the Titledb - %w", err)
	}
	var entries map[string]TitleDBEntry
	err = json.Unmarshal(fileContents, &entries)
	if err != nil {
		return fmt.Errorf("failed to parse the Titledb - %w", err)
	}
	log.Info().Msgf("Loading %d entries from %s", len(entries), path)
	//Have to insert all entries into the map with update
	db.entriesLock.Lock()
	defer db.entriesLock.Unlock()
	for _, v := range entries {
		index, err := strconv.ParseUint(v.StringID, 16, 64)
		if err == nil {
			db.entries[index] = v
		}
	}

	return nil
}

func (db *TitlesDB) QueryGameFromTitleID(titleID uint64) (TitleDBEntry, bool) {
	db.entriesLock.RLock()
	defer db.entriesLock.RUnlock()
	value, ok := db.entries[titleID]
	return value, ok
}

func (db *TitlesDB) DumpToJSON(writer io.Writer) error {
	db.entriesLock.RLock()
	defer db.entriesLock.RUnlock()
	data, err := json.Marshal(db.entries)
	if err != nil {
		return fmt.Errorf("cant JSON'ify TitleDB - %w", err)
	}
	_, err = writer.Write(data)
	return err
}
