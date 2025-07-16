package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const BaseURL = "http://w.localhost"
const RememberMe = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyaWQiOiJ2aW5jZW50Iiwid3Nlc3Npb24iOiIxNjdlM2VhYTViYjRlNzQxNTQzZSJ9.3ziPbo4cWL8kqIC1Z8RIzUBbAFW6bo661v_3HKH8UOo"

type Page struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	DateModif time.Time `json:"datemodif"`
}

// ShortResponse is the API response data
type ShortResponse struct {
	Message string `json:"message"`
}

type Client struct {
	HTTPClient http.Client
}

func NewClient() *Client {

	cookie := &http.Cookie{
		Name:  "rememberme",
		Value: RememberMe,
	}
	u, _ := url.Parse(BaseURL)

	jar, _ := cookiejar.New(nil)
	jar.SetCookies(u, []*http.Cookie{cookie})
	return &Client{
		HTTPClient: http.Client{
			Jar: jar,
		},
	}

}

func (co *Client) Get(id string) (*Page, error) {
	url := fmt.Sprint(BaseURL, "/api/v0/page/", id)
	res, err := co.HTTPClient.Get(url)
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

func (co *Client) Update(page *Page) error {
	url := fmt.Sprint(BaseURL, "/api/v0/page/", page.ID, "/update")
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(page); err != nil {
		return fmt.Errorf("encode page: %w", err)
	}
	res, err := co.HTTPClient.Post(url, "application/json", buf)
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
