package main

import (
	"os"

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

	// Try and load keys from user home
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	Titles := titledb.CreateTitlesDB(settings)
	err = Titles.UpdateTitlesDB()
	if err != nil {
		panic(err)
	}

	lib := library.NewLibrary(Titles, settings)

	if _, err := os.Stat(userHomeDir + "/.switch/prod.keys"); err == nil {
		log.Info().Msg("Loading keys")

		file, err := os.Open(userHomeDir + "/.switch/prod.keys")
		if err != nil {
			panic(err)
		}
		if err := lib.LoadKeys(file); err != nil {
			log.Info().Msgf("Could not load keys -> %v", err)
		}
		file.Close()
	}
	err = lib.Start()
	if err != nil {
		panic(err)
	}

	server := server.NewServer(lib, Titles, settings)
	server.Run()
}
