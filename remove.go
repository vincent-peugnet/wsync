package main

import (
	"fmt"
	"log"
)

func Remove(args []string) {
	if len(args) < 1 {
		log.Fatalln("remove sub-command need at least one page id argument")
	}

	database := LoadDatabase()

	for _, id := range args {
		fileDeleted, err := database.removePage(id)
		if err != nil {
			fmt.Printf("❌ error while removing %q: %v\n", id, err)
		} else if fileDeleted {
			fmt.Printf("🗑️  removed page %q and deleted local associated file\n", id)
		} else {
			fmt.Printf("🛡️  untracked page %q, but kept %q file because of local modifications\n", id, GetPagePath(id))
		}
	}

	SaveDatabase(database)
}
