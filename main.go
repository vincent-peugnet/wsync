package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
	"wsync/api"
)

const StorePath = "/tmp/wsync"
const DatabasePath = ".wsync/database.json"

type PageData struct {
	DateModif time.Time
	DateSync  time.Time
}

type Database struct {
	Pages map[string]*PageData
}

func NewDatabase() *Database {
	return &Database{
		Pages: make(map[string]*PageData),
	}
}

// HasBeenModified checks if given page has beed modified locally
func (db Database) HasBeenModified(id string) bool {
	pageData, exist := db.Pages[id]
	if !exist {
		return false // if not in db, we do not care
	}
	filename := id + ".md"
	filename = filepath.Join(StorePath, filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return false // we do not care of error
	}
	return stat.ModTime().After(pageData.DateSync)
}

func LoadDatabase() (*Database, error) {
	filename := filepath.Join(StorePath, DatabasePath)
	file, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewDatabase(), nil
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var database Database
	if err := decoder.Decode(&database); err != nil {
		return nil, fmt.Errorf("decode file: %w", err)
	}
	return &database, nil
}

func SaveDatabase(database *Database) error {
	filename := filepath.Join(StorePath, DatabasePath)
	if err := os.MkdirAll(filepath.Dir(filename), 0775); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(database); err != nil {
		return fmt.Errorf("encode file: %w", err)
	}
	return nil
}

func Download(co *api.Client, database *Database, id string) error {
	page, err := co.Get(id)
	if err != nil {
		return fmt.Errorf("get page: %w", err)
	}
	filename := id + ".md"
	filename = filepath.Join(StorePath, filename)

	if database.HasBeenModified(id) {
		return fmt.Errorf("local modification")
	}

	if err := os.WriteFile(filename, []byte(page.Content), 0664); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	pageData := &PageData{
		DateModif: page.DateModif,
		DateSync:  time.Now(),
	}
	database.Pages[id] = pageData

	return nil
}

func Upload(co *api.Client, database *Database, id string) error {
	pageData, exist := database.Pages[id]
	if !exist {
		return fmt.Errorf("ID not in database: %s", id)
	}

	filename := id + ".md"
	filename = filepath.Join(StorePath, filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if !database.HasBeenModified(id) {
		return fmt.Errorf("page has not been edited locally")
	}

	page := &api.Page{
		ID:        id,
		Content:   string(content),
		DateModif: pageData.DateModif,
	}

	if err := co.Update(page); err != nil {
		return fmt.Errorf("update page: %w", err)
	}

	database.Pages[id].DateSync = time.Now()

	return nil
}

func main() {

	if err := os.MkdirAll(StorePath, 0775); err != nil {
		log.Fatalln("could not create store:", err)
	}

	database, err := LoadDatabase()
	if err != nil {
		log.Fatalln("load database: %w", err)
	}

	id := os.Args[1]

	client := api.NewClient()

	if err := Upload(client, database, id); err != nil {
		log.Println("could not upload page:", err)
	}

	if err := Download(client, database, id); err != nil {
		log.Println("could not download updated page:", err)
	}

	if err := SaveDatabase(database); err != nil {
		log.Fatalln("save database: %w", err)
	}
}
