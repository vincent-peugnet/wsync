package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const BaseURL = "http://w.localhost"
const RememberMe = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyaWQiOiJ2aW5jZW50Iiwid3Nlc3Npb24iOiJlMDEzYTZmZDc3OWZkMGY4ZDRhZSJ9.2yhdbt1UjNSvA0FxF-u5bKThFgTo_ArG55uhV-xjLhI"
const StorePath = "/tmp/wsync"
const DatabasePath = ".wsync/database.json"

type Page struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	DateModif time.Time `json:"datemodif"`
}

// ShortResponse is the API response data
type ShortResponse struct {
	Message string `json:"message"`
}

type PageData struct {
	DateModif    time.Time
	DateDownload time.Time
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
	return stat.ModTime().After(pageData.DateDownload)
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

type Connection struct {
	Client http.Client
}

func (co *Connection) Get(id string) (*Page, error) {
	url := fmt.Sprint(BaseURL, "/api/v0/page/", id)
	res, err := co.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	var page Page
	if err := decoder.Decode(&page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}
	return &page, nil
}

func (co *Connection) Update(page *Page) error {
	url := fmt.Sprint(BaseURL, "/api/v0/page/", page.ID, "/update")
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(page); err != nil {
		return fmt.Errorf("encode page: %w", err)
	}
	res, err := co.Client.Post(url, "application/json", buf)
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		decoder := json.NewDecoder(res.Body)
		var shortResponse ShortResponse
		if err := decoder.Decode(&shortResponse); err == nil {
			return fmt.Errorf("status code: %d - %s", res.StatusCode, shortResponse.Message)
		}
		return fmt.Errorf("status code: %d", res.StatusCode)
	}

	return nil
}

func (co *Connection) Download(id string) error {
	page, err := co.Get(id)
	if err != nil {
		return fmt.Errorf("get page: %w", err)
	}
	filename := id + ".md"
	filename = filepath.Join(StorePath, filename)

	database, err := LoadDatabase()
	if err != nil {
		return fmt.Errorf("load database: %w", err)
	}
	if database.HasBeenModified(id) {
		return fmt.Errorf("local modification")
	}

	if err := os.WriteFile(filename, []byte(page.Content), 0664); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	pageData := &PageData{
		DateModif:    page.DateModif,
		DateDownload: time.Now(),
	}
	database.Pages[id] = pageData

	if err := SaveDatabase(database); err != nil {
		return fmt.Errorf("save database: %w", err)
	}

	return nil
}

func main() {

	if err := os.MkdirAll(StorePath, 0775); err != nil {
		log.Fatalln("could not create store:", err)
	}

	cookie := &http.Cookie{
		Name:  "rememberme",
		Value: RememberMe,
	}
	u, err := url.Parse(BaseURL)
	if err != nil {
		log.Fatalln("wrong BaseURL:", err)
	}

	jar, _ := cookiejar.New(nil)
	jar.SetCookies(u, []*http.Cookie{cookie})

	id := os.Args[1]
	co := Connection{
		Client: http.Client{
			Jar: jar,
		},
	}

	if err := co.Download(id); err != nil {
		log.Fatalln("could not download page:", err)
	}
}
