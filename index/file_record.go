package index

import "strings"

type FileOnDiskRecord struct {
	Path    string
	TitleID uint64
	Version uint32
	Name    string
	Size    int64
}

// ByName implements sort.Interface based on the Name field.
type ByName []FileOnDiskRecord

func (a ByName) Len() int           { return len(a) }
func (a ByName) Less(i, j int) bool { return strings.Compare(a[i].Name, a[j].Name) == -1 }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
