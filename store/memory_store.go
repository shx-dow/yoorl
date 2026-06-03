package store

import (
	"fmt"
	"sync"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
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
}
