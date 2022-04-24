package index

import "testing"

func TestIndex_AddFileRecord(t *testing.T) {
	t.Parallel()
	idx := NewIndex(nil, nil)
	emptyTitle := FileOnDiskRecord{
		Path:    "/tmp/nope",
		TitleID: 0,
		Version: 0,
		Name:    "ShouldNotAppear",
		Size:    404,
	}
	idx.AddFileRecord(&emptyTitle)
	if len(idx.filesKnown) != 0 {
		t.Error("Should not have stored a Title ID of 0")
		t.FailNow()
	}
	baseGame := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50000,
		Version: 0,
		Name:    "Test base game",
		Size:    132,
	}
	idx.AddFileRecord(&baseGame)
	if len(idx.filesKnown) != 1 {
		t.Error("Should store base game")
	}
	if _, ok := idx.filesKnown[0x50000]; !ok {
		t.Error("Should have stored the file at the masked base ID")
	}

	idx.AddFileRecord(&baseGame)
	if len(idx.filesKnown) != 1 {
		t.Error("Should overwrite base game duplicate")
	}
	if idx.statistics.TotalTitles != 1 {
		t.Error("Should track the total titles")
	}
	gameUpdate := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50800,
		Version: 0,
		Name:    "",
		Size:    0,
	}

	idx.AddFileRecord(&gameUpdate)
	if len(idx.filesKnown) != 1 {
		t.Error("Should append existing game record")
	}
	if idx.statistics.TotalUpdates != 1 {
		t.Error("Should track the total updates")
	}
	gameUpdate.Version = 2
	idx.AddFileRecord(&gameUpdate)
	if len(idx.filesKnown) != 1 {
		t.Error("Should append existing game record")
	}
	if idx.statistics.TotalUpdates != 1 {
		t.Error("Should track the total updates")
	}

	files, ok := idx.GetFilesForTitleID(0x50000)
	if !ok {
		t.Error("Should retrieve file data")
	}
	if files.BaseTitle == nil || files.BaseTitle.Size != 132 {
		t.Error("Should not have lost the base game")
	}
	if files.Update.Version != 2 {
		t.Error("Should only keep the newest update")
	}

	gameDLC1 := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50001,
		Version: 0,
		Name:    "",
		Size:    0,
	}

	idx.AddFileRecord(&gameDLC1)
	files, ok = idx.GetFilesForTitleID(0x50000)
	if !ok {
		t.Error("Should retrieve file data")
	}
	if files.DLC == nil || len(files.DLC) != 1 {
		t.Error("Should have the DLC")
	}

	gameDLC2 := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50002,
		Version: 0,
		Name:    "",
		Size:    0,
	}

	idx.AddFileRecord(&gameDLC2)
	files, ok = idx.GetFilesForTitleID(0x50000)
	if !ok {
		t.Error("Should retrieve file data")
	}
	if files.DLC == nil || len(files.DLC) != 2 {
		t.Error("Should have the DLC")
	}

	gameDLC3 := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50001,
		Version: 1,
		Name:    "",
		Size:    0,
	}

	idx.AddFileRecord(&gameDLC3)
	files, ok = idx.GetFilesForTitleID(0x50000)
	if !ok {
		t.Error("Should retrieve file data")
	}
	if files.DLC == nil || len(files.DLC) != 2 {
		t.Error("Should have the DLC")
	}

	gameDLC4 := FileOnDiskRecord{
		Path:    "",
		TitleID: 0x50001,
		Version: 10,
		Name:    "",
		Size:    0,
	}

	idx.AddFileRecord(&gameDLC4)
	files, ok = idx.GetFilesForTitleID(0x50000)
	if !ok {
		t.Error("Should retrieve file data")
	}
	if files.DLC == nil || len(files.DLC) != 2 {
		t.Error("Should have the DLC")
	}
}
