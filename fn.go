package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/vincent-peugnet/wsync/api"
)

// Identify added items to original slice
func addedItems(originals []string, editeds []string) []string {
	var addeds []string
	for _, v := range editeds {
		if !slices.Contains(originals, v) {
			addeds = append(addeds, v)
		}
	}
	return addeds
}

// identify removed item from original slice
func removedItems(originals []string, editeds []string) []string {
	var removeds []string
	for _, v := range originals {
		if !slices.Contains(editeds, v) {
			removeds = append(removeds, v)
		}
	}
	return removeds
}

type ParsedVersion struct {
	Major int
	Minor int
	Patch int
}

func parseVersion(version string) (*ParsedVersion, error) {
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)
	result := re.FindStringSubmatch(version)

	if result == nil {
		return nil, fmt.Errorf("could not match version patern in %q", version)
	}

	v := ParsedVersion{}
	v.Major, _ = strconv.Atoi(result[1])
	v.Minor, _ = strconv.Atoi(result[2])
	v.Patch, _ = strconv.Atoi(result[3])

	return &v, nil
}

func GetPagePath(id string) string {
	filename := id + ".md"
	return filepath.Join(repoPath, filename)
}

func SaveToken(token string) {
	filename := filepath.Join(repoPath, TokenPath)
	if err := os.MkdirAll(filepath.Dir(filename), 0775); err != nil {
		log.Fatalln("save token:", err)
	}

	if err := os.WriteFile(filename, []byte(token), 0640); err != nil {
		log.Fatalln("save token:", err)
	}
}

func LoadToken() string {
	filename := filepath.Join(repoPath, TokenPath)
	token, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln("load token:", err)
	}
	return string(token)
}

func conflict(db *Database, client *api.Client, id string) {
	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Which version of %q should be kept ?", id)).
				Description("⚠️  compare the two versions before choosing").
				Options(
					huh.NewOption("Both (keep conflict)", "both"),
					huh.NewOption("Server (force pull)", "server"),
					huh.NewOption("Local (force push)", "local"),
				).
				Value(&action),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	switch action {
	case "server":
		_, err := db.pullPage(client, id, true)
		if err != nil {
			fmt.Printf("❌  conflict for page %q: error while trying to force pull: %v\n", id, err)
		} else {
			fmt.Printf("⬇️  conflict for page %q: successfully force pulled\n", id)
		}
	case "local":
		_, err := db.pushPage(client, id, true)
		if err != nil {
			fmt.Printf("❌  conflict for page %q: error while trying to force push: %v\n", id, err)
		} else {
			fmt.Printf("⬆️  conflict for page %q: successfully force pushed\n", id)
		}
	default:
		fmt.Printf("⚔️  conflict for page %q: both version kept\n", id)
	}
}
