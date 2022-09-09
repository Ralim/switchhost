package titledb

import (
	"os"
	"reflect"
	"testing"

	"github.com/ralim/switchhost/settings"
)

func TestInjestTitleDBFile(t *testing.T) {
	sampleRecords := `
	{
		"FF007EF000AAA000": {
		  "id": "FF007EF000AAA000",
		  "name": "The Testing Game",
		  "releaseDate": 20210101,
		  "category": [
			"Adventure"
		  ],
		  "numberOfPlayers": 1,
		  "frontBoxArt": null,
		  "iconUrl": "https://dummy0.jpg",
		  "screenshots": [
			"https://dummy1.jpg",
			"https://dummy2.jpg"
		  ],
		  "bannerUrl": "https://dummy3.jpg"
		},
		"xxxx": {
		  "id": "FF007EF000AAA001",
		  "name": "The Testing Game, the sequal",
		  "releaseDate": 20210610,
		  "category": [
			"Adventure"
		  ],
		  "numberOfPlayers": 2,
		  "frontBoxArt": null,
		  "iconUrl": "https://dummy01.jpg",
		  "screenshots": [
			"https://dummy11.jpg",
			"https://dummy21.jpg"
		  ],
		  "bannerUrl": "https://dummy31.jpg"
		}
	  }
`
	tmpFile, err := os.CreateTemp(os.TempDir(), "titlesdb-test1-")
	if err != nil {
		t.Fatal("Cannot create temporary file", err)
	}

	defer os.Remove(tmpFile.Name())

	// Example writing to the file
	if _, err = tmpFile.Write([]byte(sampleRecords)); err != nil {
		t.Fatal("Failed to write to temporary file", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}
	settings := settings.NewSettings("/tmp/_titledbtest.json")
	db := CreateTitlesDB(settings)

	err = db.injestTitleDBFile(tmpFile.Name())
	if err != nil {
		t.Errorf("Should parse test data fine - %v", err)
	}
	if len(db.entries) != 2 {
		t.Error("Should injest both entries")
	}
	//Check entries
	expectedEntries := map[uint64]TitleDBEntry{
		18374826048940056576: {StringID: "FF007EF000AAA000",
			Name:        "The Testing Game",
			ReleaseDate: 20210101,
			NumPlayers:  1,
			IconURL:     "https://dummy0.jpg",
			BannerURL:   "https://dummy3.jpg",
			ScreenshotURLs: []string{
				"https://dummy1.jpg",
				"https://dummy2.jpg",
			},
		},
		18374826048940056577: {StringID: "FF007EF000AAA001",
			Name:        "The Testing Game, the sequal",
			ReleaseDate: 20210610,
			NumPlayers:  2,
			IconURL:     "https://dummy01.jpg",
			BannerURL:   "https://dummy31.jpg",
			ScreenshotURLs: []string{
				"https://dummy11.jpg",
				"https://dummy21.jpg",
			},
		},
	}
	if !reflect.DeepEqual(expectedEntries, db.entries) {
		t.Errorf("titledb does not match expected values, %+v <-> %+v", expectedEntries, db.entries)
	}
	value, ok := db.QueryGameFromTitleID(18374826048940056577)
	if !ok {
		t.Error("Lookup failed for 18374826048940056577")
	}
	if !reflect.DeepEqual(value, expectedEntries[18374826048940056577]) {
		t.Error("How did you manage to make this fail")
	}
	_, ok = db.QueryGameFromTitleID(0)
	if ok {
		t.Error("Should fail on unknown")
	}
}

func TestInjestTitleDBFileUnhappy(t *testing.T) {

	settings := settings.NewSettings("/tmp/_titledbtest.json")
	db := CreateTitlesDB(settings)

	err := db.injestTitleDBFile("/tmp/does-not-exist.json")
	if err == nil {
		t.Error("Should fail with error on bad file path, but didnt")
	}

	//bad json
	notJsonString := `{NotJson}`
	tmpFile, err := os.CreateTemp(os.TempDir(), "titlesdb-test1-")
	if err != nil {
		t.Fatal("Cannot create temporary file", err)
	}

	defer os.Remove(tmpFile.Name())

	// Example writing to the file
	if _, err = tmpFile.Write([]byte(notJsonString)); err != nil {
		t.Fatal("Failed to write to temporary file", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	err = db.injestTitleDBFile(tmpFile.Name())
	if err == nil {
		t.Error("Should fail with error on bad json")
	}
	tmpFile2, err := os.CreateTemp(os.TempDir(), "titlesdb-test1-")
	if err != nil {
		t.Fatal("Cannot create temporary file", err)
	}

	defer os.Remove(tmpFile2.Name())

	badTitleString := `
	{
		"FF007EF000AAA000": {
		  "id": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		  "name": "The Testing Game",
		  "releaseDate": 20210101,
		  "category": [
			"Adventure"
		  ],
		  "numberOfPlayers": 1,
		  "frontBoxArt": null,
		  "iconUrl": "https://dummy0.jpg",
		  "screenshots": [
			"https://dummy1.jpg",
			"https://dummy2.jpg"
		  ],
		  "bannerUrl": "https://dummy3.jpg"
		},
		"xxxx": {
		  "id": "",
		  "name": "The Testing Game, the sequal",
		  "releaseDate": 20210610,
		  "category": [
			"Adventure"
		  ],
		  "numberOfPlayers": 2,
		  "frontBoxArt": null,
		  "iconUrl": "https://dummy01.jpg",
		  "screenshots": [
			"https://dummy11.jpg",
			"https://dummy21.jpg"
		  ],
		  "bannerUrl": "https://dummy31.jpg"
		},
		"xxxy": {
		  "name": "The Testing Game, the sequal",
		  "releaseDate": 20210610,
		  "category": [
			"Adventure"
		  ],
		  "numberOfPlayers": 2,
		  "frontBoxArt": null,
		  "iconUrl": "https://dummy01.jpg",
		  "screenshots": [
			"https://dummy11.jpg",
			"https://dummy21.jpg"
		  ],
		  "bannerUrl": "https://dummy31.jpg"
		}
	  }
	`

	// Example writing to the file
	if _, err = tmpFile2.WriteAt([]byte(badTitleString), 0); err != nil {
		t.Fatal("Failed to write to temporary file badTitleString", err)
	}

	if err := tmpFile2.Close(); err != nil {
		t.Fatal(err)
	}

	err = db.injestTitleDBFile(tmpFile2.Name())
	if err != nil {
		t.Errorf("Should not fail with error on bad title - %v", err)
	}
}
