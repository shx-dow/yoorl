package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testStoreService = &StorageService{}

func init() {
	testStoreService = InitializeStore()
}

func TestStoreInit(t *testing.T) {
	assert.True(t, testStoreService.redisClient != nil)
}

func TestInsertionAndRetrieval(t *testing.T) {
	intialLink := "https://www.google.com"
	userUUID := "1234"
	shortUrl := "abcd"

	SaveUrlMapping(shortUrl, intialLink, userUUID)

	retrievedUrl, err := RetrieveInitialUrl(shortUrl)

	assert.Nil(t, err)
	assert.Equal(t, intialLink, retrievedUrl)
}