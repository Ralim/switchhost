package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	LogLevel            int      `json:"logLevel"`               // Log level, higher numbers reduce log output
	LogFilePath         string   `json:"logPath"`                // Path to persist logs to, if empty none are persisted
	// Private
	filePath string
	logFile  *os.File
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
		LogLevel:            1,  //Info
		LogFilePath:         "", // No log file
		OrganisationFormat:  "{TitleName}/{TitleName} {Type} {VersionDec} [{TitleID}][{Version}]",
		TitlesDBURLs: []string{
			// "https://tinfoil.media/repo/db/titles.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/US.en.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/AU.en.json",
		},
	}
	// Load the settings file if it exsts, which will override the defaults above if specified
	settings.Load()
	// Save to preserve if we have added anything to the file, and drop no-longer used settings for clarity
	settings.Save()
	// Setup the logging
	settings.setupLogging()
	log.Info().Msg("Settings Loaded")
	return settings
}

func (s *Settings) Load() {
	//Load existing settings file if possible; if not load do nothing
	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warn().Msgf("Couldn't load settings -> %v", err)
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

func (s *Settings) setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.Level(s.LogLevel))
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	if len(s.LogFilePath) > 0 {
		//Setup a mirror of the log to the specified file
		logfile, err := os.OpenFile(s.LogFilePath, os.O_APPEND, 0666)
		if err == nil {

			s.logFile = logfile

			multi := zerolog.MultiLevelWriter(consoleWriter, s.logFile)

			logger := zerolog.New(multi).With().Timestamp().Logger()
			stdlog.SetOutput(logger)
			return
		} else {
			log.Warn().Msgf("Couldn't open log file %s for writing - %v", s.LogFilePath, err)
		}
	}
	//otherwise if we are here, just setup nice console logging
	stdlog.SetOutput(consoleWriter)
	log.Logger = log.Output(consoleWriter)

}
