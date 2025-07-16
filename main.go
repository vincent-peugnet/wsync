package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
	"wsync/api"
)

var repoPath string

const (
	DatabasePath = ".wsync/database.json"
	TokenPath    = ".wsync/token"
)

type PageData struct {
	DateModif time.Time
	DateSync  time.Time
}

type Database struct {
	Pages  map[string]*PageData
	Config struct {
		BaseURL string
	}
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
	filename = filepath.Join(repoPath, filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return false // we do not care of error
	}
	return stat.ModTime().After(pageData.DateSync)
}

func LoadDatabase() *Database {
	filename := filepath.Join(repoPath, DatabasePath)
	file, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewDatabase()
		}
		log.Fatalln("load database:", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var database Database
	if err := decoder.Decode(&database); err != nil {
		log.Fatalln("load database:", err)
	}
	return &database
}

func SaveDatabase(database *Database) {
	filename := filepath.Join(repoPath, DatabasePath)
	if err := os.MkdirAll(filepath.Dir(filename), 0775); err != nil {
		log.Fatalln("save database:", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalln("save database:", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(database); err != nil {
		log.Fatalln("save database:", err)
	}
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

func Download(co *api.Client, database *Database, id string) error {
	page, err := co.Get(id)
	if err != nil {
		return fmt.Errorf("get page: %w", err)
	}
	filename := id + ".md"
	filename = filepath.Join(repoPath, filename)

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
	filename = filepath.Join(repoPath, filename)
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

func initialize(args []string) {

	files, err := os.ReadDir(repoPath)
	if err != nil {
		log.Fatalln("read folder:", err)
	}

	if len(files) > 0 {
		log.Fatalln("directory is not empty")
	}

	if len(args) < 1 {
		log.Fatalln("init sub command need an URL argument")
	}
	baseURL := args[0]
	client := api.NewClient(baseURL)

	if err := client.Health(); err != nil {
		log.Fatalln("❌ERROR: there is no W at this adress.", err)
	}

	database := LoadDatabase()
	database.Config.BaseURL = baseURL
	SaveDatabase(database)

	fmt.Println("⭐️ repository successfully initalized")
}

func login(args []string) {
	if len(args) < 2 {
		log.Fatalln("sync sub command need two arguments: USER PASSWORD")
	}

	database := LoadDatabase()

	client := api.NewClient(database.Config.BaseURL)

	token, err := client.Auth(args[0], args[1])
	if err != nil {
		log.Fatalln("login:", err)
	}

	SaveToken(token)

	log.Println("successfully logged in 🎉")
}

func sync(args []string) {
	if len(args) < 1 {
		log.Fatalln("sync sub command need a page ID argument")
	}
	id := args[0]

	if err := os.MkdirAll(repoPath, 0775); err != nil {
		log.Fatalln("could not create store:", err)
	}

	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
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

	flag.StringVar(&repoPath, "C", ".", "set the working directory")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		switch args[0] {
		case "init":
			initialize(args[1:])
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
