package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Page struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	DateModif time.Time `json:"datemodif"`
}

type Options struct {
	Fields        []string  `json:"fields,omitempty"`
	SortBy        string    `json:"sortby"`
	Order         int       `json:"order"`
	TagFilter     []string  `json:"tagfilter,omitempty"`
	TagCompare    string    `json:"tagcompare"`
	TagNot        bool      `json:"tagnot"`
	AuthorFilter  []string  `json:"authorfilter,omitempty"`
	AuthorCompare string    `json:"authorcompare"`
	Secure        int       `json:"secure"`
	LinkTo        string    `json:"linkto,omitempty"`
	Invert        bool      `json:"invert"`
	Limit         int       `json:"limit"`
	Since         time.Time `json:"since,omitzero"`
	Until         time.Time `json:"until,omitzero"`
}

func DefaultOptions() *Options {
	return &Options{
		SortBy:        "id",
		Order:         1,
		TagCompare:    "AND",
		TagNot:        false,
		AuthorCompare: "AND",
		Secure:        4,
		Invert:        false,
		Limit:         0,
	}
}

// ShortResponse is the API response data
type ShortResponse struct {
	Message string `json:"message"`
}

type Transport struct {
	token string
	trip  http.RoundTripper
}

func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t.token != "" {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}
	request.Header.Set("User-Agent", "wsync")
	return t.trip.RoundTrip(request)
}

type Client struct {
	BaseURL    string
	transport  *Transport
	httpClient http.Client
}

func NewClient(baseURL string) *Client {
	transport := &Transport{
		trip: http.DefaultTransport,
	}

	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		transport: transport,
		httpClient: http.Client{
			Transport: transport,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.transport.token = token
}

func (c *Client) Get(id string) (*Page, error) {
	url := fmt.Sprint(c.BaseURL, "/api/v0/page/", id)
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	var page Page
	if err := decoder.Decode(&page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}
	return &page, nil
}

func (c *Client) Update(page *Page) error {
	url := fmt.Sprint(c.BaseURL, "/api/v0/page/", page.ID, "/update")
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(page); err != nil {
		return fmt.Errorf("encode page: %w", err)
	}
	res, err := c.httpClient.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return err
	}

	return nil
}

func (c *Client) List() ([]string, error) {
	url := fmt.Sprint(c.BaseURL, "/api/v0/pages/list")
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	var list struct {
		Pages []string `json:"pages"`
	}
	if err := decoder.Decode(&list); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	return list.Pages, nil
}

func (c *Client) Query(options *Options) (map[string]*Page, error) {
	url := fmt.Sprint(c.BaseURL, "/api/v0/pages/query")

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(options); err != nil {
		return nil, fmt.Errorf("encode query options: %w", err)
	}

	res, err := c.httpClient.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return nil, err
	}

	var result struct {
		Pages map[string]*Page `json:"pages"`
	}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode page list: %w", err)
	}
	return result.Pages, nil
}

func (c *Client) Auth(username string, password string) (string, error) {
	url := fmt.Sprint(c.BaseURL, "/api/v0/auth")

	credentials := fmt.Sprintf(`{"username": %q, "password": %q}`, username, password)

	res, err := c.httpClient.Post(url, "application/json", strings.NewReader(credentials))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return "", err
	}

	var tokenResponse struct {
		Token string `json:"token"`
	}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return tokenResponse.Token, nil
}

func (c *Client) Health() error {
	url := fmt.Sprint(c.BaseURL, "/api/v0/health")

	res, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return chekResponse(res)
}

func chekResponse(res *http.Response) error {
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
