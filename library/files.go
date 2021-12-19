package library

import (
	"os"
	"path"
	"strings"

	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

//TitleOnDiskCollection is a semi-logical grouping of titles on disk

type TitleOnDiskCollection struct {
	BaseTitle *FileOnDiskRecord
	Update    *FileOnDiskRecord
	DLC       []FileOnDiskRecord
}

type FileOnDiskRecord struct {
	Path    string
	TitleID uint64
	Version uint32
	Name    string
	Size    int64
}

func (r *TitleOnDiskCollection) GetFiles() []FileOnDiskRecord {
	values := make([]FileOnDiskRecord, 0)
	if r.BaseTitle != nil {
		values = append(values, *r.BaseTitle)
	}
	if r.Update != nil {
		values = append(values, *r.Update)
	}

	values = append(values, r.DLC...)
	return values
}

//Lists all tracked files
func (lib *Library) ListFiles() []FileOnDiskRecord {
	values := make([]FileOnDiskRecord, 0, len(lib.filesKnown))
	for _, v := range lib.filesKnown {
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

//Will only lists title files, of if title is missing the update, if thats missing, the dlc
func (lib *Library) ListTitleFiles() []FileOnDiskRecord {
	values := make([]FileOnDiskRecord, 0, len(lib.filesKnown))
	for _, v := range lib.filesKnown {
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
func (lib *Library) LookupFileInfo(file FileOnDiskRecord) (titledb.TitleDBEntry, bool) {
	return lib.titledb.QueryGameFromTitleID(file.TitleID)
}

func (lib *Library) AddFileRecord(file *FileOnDiskRecord) {
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

	oldValue, ok := lib.filesKnown[baseTitle]
	if !ok {
		//Need to make entry
		oldValue = TitleOnDiskCollection{}
	}
	if baseTitle == file.TitleID {
		//Check if we are attempting an overwrite
		oldValue.BaseTitle = lib.handleFileCollision(oldValue.BaseTitle, file)
	} else if (file.TitleID & 0x0000000000000800) == 0x800 {
		oldValue.Update = lib.handleFileCollision(oldValue.Update, file)
	} else {
		if oldValue.DLC == nil {
			oldValue.DLC = []FileOnDiskRecord{*file}
		} else {
			matched := false
			for index, oldFile := range oldValue.DLC {
				if oldFile.TitleID == file.TitleID {
					matched = true
					oldValue.DLC[index] = *lib.handleFileCollision(&oldFile, file)
				}
			}
			if !matched {
				oldValue.DLC = append(oldValue.DLC, *file)
			}
		}
	}
	lib.filesKnown[baseTitle] = oldValue
}

func (lib *Library) handleFileCollision(existing, proposed *FileOnDiskRecord) *FileOnDiskRecord {
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
	if lib.settings.Deduplicate {
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
			//Prefer compressed files
			if strings.HasSuffix(extNew, "z") != strings.HasSuffix(extOld, "z") {
				//Mismatch compression selection
				if strings.HasSuffix(extNew, "z") {
					log.Info().Str("path", old.Path).Msg("Cleaning up file as newer is compressed")
					if err := os.Remove(old.Path); err != nil {
						log.Warn().Str("path", old.Path).Msg("Failed to delete older file on collision")
					}
					return new
				} else {
					log.Info().Str("path", new.Path).Msg("Cleaning up file as older is compressed")
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
					if lib.settings.PreferXCI {
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

func (lib *Library) GetFilesForTitleID(titleid uint64) (TitleOnDiskCollection, bool) {
	val, ok := lib.filesKnown[titleid]
	return val, ok
}
func (lib *Library) GetFileRecord(titleID uint64, version uint32) (*FileOnDiskRecord, bool) {
	baseTitle := titleID & 0xFFFFFFFFFFFFE000
	record, ok := lib.filesKnown[baseTitle]
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
