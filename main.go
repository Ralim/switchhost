package main

import (
	"os"
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
	err := lib.Start()
	if err != nil {
		panic(err)
	}

	server := server.NewServer(lib, Titles, settings)
	server.Run()
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
		log.Info().Msgf("Loading keys from %s", path)

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
