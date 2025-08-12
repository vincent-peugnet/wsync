package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Status() {
	files, err := os.ReadDir(repoPath)
	if err != nil {
		log.Fatalln("could not read folder:", err)
	}

	database := LoadDatabase()

	var untrackedFiles []string
	var trackedFiles []string
	var trackedModifiedFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".md" {
			id := strings.TrimSuffix(file.Name(), ".md")
			_, exist := database.Pages[id]
			if !exist {
				untrackedFiles = append(untrackedFiles, id)
			} else {
				trackedFiles = append(trackedFiles, id)
				modified, err := database.HasBeenModified(id)
				if err != nil {
					log.Fatalln("error:", err)
				}
				if modified {
					trackedModifiedFiles = append(trackedModifiedFiles, id)
				}
			}
		}
	}
	fmt.Println("📦️ Repo contains:")
	fmt.Println(len(trackedFiles), "tracked file(s)", trackedFiles)
	fmt.Println("  ↳ including", len(trackedModifiedFiles), "localy edited file(s)", trackedModifiedFiles)
	fmt.Println(len(untrackedFiles), "untracked file(s)", untrackedFiles)

}
