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

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/huh"
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
func (db *Database) HasBeenModified(id string) bool {
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

func (db Database) EditedPages() []string {
	var editedPages []string
	for id := range db.Pages {
		if db.HasBeenModified(id) {
			editedPages = append(editedPages, id)
		}
	}
	return editedPages
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

	_, exist := database.Pages[id]
	if exist && database.Pages[id].DateSync.After(page.DateModif) {
		return fmt.Errorf("already latest version")
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

	if err := client.Health(); err != nil {
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

func sync(args []string) {
	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.SetToken(string(token))

	if len(args) < 1 {
		if err := os.MkdirAll(repoPath, 0775); err != nil {
			log.Fatalln("could not create store:", err)
		}
		editedPages := database.EditedPages()

		for _, id := range editedPages {
			if err := Upload(client, database, id); err != nil {
				log.Println("could not upload page:", id, err)
			} else {
				fmt.Println("⬆️ page uploaded:", id)
			}
		}

		// TODO: try to download all modified pages from the server
	} else {
		id := args[0]

		if err := Upload(client, database, id); err != nil {
			log.Println("could not upload page:", err)
		}

		if err := Download(client, database, id); err != nil {
			log.Println("could not download updated page:", err)
		}
	}

	SaveDatabase(database)
}

func menu() {
	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What to do ?").
				Options(
					huh.NewOption("Init", "init"),
					huh.NewOption("nothing", "nothing"),
				).
				Value(&action),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	switch action {
	case "init":
		initialize(nil)
	default:
		fmt.Println("bye bye 👋")
	}
}

func main() {

	flag.StringVar(&repoPath, "C", ".", "set the working directory")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		switch args[0] {
		case "init":
			initialize(args[1:])
		case "sync":
			sync(args[1:])
		default:
			log.Fatalln("invalid sub command")
		}
	} else {
		menu()
	}

}
