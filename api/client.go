package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrConflict = errors.New("conflict")

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

type Client struct {
	BaseURL string
	Token   string
}

func NewClient(baseURL string) *Client {

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := c.BaseURL + path
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}
	request.Header.Set("User-Agent", "wsync")
	return request, nil
}

func (c *Client) post(path string, content any) (*http.Response, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(&content); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	request, err := c.newRequest(http.MethodPost, path, buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(request)
}

func (c *Client) get(path string) (*http.Response, error) {
	request, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return http.DefaultClient.Do(request)
}

func (c *Client) Get(id string) (*Page, error) {
	path := fmt.Sprint("/api/v0/page/", id)
	res, err := c.get(path)
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

// try to update page
// if response is 409, returned error is ErrConflict
func (c *Client) Update(page *Page, force bool) (*Page, error) {
	var query string
	if force {
		v := url.Values{}
		v.Set("force", "1")
		query = "?" + v.Encode()
	}
	path := fmt.Sprint("/api/v0/page/", page.ID, "/update", query)
	res, err := c.post(path, page)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := chekResponse(res); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	var updatedPage Page
	if err := decoder.Decode(&updatedPage); err != nil {
		return nil, fmt.Errorf("decode updated page: %w", err)
	}
	return &updatedPage, nil
}

func (c *Client) List() ([]string, error) {
	res, err := c.get("/api/v0/pages/list")
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
	res, err := c.post("/api/v0/pages/query", options)
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

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	res, err := c.post("/api/v0/auth", credentials)
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
	res, err := c.get("/api/v0/health")
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
		msg := fmt.Sprintf("status code: %d", res.StatusCode)
		if err := decoder.Decode(&shortResponse); err == nil {
			msg = fmt.Sprintf("status code: %d - %s", res.StatusCode, shortResponse.Message)
		}
		switch res.StatusCode {
		case 409:
			return fmt.Errorf("%w: %v", ErrConflict, msg)
		default:
			return errors.New(msg)
		}
	}

	return nil
}
