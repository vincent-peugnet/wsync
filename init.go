package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/huh"
	"github.com/vincent-peugnet/wsync/api"
)

func Init(args []string) {

	files, err := os.ReadDir(repoPath)
	if err != nil {
		log.Fatalln("read folder:", err)
	}

	if len(files) > 0 {
		log.Fatalln("directory is not empty")
	}

	absoluteRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatalln(err)
	}

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirm use of path: '" + absoluteRepoPath + "'").
				Description("Do you want to use this folder to store the pages ?").
				Value(&confirm),
		),
	)
	if err := confirmForm.Run(); err != nil {
		log.Fatal(err)
	}
	if !confirm {
		log.Fatalln("❌ init aborted")
	}

	var baseURL string
	if len(args) < 1 {
		baseURLForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("What is the URL where W is installed ?").
					Value(&baseURL).
					Placeholder("https://example.com"),
			),
		)

		if err := baseURLForm.Run(); err != nil {
			log.Fatal(err)
		}
	} else {
		baseURL = args[0]
	}
	client := api.NewClient(baseURL)

	if _, err := client.Version(); err != nil {
		log.Fatalln("❌ERROR: there is no W at this adress.", err)
	}

	database := LoadDatabase()
	database.Config.BaseURL = baseURL

	log.Println("🔌 connected to W")

	var username string
	var password string

	credentialForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&username),
			huh.NewInput().
				EchoMode(huh.EchoMode(textinput.EchoPassword)).
				Title("Password").
				Value(&password),
		),
	)

	if err := credentialForm.Run(); err != nil {
		log.Fatal(err)
	}

	token, err := client.Auth(username, password)
	if err != nil {
		log.Fatal(err)
	}

	SaveDatabase(database)
	SaveToken(token)

	log.Println("🔓️ logged in")
	fmt.Println("⭐️ repository initalized")
}
