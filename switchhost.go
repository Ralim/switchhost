package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/termui"
	"github.com/ralim/switchhost/titledb"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type SwitchHost struct {
	ConfigFilePath string `flag:"config" help:"Path to config file"`
	KeysFilePath   string `flag:"keys" help:"Path to your switch's keyfile"`
	NoCUI          bool   `flag:"noCUI" help:"Disable the Console UI"`

	lib      *library.Library   `flag:"-"`
	ui       *termui.TermUI     `flag:"-"`
	settings *settings.Settings `flag:"-"`
	titleDB  *titledb.TitlesDB  `flag:"-"`
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

	// Download TitlesDB
	m.loadTitlesDB()

	m.lib = library.NewLibrary(m.titleDB, m.settings, m.ui)

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

func (m *SwitchHost) loadTitlesDB() {

	if m.ui != nil {
		titlesDBInfo := m.ui.RegisterTask("TitlesDB")
		titlesDBInfo.UpdateStatus("Downloading")
		defer titlesDBInfo.UpdateStatus("Done")
	}

	m.titleDB = titledb.CreateTitlesDB(m.settings)
	m.titleDB.UpdateTitlesDB()

}
func (m *SwitchHost) tryAndLoadKeys() {
	//First try cli arg path if we can
	if ok := loadKeys(m.KeysFilePath, m.lib); ok {
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
		if ok := loadKeys(filePath, m.lib); ok {
			return // Done loading
		}
	}
	log.Warn().Msg("No keys could be loaded, functionality will be limited")
}

func loadKeys(filePath string, lib *library.Library) bool {
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
		return true
	}
	return false
}
