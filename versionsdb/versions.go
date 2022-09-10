package versionsdb

import (
	"encoding/json"
	"github.com/ralim/switchhost/utilities"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strconv"
	"sync"
)

type VersionDB struct {
	sync.RWMutex
	latestVersions map[uint64]uint32
}

func NewVersionDBFromURL(url, cacheFolder string) *VersionDB {

	filePath, err := utilities.DownloadFileWithVersioning(url, cacheFolder)
	if err != nil {
		return &VersionDB{}
	}
	file, err := os.Open(filePath)
	if err != nil {
		return &VersionDB{}
	}
	defer file.Close()
	return NewVersionDB(file)
}

func NewVersionDB(r io.Reader) *VersionDB {
	db := &VersionDB{
		latestVersions: make(map[uint64]uint32),
	}
	db.Lock()
	defer db.Unlock()
	data := &map[string]map[string]string{}
	jsonBlob, err := io.ReadAll(r)
	if err != nil {
		return db
	}
	if err := json.Unmarshal(jsonBlob, data); err != nil || data == nil {
		return db
	}
	//Now walk the map and parse it into a usable lookup
	for titleID, versions := range *data {
		titleInt := uint64(0)
		if titleInt, err = strconv.ParseUint(titleID, 16, 64); err != nil {
			log.Warn().Str("title", titleID).Msg("TitleID failed parsing in versions update")
		}
		value := uint32(0)
		if existing, ok := db.latestVersions[titleInt]; ok {
			value = existing
		}
		//Find newest version
		for k := range versions {
			thisVersion := uint64(0)
			if thisVersion, err = strconv.ParseUint(k, 10, 32); err != nil {
				log.Warn().Err(err).Str("value", k).Msg("Title version failed parsing in versions update")
			}
			if uint32(thisVersion) > value {
				value = uint32(thisVersion)
			}
		}
		db.latestVersions[titleInt] = value

	}
	return db
}

// LookupLatestVersion returns the newest version for this TitleID or 0 if none found
func (db *VersionDB) LookupLatestVersion(titleID uint64) uint32 {
	db.RLock()
	defer db.RUnlock()
	if newest, ok := db.latestVersions[titleID]; ok {
		return newest
	}
	return 0
}
