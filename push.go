package main

import (
	"fmt"
	"maps"
	"slices"

	"github.com/vincent-peugnet/wsync/api"
)

func Push(args []string) {
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
		pushed, err := database.pushPage(client, id, force)
		if err != nil {
			fmt.Printf("❌ could not push page: %q %v\n", id, err)
			i++
		}
		if pushed {
			fmt.Printf("⬆️  pushed page %q - ", id)
			fmt.Print(database.Config.BaseURL + "/" + id + "\n")
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all tracked pages are already up to date")
	}
	SaveDatabase(database)
}
