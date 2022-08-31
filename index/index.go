package index

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

type Index struct {
	sync.RWMutex // mutex to lock the entire filesKnown map
	// Totals for statistics
	statistics termui.Statistics

	//Passed in from the lib
	titledb  *titledb.TitlesDB
	settings *settings.Settings

	filesKnown map[uint64]TitleOnDiskCollection
}

func NewIndex(titledb *titledb.TitlesDB,
	settings *settings.Settings) *Index {
	return &Index{
		titledb:    titledb,
		settings:   settings,
		filesKnown: make(map[uint64]TitleOnDiskCollection),
	}
}

func (idx *Index) GetStats() termui.Statistics {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()
	return idx.statistics
}

// Lists all tracked files
func (idx *Index) ListFiles() []FileOnDiskRecord {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()

	values := make([]FileOnDiskRecord, 0, len(idx.filesKnown))
	for _, v := range idx.filesKnown {
		if v.BaseTitle != nil {
			values = append(values, *v.BaseTitle)
		}
		if v.Update != nil {
			values = append(values, *v.Update)
		}

		values = append(values, v.DLC...)
	}
	return values
}

// Will only lists title files, of if title is missing the update, if thats missing, the dlc
func (idx *Index) ListTitleFiles() []FileOnDiskRecord {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()

	values := make([]FileOnDiskRecord, 0, len(idx.filesKnown))
	for _, v := range idx.filesKnown {
		if v.BaseTitle != nil {
			values = append(values, *v.BaseTitle)
		} else if v.Update != nil {
			values = append(values, *v.Update)
		} else {
			values = append(values, v.DLC...)
		}
	}
	return values
}
func (idx *Index) LookupFileInfo(file FileOnDiskRecord) (titledb.TitleDBEntry, bool) {
	return idx.titledb.QueryGameFromTitleID(file.TitleID)
}

func (idx *Index) AddFileRecord(file *FileOnDiskRecord) {
	idx.RWMutex.Lock()
	defer idx.RWMutex.Unlock()
	//Depending on game type add to the appropriate record
	//Game updates have the same ProgramId as the main application, except with bitmask 0x800 set.
	//https://wiki.gbatemp.net/wiki/List_of_Switch_homebrew_titleID
	//ALL current Titles for Switch begins with 0100
	//System Titles are all in "010000000000XXXX".
	//Games end with "Y000". With Y being an even digit. Pattern: TitleID & 0xFFFFFFFFFFFFE000 (AND operand).
	//DLCs ends with "YXXX". With Y being an odd digit, and XXX a DLC ID from 0x000 to 0xFFF. DLC is 0x1000 greater than base Title ID. Pattern: Base TitleID | 1XXX (OR operand).
	//Updates ends with "0800"

	//Thus; base titleID can be found by AND'ing with 0xFFFFFFFFFFFFE000
	// Then we can sort by the fields to know what kind of file it is
	baseTitle := file.TitleID & 0xFFFFFFFFFFFFE000
	if baseTitle == 0 { // If we masked out all bits, not a valid file for us to sort and store
		log.Error().Str("file", file.Path).Msg("Couldn't determine titleID")
		return
	}

	oldValue, ok := idx.filesKnown[baseTitle]
	if !ok {
		//Need to make entry
		oldValue = TitleOnDiskCollection{}
	}
	if baseTitle == file.TitleID {
		//Check if we are attempting overwriting an existing entry
		if oldValue.BaseTitle == nil {
			idx.statistics.TotalTitles++
		}
		oldValue.BaseTitle = idx.handleFileCollision(oldValue.BaseTitle, file)
	} else if (file.TitleID & 0x0000000000000800) == 0x800 {
		if oldValue.Update == nil {
			idx.statistics.TotalUpdates++
		}
		oldValue.Update = idx.handleFileCollision(oldValue.Update, file)
	} else {
		if oldValue.DLC == nil {
			oldValue.DLC = []FileOnDiskRecord{*file}
			idx.statistics.TotalDLC++
		} else {
			matched := false
			for index, oldFile := range oldValue.DLC {
				if oldFile.TitleID == file.TitleID {
					matched = true
					oldValue.DLC[index] = *idx.handleFileCollision(&oldFile, file)
				}
			}
			if !matched {
				oldValue.DLC = append(oldValue.DLC, *file)
				idx.statistics.TotalDLC++
			}
		}
	}
	idx.filesKnown[baseTitle] = oldValue
}

func (idx *Index) handleFileCollision(existing, proposed *FileOnDiskRecord) *FileOnDiskRecord {
	//Given a collision, figure out the one to keep, do any deletes, and return the kept one
	if existing == nil {
		return proposed
	} else if proposed == nil {
		return existing
	}
	if existing.Path == proposed.Path {
		//Same file, dont care, send back newest
		return proposed
	}
	new := proposed
	old := existing
	if existing.Version > proposed.Version {
		//swapped
		old = proposed
		new = existing
	}
	if idx.settings.Deduplicate {
		//remove the older of the pair of files, or based on preferences
		if new.Version != old.Version {
			log.Info().Str("path", old.Path).Msg("Cleaning up file as newer exists")
			if err := os.Remove(old.Path); err != nil {
				log.Warn().Str("path", old.Path).Msg("Failed to delete older file on collision")
			}
			return new
		} else {
			//Same version, cleanup based on file extension
			extNew := strings.ToLower(path.Ext(new.Path))
			extOld := strings.ToLower(path.Ext(old.Path))
			//Prefer compressed files, if they are different we can use this to decide
			if strings.HasSuffix(extNew, "z") != strings.HasSuffix(extOld, "z") {
				//Mismatch compression selection
				selectNew := (strings.HasSuffix(extNew, "z") && idx.settings.PreferCompressed) || strings.HasSuffix(extOld, "z") && !idx.settings.PreferCompressed

				if selectNew {
					log.Info().Str("path", old.Path).Msg("Cleaning up file based on compression rules")
					if err := os.Remove(old.Path); err != nil {
						log.Warn().Str("path", old.Path).Msg("Failed to delete older file on collision")
					}
					return new
				} else {
					log.Info().Str("path", new.Path).Msg("Cleaning up file based on compression rules")
					if err := os.Remove(new.Path); err != nil {
						log.Warn().Str("path", new.Path).Msg("Failed to delete new file on collision")
					}
					return old
				}
			} else {
				//Compare on file types
				if extNew[0:3] != extOld[0:3] {
					newType := extNew[1:3]
					oldType := extOld[1:3]
					if idx.settings.PreferXCI {
						if newType == "xc" {
							log.Info().Str("path", old.Path).Msg("Cleaning up file as newer is preferred type")
							if err := os.Remove(old.Path); err != nil {
								log.Warn().Str("path", old.Path).Msg("Failed to delete older file on preferred type")
							}
							return new
						} else if oldType == "xc" {
							log.Info().Str("path", new.Path).Msg("Cleaning up file as older is preferred type")
							if err := os.Remove(new.Path); err != nil {
								log.Warn().Str("path", new.Path).Msg("Failed to delete new file on preferred type")
							}
							return old
						}
					} else {
						if newType == "ns" {
							log.Info().Str("path", old.Path).Msg("Cleaning up file as newer is preferred type")
							if err := os.Remove(old.Path); err != nil {
								log.Warn().Str("path", old.Path).Msg("Failed to delete older file on preferred type")
							}
							return new
						} else if oldType == "ns" {
							log.Info().Str("path", new.Path).Msg("Cleaning up file as older is preferred type")
							if err := os.Remove(new.Path); err != nil {
								log.Warn().Str("path", new.Path).Msg("Failed to delete new file on preferred type")
							}
							return old
						}
					}
				}
			}

		}
	}
	return new
}

func (idx *Index) GetFilesForTitleID(titleID uint64) (TitleOnDiskCollection, bool) {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()
	baseTitle := titleID & 0xFFFFFFFFFFFFE000
	val, ok := idx.filesKnown[baseTitle]
	return val, ok
}
func (idx *Index) GetAllRecordsForTitle(titleID uint64) []FileOnDiskRecord {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()
	resp := make([]FileOnDiskRecord, 0, 3)
	baseTitle := titleID & 0xFFFFFFFFFFFFE000
	record, ok := idx.filesKnown[baseTitle]
	if !ok {
		return resp
	}
	if record.BaseTitle != nil {
		resp = append(resp, *record.BaseTitle)
	}
	if record.Update != nil {
		resp = append(resp, *record.Update)
	}
	for _, dlcRecord := range record.DLC {
		resp = append(resp, dlcRecord)
	}
	return resp
}

func (idx *Index) GetFileRecord(titleID uint64, version uint32) (*FileOnDiskRecord, bool) {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()
	baseTitle := titleID & 0xFFFFFFFFFFFFE000
	record, ok := idx.filesKnown[baseTitle]
	if !ok {
		return nil, false
	}
	//Now look for the version tag && right titleid
	if record.BaseTitle != nil {
		if record.BaseTitle.TitleID == titleID && record.BaseTitle.Version == version {
			return record.BaseTitle, true
		}
	}
	if record.Update != nil {
		if record.Update.TitleID == titleID && record.Update.Version == version {
			return record.Update, true
		}
	}
	if record.DLC != nil {
		for _, v := range record.DLC {
			if v.TitleID == titleID && v.Version == version {
				return &v, true
			}
		}
	}
	return nil, false

}

func (idx *Index) GetTitleRecords(titleID uint64) (TitleOnDiskCollection, bool) {
	idx.RWMutex.RLocker().Lock()
	defer idx.RWMutex.RLocker().Unlock()
	baseTitle := titleID & 0xFFFFFFFFFFFFE000
	record, ok := idx.filesKnown[baseTitle]
	return record, ok

}
func (idx *Index) RemoveFile(path string) {
	idx.RWMutex.Lock()
	defer idx.RWMutex.Unlock()
	// Scan the list of known files and check if the path matches
	if oldPath, err := filepath.Abs(path); err == nil {
		log.Info().Str("path", oldPath).Msg("Delete event")
		for key, item := range idx.filesKnown {
			save := false
			if item.BaseTitle != nil && oldPath == item.BaseTitle.Path {
				item.BaseTitle = nil
				save = true
				idx.statistics.TotalTitles--
			} else if item.Update != nil && oldPath == item.Update.Path {
				item.Update = nil
				save = true
				idx.statistics.TotalUpdates--
			} else {
				//Check the DLC's
				matchingIndex := -1
				for i, d := range item.DLC {
					if d.Path == oldPath {

						matchingIndex = i
					}
				}
				if matchingIndex >= 0 {
					save = true
					idx.statistics.TotalDLC--
					//Slice out the item
					item.DLC[matchingIndex] = item.DLC[len(item.DLC)-1]
					item.DLC = item.DLC[:len(item.DLC)-1]
				}

			}

			if save {
				idx.filesKnown[key] = item
				return
			}
		}
	}
}
