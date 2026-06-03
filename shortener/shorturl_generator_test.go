package shortener

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const UserId = "1234"

func TestShortLinkGeneration(t *testing.T) {
	intialLink1 := "https://www.google.com"
	shortLink1 := GenerateShortLink(intialLink1, UserId)

	intialLink2 := "https://chitransh.me"
	shortLink2 := GenerateShortLink(intialLink2, UserId)

	assert.Equal(t, shortLink1, "2aG8mL9X")
	assert.Equal(t, shortLink2, "9sXo7n8e")
}