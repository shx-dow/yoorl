package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shx-dow/yoorl/store"
)

type Client struct {
	BaseURL string
	APIKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  normalizeAPIKey(apiKey),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func normalizeAPIKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if key, _, ok := strings.Cut(apiKey, ":"); ok {
		return key
	}
	return apiKey
}

func (c *Client) Health() error {
	resp, err := c.http.Get(c.BaseURL + "/health")
	if err != nil {
		return fmt.Errorf("connect to %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connect to %s: health check returned HTTP %d", c.BaseURL, resp.StatusCode)
	}
	return nil
}

type apiResponse struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func (c *Client) doRequest(method, path string, body interface{}) (apiResponse, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return apiResponse{}, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return apiResponse{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return apiResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return apiResponse{}, fmt.Errorf("parse response: %w", err)
	}

	if resp.StatusCode >= 400 && apiResp.Error == "" {
		apiResp.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return apiResp, nil
}

func (c *Client) ListURLs() ([]*store.UrlEntry, error) {
	resp, err := c.doRequest("GET", "/v1/urls", nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("API error: %s", resp.Error)
	}

	var urls []*store.UrlEntry
	if err := json.Unmarshal(resp.Data, &urls); err != nil {
		return nil, fmt.Errorf("parse URLs: %w", err)
	}
	return urls, nil
}

func (c *Client) CreateURL(longURL, alias string) (string, error) {
	body := map[string]string{
		"long_url": longURL,
		"user_id":  "tui",
	}
	if alias != "" {
		body["custom_alias"] = alias
	}

	resp, err := c.doRequest("POST", "/v1/urls", body)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("API error: %s", resp.Error)
	}

	var data struct {
		ShortUrl string `json:"short_url"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return data.ShortUrl, nil
}

func (c *Client) DeleteURL(shortURL string) error {
	path := "/v1/urls/" + strings.TrimLeft(shortURL, "/")
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("API error: %s", resp.Error)
	}
	return nil
}

func (c *Client) GetAnalytics(shortURL string) (*store.Analytics, error) {
	path := "/v1/urls/" + strings.TrimLeft(shortURL, "/") + "/analytics"
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("API error: %s", resp.Error)
	}

	var a store.Analytics
	if err := json.Unmarshal(resp.Data, &a); err != nil {
		return nil, fmt.Errorf("parse analytics: %w", err)
	}
	return &a, nil
}
