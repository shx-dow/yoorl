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

	SaveUrlMapping(shortUrl, initialLink, userUUID)

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

	SaveUrlMapping("abc", "https://example.com", "user1")
	DeleteUrlMapping("abc")

	_, err := RetrieveInitialUrl("abc")
	assert.Error(t, err)
}

func TestMemoryStoreAnalytics(t *testing.T) {
	SetStore(NewMemoryStore())

	SaveUrlMapping("abc", "https://example.com", "user1")

	stats, err := GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), stats.TotalClicks)

	RecordClick("abc", ClickEvent{IP: "1.2.3.4"})
	RecordClick("abc", ClickEvent{IP: "5.6.7.8"})

	stats, err = GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(2), stats.TotalClicks)
	assert.Equal(t, "5.6.7.8", stats.RecentClicks[0].IP)
	assert.Equal(t, "1.2.3.4", stats.RecentClicks[1].IP)
}

func TestMemoryStoreAnalyticsDeleteClearsClicks(t *testing.T) {
	SetStore(NewMemoryStore())

	SaveUrlMapping("abc", "https://example.com", "user1")
	RecordClick("abc", ClickEvent{IP: "1.2.3.4"})

	DeleteUrlMapping("abc")

	stats, err := GetAnalytics("abc")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), stats.TotalClicks)
}
