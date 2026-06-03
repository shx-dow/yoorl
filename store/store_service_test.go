package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStoreInsertionAndRetrieval(t *testing.T) {
	SetStore(NewMemoryStore())

	intialLink := "https://www.google.com"
	userUUID := "1234"
	shortUrl := "abcd"

	SaveUrlMapping(shortUrl, intialLink, userUUID)

	retrievedUrl, err := RetrieveInitialUrl(shortUrl)

	assert.Nil(t, err)
	assert.Equal(t, intialLink, retrievedUrl)
}

func TestMemoryStoreMissingUrl(t *testing.T) {
	SetStore(NewMemoryStore())

	_, err := RetrieveInitialUrl("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStoreDelete(t *testing.T) {
	SetStore(NewMemoryStore())

	SaveUrlMapping("abc", "https://example.com", "user1")
	DeleteUrlMapping("abc")

	_, err := RetrieveInitialUrl("abc")
	assert.Error(t, err)
}
