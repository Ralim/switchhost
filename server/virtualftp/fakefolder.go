package virtualftp

import (
	"os"
	"time"
)

type FakeFolder struct {
	os.FileInfo
	fakePath string
}

func NewFakeFolder(virtualFolder string) FakeFolder {
	return FakeFolder{
		fakePath: virtualFolder,
	}
}

func (v *FakeFolder) Name() string {
	return v.fakePath

}
func (v *FakeFolder) Size() int64 {
	return 0

}
func (v *FakeFolder) Mode() os.FileMode {
	return 0666

}
func (v *FakeFolder) ModTime() time.Time {
	return time.Now()
}
func (v *FakeFolder) IsDir() bool {
	return true
}

func (v *FakeFolder) Sys() interface{} {
	return v
}
