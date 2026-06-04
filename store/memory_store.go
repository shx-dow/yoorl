package store

import (
	"fmt"
	"sync"
)

type MemoryStore struct {
	mu       sync.RWMutex
	data     map[string]string
	clicks   map[string][]ClickEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:   make(map[string]string),
		clicks: make(map[string][]ClickEvent),
	}
}

func (m *MemoryStore) SaveUrlMapping(shortUrl string, originalUrl string, userId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[shortUrl] = originalUrl
}

func (m *MemoryStore) RetrieveInitialUrl(shortUrl string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	original, ok := m.data[shortUrl]
	if !ok {
		return "", fmt.Errorf("short URL %s not found", shortUrl)
	}
	return original, nil
}

func (m *MemoryStore) DeleteUrlMapping(shortUrl string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, shortUrl)
	delete(m.clicks, shortUrl)
}

func (m *MemoryStore) RecordClick(shortUrl string, event ClickEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clicks[shortUrl] = append([]ClickEvent{event}, m.clicks[shortUrl]...)
	if len(m.clicks[shortUrl]) > 100 {
		m.clicks[shortUrl] = m.clicks[shortUrl][:100]
	}
}

func (m *MemoryStore) GetAnalytics(shortUrl string) (*Analytics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clicks := m.clicks[shortUrl]
	recent := make([]ClickEvent, len(clicks))
	copy(recent, clicks)

	return &Analytics{
		ShortUrl:     shortUrl,
		TotalClicks:  int64(len(recent)),
		RecentClicks: recent,
	}, nil
}
