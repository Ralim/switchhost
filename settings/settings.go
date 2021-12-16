package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Settings struct {
	PreferredLangOrder  []int    `json:"preferredLanguageOrder"` // List of language id's to use when parsing CNMT data area
	TitlesDBURLs        []string `json:"titlesDbUrls"`           // URL's to use when loading the local titledb
	FoldersToScan       []string `json:"sourceFolders"`          // Folders to look for new files in
	HTTPPort            int      `json:"httpPort"`               // Port used for HTTP
	FTPPort             int      `json:"ftpPort"`                // Port used for FTP
	StorageFolder       string   `json:"storageFolder"`          // Where sorted files are stored to
	CacheFolder         string   `json:"cacheFolder"`            // Folder to cache downloads and other temp files, if preserved will avoid re-downloads. Can be temp tho
	OrganisationFormat  string   `json:"organisationFormat"`     // Organisation format string
	EnableSorting       bool     `json:"enableSorting"`          // If sorting should be performed
	CleanupEmptyFolders bool     `json:"cleanupEmptyFolders"`    // Should we cleanup empty folders in the search and storage paths
	ServerMOTD          string   `json:"serverMOTD"`             // Server title used for public facing info
	// Private
	filePath string
}

// NewSettings creates settings with sane defaults
// And then loads any settings from the provided path (overwriting defaults)
func NewSettings(path string) *Settings {

	settings := &Settings{
		filePath:            path,
		PreferredLangOrder:  []int{1, 0},
		FoldersToScan:       []string{"./incoming_files"},
		StorageFolder:       "./game_library",
		CacheFolder:         "/tmp/",
		EnableSorting:       false, // default "safe"
		CleanupEmptyFolders: true,  // Relatively safe
		HTTPPort:            8080,
		FTPPort:             2121,
		ServerMOTD:          "Switchroot",
		OrganisationFormat:  "{TitleName}/{TitleName} {Type} {VersionDec} [{TitleID}][{Version}]",
		TitlesDBURLs: []string{
			// "https://tinfoil.media/repo/db/titles.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/US.en.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/AU.en.json",
		},
	}
	//Load the settings file if it exsts, which will override the defaults above if specified
	settings.Load()
	//Save to preserve if we have added anything to the file, and drop no-longer used settings for clarity
	settings.Save()
	return settings
}

func (s *Settings) Load() {
	//Load existing settings file if possible; if not load do nothing
	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, s); err != nil {
		fmt.Println("Couldn't load settings", err)
	}
}

func (s *Settings) Save() {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't save settings - %v", err)
		return
	}
	err = ioutil.WriteFile(s.filePath, data, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't save settings - %v", err)
	}
}

func (s *Settings) GetAllScanFolders() []string {
	res := []string{s.StorageFolder}
	res = append(res, s.FoldersToScan...)
	return res
}
