package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vincent-peugnet/wsync/api"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/huh"
)

var repoPath string
var force bool

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
func (db *Database) HasBeenModified(id string) (bool, error) {
	pageData, exist := db.Pages[id]
	if !exist {
		return false, fmt.Errorf("page not found in local database: %q", id)
	}
	filename := id + ".md"
	filename = filepath.Join(repoPath, filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return false, fmt.Errorf("file not found")
	}
	return stat.ModTime().After(pageData.DateSync), nil
}

func (db Database) EditedPages() []string {
	var editedPages []string
	for id := range db.Pages {
		modified, err := db.HasBeenModified(id)
		if err == nil && modified {
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

// func syncPage(co *api.Client, database *Database, id string) (string, error) {
// 	pushed, pushErr := push(co, database, id)
// 	pulled, pullErr := pull(co, database, id)

// 	if pullErr != nil || pushErr != nil {
// 		return fmt.Errorf("sync: %w %w", pushErr, pullErr)
// 	}
// 	var message string
// 	if pushed {

// 	}
// }

func pullPage(co *api.Client, database *Database, id string) (bool, error) {
	page, err := co.Get(id)
	if err != nil {
		return false, fmt.Errorf("get page: %w", err)
	}

	pageData, exist := database.Pages[id]
	if exist && pageData.DateSync.After(page.DateModif) {
		return false, nil // already up to date
	}

	filename := id + ".md"
	filename = filepath.Join(repoPath, filename)

	modified, err := database.HasBeenModified(id)
	if exist && err != nil {
		return false, err
	}
	if modified && !force {
		return false, fmt.Errorf("local modification")
	}

	if err := os.WriteFile(filename, []byte(page.Content), 0664); err != nil {
		return false, fmt.Errorf("write file: %w", err)
	}

	pageData = &PageData{
		DateModif: page.DateModif,
		DateSync:  time.Now(),
	}
	database.Pages[id] = pageData

	return true, nil
}

func pushPage(co *api.Client, database *Database, id string) (bool, error) {
	pageData, exist := database.Pages[id]
	if !exist {
		return false, fmt.Errorf("ID not in database: %s", id)
	}

	filename := id + ".md"
	filename = filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		return false, fmt.Errorf("read file: %w", err)
	}

	modified, err := database.HasBeenModified(id)
	if err != nil {
		return false, err
	}
	if modified {
		page := &api.Page{
			ID:        id,
			Content:   string(content),
			DateModif: pageData.DateModif,
		}

		updatedPage, err := co.Update(page, force)
		if err := err; err != nil {
			return false, fmt.Errorf("update page: %w", err)
		}
		pageData.DateModif = updatedPage.DateModif
	}
	pageData.DateSync = time.Now()

	return modified, nil
}

func Push(args []string) {
	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	var pages []string
	if len(args) > 0 {
		pages = args
	} else {
		pages = slices.Collect(maps.Keys(database.Pages))
	}
	var i int
	for _, id := range pages {
		pushed, err := pushPage(client, database, id)
		if err != nil {
			fmt.Printf("❌ could not push page: %q %v\n", id, err)
			i++
		}
		if pushed {
			fmt.Printf("⬆️  pushed page %q - ", id)
			fmt.Print(database.Config.BaseURL + "/" + id + "\n")
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all pages are already up to date")
	}
	SaveDatabase(database)
}

func Pull(args []string) {
	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	var pages []string
	if len(args) > 0 {
		pages = args
	} else {
		pages = slices.Collect(maps.Keys(database.Pages))
	}
	var i int
	for _, id := range pages {
		pushed, err := pullPage(client, database, id)
		if err != nil {
			fmt.Printf("❌ could not pull page: %q: %v\n", id, err)
			i++
		}
		if pushed {
			fmt.Printf("⬇️  pulled page %q\n", id)
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all pages are already up to date")
	}

	SaveDatabase(database)
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

func list() {
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
		options = append(options, huh.NewOption(id, id))
	}

	var selectedids []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select pages").
				Options(options...).
				Value(&selectedids).
				WithHeight(10),
		),
	)
	if err := form.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(selectedids)
}

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

func menu() {
	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What to do ?").
				Options(
					huh.NewOption("Init", "init"),
					huh.NewOption("Push", "push"),
					huh.NewOption("Pull", "pull"),
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
	case "push":
		Push(nil)
	case "pull":
		Pull(nil)
	default:
		fmt.Println("bye bye 👋")
	}
}

func main() {

	flag.StringVar(&repoPath, "C", ".", "set the working directory")
	flag.BoolVar(&force, "F", false, "force push or pull")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		switch args[0] {
		case "init":
			initialize(args[1:])
		case "pull":
			Pull(args[1:])
		case "push":
			Push(args[1:])
		case "list":
			list()
		case "status":
			Status()
		default:
			log.Fatalln("invalid sub command")
		}
	} else {
		menu()
	}

}
