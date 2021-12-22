package main

import (
	"os"
	"os/signal"
	"path"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/server"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/titledb"
	"github.com/rs/zerolog/log"
)

func main() {
	settingsPath := "config.json"
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	}
	settings := settings.NewSettings(settingsPath)
	Titles := titledb.CreateTitlesDB(settings)
	Titles.UpdateTitlesDB()
	lib := library.NewLibrary(Titles, settings)
	tryAndLoadKeys(lib)
	if err := lib.Start(); err != nil {
		panic(err)
	}

	server := server.NewServer(lib, Titles, settings)

	server.Run()
	SignalChannel := make(chan os.Signal, 1)

	signal.Notify(SignalChannel, os.Interrupt)

	<-SignalChannel
	log.Warn().Msg("Ctrl-c pressed, closing up")
	server.Stop() // stop the servers
	lib.Stop()    // wait for library to close down

}

func tryAndLoadKeys(lib *library.Library) {
	paths := []string{"."}
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, path.Join(userHomeDir, ".switch"))
	}

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
