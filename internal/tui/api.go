package tui

import (
	"github.com/shx-dow/yoorl/internal/client"
	"github.com/shx-dow/yoorl/store"
)

type Client struct {
	*client.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{client.New(baseURL, apiKey)}
}

func (c *Client) ListURLs() ([]*store.UrlEntry, error) {
	return c.Client.ListURLs()
}

func (c *Client) CreateURL(longURL, alias string) (string, error) {
	result, err := c.Client.CreateURL(longURL, alias, "tui")
	if err != nil {
		return "", err
	}
	return result.ShortURL, nil
}

func (c *Client) DeleteURL(shortURL string) error {
	return c.Client.DeleteURL(shortURL)
}

func (c *Client) GetAnalytics(shortURL string) (*store.Analytics, error) {
	return c.Client.GetAnalytics(shortURL)
}
