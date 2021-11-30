package virtualftp

import (
	"os"
	"time"
)

type FakeFile struct {
	os.FileInfo
	fakePath string
	realFile os.FileInfo
}

func NewFakeFile(fakepath string, realFile os.FileInfo) FakeFile {
	return FakeFile{
		fakePath: fakepath,
		realFile: realFile,
	}
}

func (v *FakeFile) Name() string {
	return v.fakePath
}
func (v *FakeFile) Size() int64 {
	return v.realFile.Size()

}
func (v *FakeFile) Mode() os.FileMode {
	return v.realFile.Mode()

}
func (v *FakeFile) ModTime() time.Time {
	return v.realFile.ModTime()
}
func (v *FakeFile) IsDir() bool {
	return v.realFile.IsDir()
}

func (v *FakeFile) Sys() interface{} {
	return v.realFile.Sys()
}
