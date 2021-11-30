package library

import titledb "github.com/ralim/switchhost/titledb"

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
	oldValue, ok := lib.filesKnown[baseTitle]
	if !ok {
		//Need to make entry
		oldValue = TitleOnDiskCollection{}
	}
	if baseTitle == file.TitleID {
		oldValue.BaseTitle = file
	} else if (file.TitleID & 0x0000000000000800) == 0x800 {
		if oldValue.Update != nil {
			if oldValue.Update.Version < file.Version {
				oldValue.Update = file
			}
		} else {
			oldValue.Update = file
		}
	} else {
		if oldValue.DLC == nil {
			oldValue.DLC = []FileOnDiskRecord{*file}
		} else {
			oldValue.DLC = append(oldValue.DLC, *file)
		}
	}
	lib.filesKnown[baseTitle] = oldValue

	//TODO this function should note duplicates found on overwrite
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
