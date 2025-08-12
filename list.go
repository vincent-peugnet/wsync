package main

import (
	"fmt"
	"log"
	"maps"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/vincent-peugnet/wsync/api"
)

func List() {
	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	ids, err := client.List()
	if err != nil {
		log.Fatalln(err)
	}

	var options []huh.Option[string]
	for _, id := range ids {
		_, tracked := database.Pages[id]
		options = append(options, huh.NewOption(id, id).Selected(tracked))
	}

	var selectedIds []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select pages to track").
				Options(options...).
				Value(&selectedIds).
				WithHeight(10),
		),
	)
	if err := form.Run(); err != nil {
		log.Fatal(err)
	}

	addedIds := addedItems(slices.Collect(maps.Keys(database.Pages)), selectedIds)
	removedIds := removedItems(slices.Collect(maps.Keys(database.Pages)), selectedIds)

	//TODO: the following sections call Pull() and Remove(), which also load the Database.
	// Would be better to use the same, already loaded, database
	// Maybe there should be removePages(), pullPages() and pushPages() functions that would take a database argument

	if len(addedIds) > 0 {
		var confirmAdd bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintln("add", len(addedIds), "new pages to your local repo ?")).
					Description(fmt.Sprintln(addedIds)).
					Value(&confirmAdd),
			),
		)
		if err := confirmForm.Run(); err != nil {
			log.Fatal(err)
		}
		if confirmAdd {
			Add(addedIds)
		}
	}

	if len(removedIds) > 0 {
		var confirmRemove bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintln("remove", len(removedIds), "pages from your local repo ?")).
					Description(fmt.Sprintln(removedIds)).
					Value(&confirmRemove),
			),
		)
		if err := confirmForm.Run(); err != nil {
			log.Fatal(err)
		}
		if confirmRemove {
			Remove(removedIds)
		}
	}
}
