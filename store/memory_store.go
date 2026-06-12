package store

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]string
	entries map[string]*UrlEntry
	clicks  map[string][]ClickEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:    make(map[string]string),
		entries: make(map[string]*UrlEntry),
		clicks:  make(map[string][]ClickEvent),
	}
}

func (m *MemoryStore) SaveUrlMapping(shortUrl string, originalUrl string, userId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[shortUrl] = originalUrl

	existing, ok := m.entries[shortUrl]
	if !ok {
		m.entries[shortUrl] = &UrlEntry{
			ShortUrl:  shortUrl,
			LongUrl:   originalUrl,
			UserId:    userId,
			CreatedAt: time.Now(),
		}
	} else {
		existing.LongUrl = originalUrl
		existing.UserId = userId
	}
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
	delete(m.entries, shortUrl)
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

func (m *MemoryStore) ListUrls(userId string) ([]*UrlEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]*UrlEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if userId != "" && e.UserId != userId {
			continue
		}
		clicks := len(m.clicks[e.ShortUrl])
		entry := *e
		entry.TotalClicks = int64(clicks)
		entries = append(entries, &entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}
