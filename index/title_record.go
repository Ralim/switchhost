package index

//TitleOnDiskCollection is a semi-logical grouping of titles on disk
type TitleOnDiskCollection struct {
	BaseTitle *FileOnDiskRecord
	Update    *FileOnDiskRecord
	DLC       []FileOnDiskRecord
}

//Returns all the files in the collection
func (r *TitleOnDiskCollection) GetFiles() []FileOnDiskRecord {
	values := []FileOnDiskRecord{}
	if r.BaseTitle != nil {
		values = append(values, *r.BaseTitle)
	}
	if r.Update != nil {
		values = append(values, *r.Update)
	}

	values = append(values, r.DLC...)
	return values
}
