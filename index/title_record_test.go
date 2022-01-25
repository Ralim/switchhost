package index

import (
	"reflect"
	"testing"
)

func TestGetFiles(t *testing.T) {
	itm := TitleOnDiskCollection{
		BaseTitle: &FileOnDiskRecord{Path: "111"},
		Update:    &FileOnDiskRecord{Path: "222"},
		DLC:       []FileOnDiskRecord{{Path: "333"}, {Path: "444"}},
	}
	files := itm.GetFiles()
	expected := []FileOnDiskRecord{{Path: "111"}, {Path: "222"}, {Path: "333"}, {Path: "444"}}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("Failed, wanted %v, got %v", expected, files)
	}
}
