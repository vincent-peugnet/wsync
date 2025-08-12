package main

import (
	"fmt"
	"log"

	"github.com/vincent-peugnet/wsync/api"
)

func Add(args []string) {
	if len(args) < 1 {
		log.Fatalln("add sub-command need at least one page id argument")
	}

	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	for _, id := range args {
		err := database.addPage(client, id)
		if err != nil {
			fmt.Printf("❌ error while adding page %q: %v\n", id, err)
		} else {
			fmt.Printf("⭐️ added new tracked page %q, created new file %q\n", id, GetPagePath(id))
		}
	}

	SaveDatabase(database)
}
