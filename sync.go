package main

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/vincent-peugnet/wsync/api"
)

func Sync(args []string) {
	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	var pages []string
	if len(args) > 0 {
		pages = args
	} else {
		pages = slices.Collect(maps.Keys(database.Pages))
	}
	var i int
	for _, id := range pages {
		synced, err := database.syncPage(client, id)
		if interactive && errors.Is(err, api.ErrConflict) {
			conflict(database, client, id)
			i++
		} else if err != nil {
			fmt.Printf("❌ could not sync page %q: %v\n", id, err)
			i++
		} else if synced {
			fmt.Printf("🔃 synced page %q ", id)
			fmt.Print(database.Config.BaseURL + "/" + id + "\n")
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all tracked pages are already in sync")
	}
	SaveDatabase(database)

}
