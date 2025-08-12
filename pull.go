package main

import (
	"fmt"
	"maps"
	"slices"

	"github.com/vincent-peugnet/wsync/api"
)

func Pull(args []string) {
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
		pulled, err := database.pullPage(client, id, force)
		if err != nil {
			fmt.Printf("❌ could not pull page: %q: %v\n", id, err)
			i++
		}
		if pulled {
			fmt.Printf("⬇️  pulled page %q\n", id)
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all tracked pages are already up to date")
	}

	SaveDatabase(database)
}
