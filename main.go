package main

import (
	"fmt"
	"os"
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

func main() {
	settingsPath := "config.json"
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	}

	ui := termui.NewTermUI()

	settings := settings.NewSettings(settingsPath)
	settings.SetupLogging(tview.ANSIWriter(ui.LogsView))

	uiExit := make(chan bool, 1)
	go func() {
		ui.Run()
		ui.Stop()
		uiExit <- true
	}()

	// Download TitlesDB

	titlesDBInfo := ui.RegisterTask("TitlesDB")
	titlesDBInfo.UpdateStatus("Downloading")
	Titles := titledb.CreateTitlesDB(settings)
	Titles.UpdateTitlesDB()
	titlesDBInfo.UpdateStatus("Done")

	lib := library.NewLibrary(Titles, settings, ui)

	tryAndLoadKeys(lib)

	lib.Start()

	server := server.NewServer(lib, Titles, settings)

	server.Run()
	<-uiExit
	//Rediect logs back to terminal
	settings.SetupLogging(os.Stdout)
	log.Warn().Msg("Ctrl-c pressed, closing up")
	fmt.Println("Waiting for tasks to stop")
	server.Stop() // stop the servers
	lib.Stop()    // wait for library to close down

}

func tryAndLoadKeys(lib *library.Library) {
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
	fmt.Println(paths)

	for _, path := range paths {
		if ok := loadKeys(path, lib); ok {
			return // Done loading
		}
	}
	log.Warn().Msg("No keys could be loaded, functionality will be limited")
}

func loadKeys(folder string, lib *library.Library) bool {
	path := path.Join(folder, "prod.keys")
	if _, err := os.Stat(path); err == nil {
		log.Info().Str("path", path).Msg("Loading keys...")

		file, err := os.Open(path)
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
