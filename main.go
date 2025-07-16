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
const TokenPath = ".wsync/token"

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

func LoadDatabase() *Database {
	filename := filepath.Join(StorePath, DatabasePath)
	file, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewDatabase()
		}
		log.Fatalln("load database: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var database Database
	if err := decoder.Decode(&database); err != nil {
		log.Fatalln("load database: %w", err)
	}
	return &database
}

func SaveDatabase(database *Database) {
	filename := filepath.Join(StorePath, DatabasePath)
	if err := os.MkdirAll(filepath.Dir(filename), 0775); err != nil {
		log.Fatalln("save database: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalln("save database: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(database); err != nil {
		log.Fatalln("save database: %w", err)
	}
}

func SaveToken(token string) {
	filename := filepath.Join(StorePath, TokenPath)
	if err := os.MkdirAll(filepath.Dir(filename), 0775); err != nil {
		log.Fatalln("save token: %w", err)
	}

	if err := os.WriteFile(filename, []byte(token), 0640); err != nil {
		log.Fatalln("save token: %w", err)
	}
}

func LoadToken() string {
	filename := filepath.Join(StorePath, TokenPath)
	token, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln("load token: %w", err)
	}
	return string(token)
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

func login(args []string) {
	if len(args) < 2 {
		log.Fatalln("sync sub command need two arguments: user, password")
	}

	client := api.NewClient("http://w.localhost")

	token, err := client.Auth(args[0], args[1])
	if err != nil {
		log.Fatalln("login: %w", err)
	}

	SaveToken(token)

	log.Println("successfully logged in 🎉")
}

func sync(args []string) {
	if len(args) < 1 {
		log.Fatalln("sync sub command need a page ID argument")
	}
	id := args[0]

	if err := os.MkdirAll(StorePath, 0775); err != nil {
		log.Fatalln("could not create store:", err)
	}

	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient("http://w.localhost")
	client.SetToken(string(token))

	if err := Upload(client, database, id); err != nil {
		log.Println("could not upload page:", err)
	}

	if err := Download(client, database, id); err != nil {
		log.Println("could not download updated page:", err)
	}

	SaveDatabase(database)
}

func main() {

	args := os.Args[1:]
	if len(args) >= 1 {
		switch args[0] {
		case "login":
			login(args[1:])
		case "sync":
			sync(args[1:])
		default:
			log.Fatalln("invalid sub command")
		}
	} else {
		log.Fatalln("command need at least one argument")
	}

}
