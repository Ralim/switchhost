package index

type FileOnDiskRecord struct {
	Path    string
	TitleID uint64
	Version uint32
	Name    string
	Size    int64
}
