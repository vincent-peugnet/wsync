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

var repoPath string  // local repo path
var force bool       // force pull and push operations
var interactive bool // interactive mode

const (
	DatabasePath = ".wsync/database.json"
	TokenPath    = ".wsync/token"
)

//___________________________ GENERIC ___________________________

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

type PageData struct {
	DateModif time.Time
	DateSync  time.Time
}

//___________________________ DATABASE ___________________________

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

// checks if given page has beed modified locally
func (db *Database) HasBeenModified(id string) (bool, error) {
	pageData, exist := db.Pages[id]
	if !exist {
		return false, fmt.Errorf("not found in tracked pages")
	}
	filename := GetPagePath(id)

	stat, err := os.Stat(filename)
	if err != nil {
		return false, fmt.Errorf("file not found: %w", err)
	}
	return stat.ModTime().After(pageData.DateSync), nil
}

// return list of localy edited pages
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

// Remove local page
// return true if local file was deleted
func (db *Database) removePage(id string) (bool, error) {
	modified, err := db.HasBeenModified(id)
	if err != nil {
		return false, fmt.Errorf("tried to untrack: %w", err)
	}
	delete(db.Pages, id) // untrack
	if modified {        // Do not delete the page if localy edited
		return false, nil
	} else {
		err := os.Remove(GetPagePath(id))
		if err != nil {
			return false, fmt.Errorf("tried to delete file: %w", err)
		}
		return true, nil
	}
}

func (db *Database) addPage(co *api.Client, id string) error {
	_, exist := db.Pages[id]
	if exist {
		return fmt.Errorf("page is already tracked")
	}

	filename := GetPagePath(id)

	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("local file already exist")
	}

	page, err := co.Get(id)
	if err != nil {
		return fmt.Errorf("tried to get page: %w", err)
	}

	if err := os.WriteFile(filename, []byte(page.Content), 0664); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	pageData := &PageData{
		DateModif: page.DateModif,
		DateSync:  time.Now(),
	}
	db.Pages[id] = pageData

	return nil
}

func (db *Database) pullPage(co *api.Client, id string, force bool) (bool, error) {
	page, err := co.Get(id)
	if err != nil {
		return false, fmt.Errorf("get page: %w", err)
	}

	filename := GetPagePath(id)

	pageData, exist := db.Pages[id]
	if !exist {
		if _, err := os.Stat(filename); err == nil {
			return false, fmt.Errorf("local file already exist")
		}
		return false, fmt.Errorf("untracked page")
	}

	if pageData.DateSync.After(page.DateModif) {
		return false, nil // already up to date
	}

	modified, err := db.HasBeenModified(id)
	if err != nil {
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
	db.Pages[id] = pageData

	return true, nil
}

func (db *Database) pushPage(co *api.Client, id string, force bool) (bool, error) {
	pageData, exist := db.Pages[id]
	if !exist {
		return false, fmt.Errorf("ID not in database: %s", id)
	}

	filename := GetPagePath(id)
	content, err := os.ReadFile(filename)
	if err != nil {
		return false, fmt.Errorf("read file: %w", err)
	}

	modified, err := db.HasBeenModified(id)
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
		if err != nil {
			return false, fmt.Errorf("update page: %w", err)
		}
		pageData.DateModif = updatedPage.DateModif
		pageData.DateSync = time.Now()
	}

	return modified, nil
}

func (db *Database) syncPage(co *api.Client, id string) (bool, error) {
	pushed, pushErr := db.pushPage(co, id, false)
	if pushErr != nil {
		return false, pushErr
	}
	pulled, pullErr := db.pullPage(co, id, false)
	if pullErr != nil {
		return false, pullErr
	}

	return pushed || pulled, nil
}

//___________________________ SUB COMMANDS ___________________________

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
		pushed, err := database.pushPage(client, id, force)
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
		fmt.Println("✅ all tracked pages are already up to date")
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
		pulled, err := database.pullPage(client, id, force)
		if err != nil {
			fmt.Printf("❌ could not pull page: %q: %v\n", id, err)
			i++
		}
		if pulled {
			fmt.Printf("⬇️  pulled page %q\n", id)
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all tracked pages are already up to date")
	}

	SaveDatabase(database)
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

func Sync(args []string) {
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
		synced, err := database.syncPage(client, id)
		if interactive && errors.Is(err, api.ErrConflict) {
			conflict(database, client, id)
			i++
		} else if err != nil {
			fmt.Printf("❌ could not sync page %q: %v\n", id, err)
			i++
		} else if synced {
			fmt.Printf("🔃 synced page %q\n", id)
			i++
		}
	}
	if i == 0 {
		fmt.Println("✅ all tracked pages are already in sync")
	}
	SaveDatabase(database)

}

func Initialize(args []string) {

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

func Add(args []string) {
	if len(args) < 1 {
		log.Fatalln("add sub-command need at least one page id argument")
	}

	database := LoadDatabase()
	token := LoadToken()

	client := api.NewClient(database.Config.BaseURL)
	client.Token = token

	for _, id := range args {
		err := database.addPage(client, id)
		if err != nil {
			fmt.Printf("❌ error while adding page %q: %v\n", id, err)
		} else {
			fmt.Printf("⭐️ added new tracked page %q, created new file %q\n", id, GetPagePath(id))
		}
	}

	SaveDatabase(database)
}

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

// ___________________________ INTERFACE ___________________________

func menu() {
	interactive = true

	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What to do ?").
				Options(
					huh.NewOption("Status", "status"),
					huh.NewOption("Sync", "sync"),
					huh.NewOption("Push", "push"),
					huh.NewOption("Pull", "pull"),
					huh.NewOption("List", "list"),
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
		Initialize(nil)
	case "status":
		Status()
	case "list":
		List()
	case "sync":
		Sync(nil)
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
	flag.BoolVar(&interactive, "i", false, "disable interactive mode")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		switch args[0] {
		case "init":
			Initialize(args[1:])
		case "sync":
			Sync(args[1:])
		case "pull":
			Pull(args[1:])
		case "push":
			Push(args[1:])
		case "remove":
			Remove(args[1:])
		case "add":
			Add(args[1:])
		case "list":
			List()
		case "status":
			Status()
		default:
			log.Fatalln("invalid sub command")
		}
	} else {
		menu()
	}

}
