package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/ralim/switchhost/versionsdb"
	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/titledb"
	"github.com/ralim/switchhost/utilities"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type SwitchHost struct {
	ConfigFilePath string `flag:"config" help:"Path to config file"`
	KeysFilePath   string `flag:"keys" help:"Path to your switch's keyfile"`
	NoCUI          bool   `flag:"noCUI" help:"Disable the Console UI"`

	lib       *library.Library      `flag:"-"`
	ui        *termui.TermUI        `flag:"-"`
	settings  *settings.Settings    `flag:"-"`
	titleDB   *titledb.TitlesDB     `flag:"-"`
	versionDB *versionsdb.VersionDB `flag:"-"`
}

func NewSwitchHost() *SwitchHost {
	return &SwitchHost{}
}

func (m *SwitchHost) Run() error {
	uiExit := make(chan bool, 1)

	settingsPath := "./config.json"
	if m.ConfigFilePath != "" {
		settingsPath = m.ConfigFilePath
	}
	m.settings = settings.NewSettings(settingsPath)
	m.ui = termui.NewTermUI(m.NoCUI)
	if !m.NoCUI {
		m.settings.SetupLogging(tview.ANSIWriter(m.ui.LogsView))
		go func() {
			m.ui.Run()
			m.ui.Stop()
			uiExit <- true
		}()
	} else {
		m.settings.SetupLogging(os.Stdout)
		//Run hook listener for ctrl-c
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				// sig is a ^C, handle it
				log.Warn().Msg("Control-C received, shutting down")
				uiExit <- true
			}
		}()
	}

	m.loadVersionInfo()
	// Download TitlesDB
	m.loadTitlesDB()

	m.lib = library.NewLibrary(m.titleDB, m.settings, m.ui, m.versionDB)

	m.tryAndLoadKeys()

	m.lib.Start()

	server := server.NewServer(m.lib, m.titleDB, m.settings)

	server.Run()

	//Wait for exit
	<-uiExit

	//Rediect logs back to terminal since UI has exited
	m.settings.SetupLogging(os.Stdout)
	log.Warn().Msg("Ctrl-c pressed, closing up")
	fmt.Println("Waiting for tasks to stop")
	server.Stop() // stop the servers
	m.lib.Stop()  // wait for library to close down

	return nil
}

func (m *SwitchHost) checkKeys(pathUsed string) {
	//So this is a bit of a hack around because nsz.py is inflexible in the prod.keys file location
	//So we check, if we have been told to load from a not ~/.switch/ location, and the dest file does not exist, we copy it there for nsz.py
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		homeFileFolderPath := path.Join(userHomeDir, ".switch")
		err := os.MkdirAll(homeFileFolderPath, os.ModePerm)
		if err == nil {
			homeFilePath := path.Join(homeFileFolderPath, "prod.keys")

			if homeFilePath != pathUsed && !utilities.Exists(homeFilePath) {
				log.Info().Str("pathUsed", pathUsed).Msg("Path used for keys does not equal the one nsz wants, so going to try and copy it there")
				err := utilities.CopyFile(pathUsed, homeFilePath)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to copy the prod.keys file, nsz.py may not work")
				}
			}
		} else {
			log.Warn().Err(err).Msg("Failed to copy the prod.keys file, as could not make folder; nsz.py may not work")
		}
	}
}

func (m *SwitchHost) loadTitlesDB() {
	if m.ui != nil {
		titlesDBInfo := m.ui.RegisterTask("TitlesDB")
		titlesDBInfo.UpdateStatus("Downloading")
		defer titlesDBInfo.UpdateStatus("Done")
	}

	m.titleDB = titledb.CreateTitlesDB(m.settings)
	m.titleDB.UpdateTitlesDB()

}

func (m *SwitchHost) loadVersionInfo() {

	if m.ui != nil {
		versionInfo := m.ui.RegisterTask("Version Info")
		versionInfo.UpdateStatus("Downloading")
		defer versionInfo.UpdateStatus("Done")
	}
	versionInfo := versionsdb.NewVersionDBFromURL(m.settings.VersionsDBURL, m.settings.CacheFolder)
	m.versionDB = versionInfo
}
func (m *SwitchHost) tryAndLoadKeys() {
	//First try cli arg path if we can
	if ok := m.loadKeys(m.KeysFilePath, m.lib); ok {
		return // Done loading
	}
	paths := []string{"."}
	//Append user home folder if it exists
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, path.Join(userHomeDir, ".switch"))
	}
	//Append executable folder if it exists
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	paths = append(paths, exPath)

	for _, folder := range paths {
		filePath := path.Join(folder, "prod.keys")
		if ok := m.loadKeys(filePath, m.lib); ok {
			return // Done loading
		}
	}
	log.Warn().Msg("No keys could be loaded, functionality will be limited")
}

func (m *SwitchHost) loadKeys(filePath string, lib *library.Library) bool {
	if _, err := os.Stat(filePath); err == nil {
		log.Info().Str("path", filePath).Msg("Loading keys...")

		file, err := os.Open(filePath)
		if err != nil {
			return false
		}
		defer file.Close()
		if err := lib.LoadKeys(file); err != nil {
			log.Info().Err(err).Msg("Could not load keys")
			return false
		}
		m.checkKeys(filePath)
		return true
	}
	return false
}
