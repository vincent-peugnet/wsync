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

	"github.com/vincent-peugnet/wsync/api"
)

type PageData struct {
	Version   int
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

	if err := os.WriteFile(filename, []byte(page.Primary()), 0664); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	pageData := &PageData{
		Version:   page.Version,
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

	if err := os.WriteFile(filename, []byte(page.Primary()), 0664); err != nil {
		return false, fmt.Errorf("write file: %w", err)
	}

	pageData = &PageData{
		Version:   page.Version,
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
			Version:   pageData.Version,
			DateModif: pageData.DateModif,
		}
		page.SetPrimary(string(content))

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
