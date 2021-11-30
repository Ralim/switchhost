package library

import (
	"fmt"
)

//Wrappers for working with the titlesdb

func (lib *Library) QueryGameTitleFromTitleID(TitleID uint64) (string, error) {
	//Note: Game updates have the same ProgramId as the main application, except with bitmask 0x800 set.
	baseTitle := TitleID & 0xFFFFFFFFFFFFE000
	value, ok := lib.titledb.QueryGameFromTitleID(baseTitle)
	if !ok {
		return "", fmt.Errorf("couldnt look up title [%016x] - [%016x]", TitleID, baseTitle)
	}
	return value.Name, nil
}
