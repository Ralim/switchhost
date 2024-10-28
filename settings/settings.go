package settings

import (
	"encoding/json"
	"io"
	stdlog "log"
	"os"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type AuthUser struct {
	Username      string `json:"username"`      // User username for authentication
	Password      string `json:"password"`      // User password for authentication
	AllowFTP      bool   `json:"allowFTP"`      // Can user use the ftp server
	AllowHTTP     bool   `json:"allowHTTP"`     // Can user use the http server
	AllowUpload   bool   `json:"allowUpload"`   // Can user upload new files
	AllowSettings bool   `json:"allowSettings"` // Can user edit settings
}

type Settings struct {
	// File parsing

	PreferredLangOrder []int    `json:"preferredLanguageOrder"` // List of language id's to use when parsing CNMT data area
	TitlesDBURLs       []string `json:"titlesDbUrls"`           // URL's to use when loading the local titledb
	VersionsDBURL      string   `json:"versionsDBURL"`          // Versions JSON for updates
	FoldersToScan      []string `json:"sourceFolders"`          // Folders to look for new files in
	CacheFolder        string   `json:"cacheFolder"`            // Folder to cache downloads and other temp files, if preserved will avoid re-downloads. Can be /tmp/ though
	// Organisation
	StorageFolder       string `json:"storageFolder"`       // Where sorted files are stored to
	OrganisationFormat  string `json:"organisationFormat"`  // Organisation format string
	EnableSorting       bool   `json:"enableSorting"`       // If sorting should be performed
	CleanupEmptyFolders bool   `json:"cleanupEmptyFolders"` // Should we cleanup empty folders in the search and storage paths

	Deduplicate      bool `json:"deduplicate"`      // If we remove duplicate files for the same titleID, or old update files
	PreferXCI        bool `json:"preferXCI"`        // If when we find duplicates we pick the xci/xcz file over nsp/nsz
	PreferCompressed bool `json:"preferCompressed"` // Prefer compressed form of files on duplicate

	//Serving files
	HTTPSRewriteDomain string     `json:"httpsRewriteDomain"` // If this domain is used for HTTP, use HTTPS in response
	PublicIP           string     `json:"publicIP"`           // Public IP, required for FTP
	HTTPPort           int        `json:"httpPort"`           // Port used for HTTP
	FTPPort            int        `json:"ftpPort"`            // Port used for FTP
	FTPPassivePorts    string     `json:"FTPPassivePorts"`    // Passive port range for FTP
	FTPHost            string     `json:"FTPHost"`            //
	AllowAnonFTP       bool       `json:"allowAnonFTP"`       // Allow anon (open to public) FTP
	AllowAnonHTTP      bool       `json:"allowAnonHTTP"`      // Allow anon (open to public) HTTP
	Users              []AuthUser `json:"users"`              // User accounts
	JSONLocations      []string   `json:"jsonLocations"`      // Extra locations to add to locations field in json for backup instances
	ServerMOTD         string     `json:"serverMOTD"`         // Server title used for public facing info

	// Incoming
	UploadingAllowed bool   `json:"uploadingAllowed"`  // Can FTP be used to push new files
	TempFilesFolder  string `json:"tempFilesFolder"`   // Temporary file storage location for FTP uploads
	OpTheadCounts    int    `json:"workerThreadCount"` // Optional thread count override
	// File validation
	ValidateLibrary         bool `json:"validateLibrary"`       // If all files found in the main library location are validated for checksums
	ValidateNewFiles        bool `json:"validateUploads"`       // If uploads must validate before being added, even if above toggles are off
	ValidateCompressedFiles bool `json:"validateCompressed"`    // If files are re-validated after compression
	DeleteValidationFails   bool `json:"deleteValidationFails"` // If a file fails validation, should it be deleted

	// Compression
	NSZCommandLine         string `json:"NSZCommandLine"`         // Base command line used to run NSZ
	CompressionEnabled     bool   `json:"compressionEnabled"`     // Should files be converted to their compressed verions
	CompressionTimeoutMins uint32 `json:"compressionTimeoutMins"` // How many mins compression can take max

	// Misc
	LogLevel    int    `json:"logLevel"`    // Log level, higher numbers reduce log output
	LogFilePath string `json:"logPath"`     // Path to persist logs to, if empty none are persisted
	QueueLength int    `json:"queueLength"` // How deep our internal queues are
	// Private
	filePath string
	logFile  *os.File
}

// NewSettings creates settings with sane defaults
// And then loads any settings from the provided path (overwriting defaults)
func NewSettings(path string) *Settings {

	settings := &Settings{
		OpTheadCounts:          -1, // No thread count override
		filePath:               path,
		PreferredLangOrder:     []int{1, 0},
		FoldersToScan:          []string{"./incoming_files"},                                         // Search locations
		JSONLocations:          []string{},                                                           // Locations in the json to point to backup instances
		StorageFolder:          "./game_library",                                                     // Storage location
		CacheFolder:            "/tmp/",                                                              // Where to cache downloaded files to (titledb)
		EnableSorting:          false,                                                                // default "safe"
		CleanupEmptyFolders:    true,                                                                 // Relatively safe
		HTTPPort:               8080,                                                                 // Ports
		FTPPort:                2121,                                                                 // Ports
		FTPPassivePorts:        "2130-2140",                                                          // FTP Passive ports
		FTPHost:                "::",                                                                 // Default to all ftp hosts
		PublicIP:               "",                                                                   // Default to not set
		ServerMOTD:             "Switchroot",                                                         // MOTD to include in the json file
		LogLevel:               1,                                                                    // Info
		LogFilePath:            "",                                                                   // No log file
		OrganisationFormat:     "{TitleName}/{TitleName} {Type} {VersionDec} [{TitleID}][{Version}]", // Path used for organising files
		NSZCommandLine:         "nsz --verify -w -C -p -t 4 --rm-source ",                            // NSZ command used for file compression
		CompressionEnabled:     false,                                                                // Should files be compressed using NSZ
		PreferCompressed:       true,                                                                 // Should compressed files be preferred over non-compressed on duplicate
		PreferXCI:              false,                                                                // Should XCI files be preferred over nsp on duplicate
		UploadingAllowed:       false,                                                                // Should FTP allow file uploads
		Deduplicate:            false,                                                                // Should the software delete duplicate files
		AllowAnonFTP:           false,                                                                // Should anon users be allowed FTP access
		AllowAnonHTTP:          false,                                                                // Should anon users be allowed HTTP access
		DeleteValidationFails:  false,                                                                //
		logFile:                nil,                                                                  // Optional path to a file to log to
		TempFilesFolder:        "/tmp",                                                               // Temp files location used for staging FTP uploads
		ValidateLibrary:        false,                                                                // Should all existing library files be validated
		ValidateNewFiles:       true,                                                                 // Should "new" files be validated (upload + not library)
		QueueLength:            128,                                                                  // Default to a medium sized queue. Large values are good for speed but consume ram
		CompressionTimeoutMins: 60,                                                                   // We are super conservative incase of user with slow pc
		//Add a demo account
		Users: []AuthUser{
			{
				Username:    "demo",
				Password:    "demo",
				AllowFTP:    false,
				AllowHTTP:   false,
				AllowUpload: false,
			},
		},
		TitlesDBURLs: []string{
			// "https://tinfoil.media/repo/db/titles.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/US.en.json",
			"https://raw.githubusercontent.com/blawar/titledb/master/AU.en.json",
		},
		VersionsDBURL: "https://raw.githubusercontent.com/blawar/titledb/master/versions.json",
	}
	// Load the settings file if it exsts, which will override the defaults above if specified
	settings.Load()
	// Clean up paths
	settings.cleanPaths()
	// Save to preserve if we have added anything to the file, and drop no-longer used settings for clarity
	settings.Save()
	log.Info().Msg("Settings loaded, merged and saved")
	return settings
}
func (s *Settings) LoadFrom(reader io.Reader) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warn().Err(err).Msg("Couldn't load settings")
	}
}
func (s *Settings) Load() {
	//Load existing settings file if possible; if not load do nothing
	log.Info().Str("path", s.filePath).Msg("Loading settings")
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warn().Err(err).Msg("Couldn't load settings")
	}
}
func (s *Settings) SaveTo(wr io.Writer) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	_, err = wr.Write(data)
	if err != nil {
		return err
	}
	return nil
}
func (s *Settings) Save() {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Warn().Err(err).Msg("Couldn't save settings - JSONification")
		return
	}
	if err = os.WriteFile(s.filePath, data, 0666); err != nil {
		log.Warn().Err(err).Msg("Couldn't save settings - writing file")
	}
}

func (s *Settings) GetAllScanFolders() []string {
	res := []string{s.StorageFolder}
	for _, folder := range s.FoldersToScan {
		if folder != s.StorageFolder {
			res = append(res, folder)
		}
	}
	return res
}

func (s *Settings) SetupLogging(logoutput io.Writer) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.Level(s.LogLevel))

	consoleWriter := zerolog.ConsoleWriter{
		Out:        logoutput,
		TimeFormat: zerolog.TimeFormatUnix,
	}

	stdlog.SetOutput(consoleWriter)
	log.Logger = log.Output(consoleWriter)

	if len(s.LogFilePath) > 0 {
		//Setup a mirror of the log to the specified file
		logfile, err := os.OpenFile(s.LogFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Warn().Str("file", s.LogFilePath).Err(err).Msg("Couldn't open log file for writing")
			return
		}
		s.logFile = logfile

		multi := zerolog.MultiLevelWriter(consoleWriter, s.logFile)

		stdlog.SetOutput(multi)
		log.Logger = log.Output(multi)
		log.Info().Msg("Started logging to file")

	}
}

func (s *Settings) cleanPaths() {
	//Since users may make mistakes and start or end the paths with a string, clean all of these up
	s.TempFilesFolder = strings.TrimSpace(s.TempFilesFolder)
	s.StorageFolder = strings.TrimSpace(s.StorageFolder)
	s.CacheFolder = strings.TrimSpace(s.CacheFolder)
	for i, v := range s.FoldersToScan {
		s.FoldersToScan[i] = strings.TrimSpace(v)
	}

}

func (s *Settings) GetCPUCount() int {
	if s.OpTheadCounts > 0 {
		return s.OpTheadCounts
	}
	return runtime.NumCPU()
}
