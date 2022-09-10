package library

type GameUpdatePair struct {
	Title          string
	TitleID        uint64
	CurrentVersion uint32
	LatestVersion  uint32
}

func (lib *Library) GetGamesNeedingUpdate() []GameUpdatePair {
	//For all known games in the library, find out all the games that need an update
	results := make([]GameUpdatePair, 0, 50)
	if lib.versiondb == nil || lib.FileIndex == nil {
		return results
	}
	for _, title := range lib.FileIndex.ListTitleFiles() {
		titleInfo, _ := lib.FileIndex.GetTitleRecords(title.TitleID)
		latest := lib.versiondb.LookupLatestVersion(title.TitleID)
		updateLatest := uint32(0)
		if titleInfo.Update != nil {
			updateLatest = titleInfo.Update.Version
		}
		if latest > updateLatest {
			results = append(results, GameUpdatePair{
				Title:          title.Name,
				TitleID:        title.TitleID,
				CurrentVersion: updateLatest,
				LatestVersion:  latest,
			})
		}
	}

	return results
}
