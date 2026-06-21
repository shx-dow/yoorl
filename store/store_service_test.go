package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStoreInsertionAndRetrieval(t *testing.T) {
	SetStore(NewMemoryStore())

	initialLink := "https://www.google.com"
	userUUID := "1234"
	shortUrl := "abcd"

	err := SaveUrlMapping(shortUrl, initialLink, userUUID)
	assert.Nil(t, err)

	retrievedUrl, err := RetrieveInitialUrl(shortUrl)

	assert.Nil(t, err)
	assert.Equal(t, initialLink, retrievedUrl)
}

func TestMemoryStoreMissingUrl(t *testing.T) {
	SetStore(NewMemoryStore())

	_, err := RetrieveInitialUrl("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStoreDelete(t *testing.T) {
	SetStore(NewMemoryStore())

	assert.Nil(t, SaveUrlMapping("abc", "https://example.com", "user1"))
	assert.Nil(t, DeleteUrlMapping("abc"))

	_, err := RetrieveInitialUrl("abc")
	assert.Error(t, err)
}

func TestMemoryStoreAnalytics(t *testing.T) {
	SetStore(NewMemoryStore())

	assert.Nil(t, SaveUrlMapping("abc", "https://example.com", "user1"))

	stats, err := GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), stats.TotalClicks)

	assert.Nil(t, RecordClick("abc", ClickEvent{IP: "1.2.3.4"}))
	assert.Nil(t, RecordClick("abc", ClickEvent{IP: "5.6.7.8"}))

	stats, err = GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(2), stats.TotalClicks)
	assert.Equal(t, "5.6.7.8", stats.RecentClicks[0].IP)
	assert.Equal(t, "1.2.3.4", stats.RecentClicks[1].IP)
}

func TestMemoryStoreAnalyticsDeleteClearsClicks(t *testing.T) {
	SetStore(NewMemoryStore())

	assert.Nil(t, SaveUrlMapping("abc", "https://example.com", "user1"))
	assert.Nil(t, RecordClick("abc", ClickEvent{IP: "1.2.3.4"}))

	assert.Nil(t, DeleteUrlMapping("abc"))

	stats, err := GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), stats.TotalClicks)
}
