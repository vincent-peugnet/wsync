package main

import (
	"encoding/json"
	"fmt"
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

type Page struct {
	ID        string
	Title     string
	Content   string
	Datemodif time.Time
}

type PageData struct {
	ID        string
	Datemodif time.Time
}

type Connection struct {
	Client http.Client
}

func (co *Connection) GetPage(id string) (*Page, error) {
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
		return nil, err
	}
	return &page, nil
}

func (co *Connection) Download(id string) error {
	page, err := co.GetPage(id)
	if err != nil {
		return fmt.Errorf("get page: %w", err)
	}
	filename := id + ".md"
	filename = filepath.Join(StorePath, filename)
	if err := os.WriteFile(filename, []byte(page.Content), 0664); err != nil {
		return fmt.Errorf("write file: %w", err)
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
		log.Fatalln("could not retrieve page:", err)
	}
}
