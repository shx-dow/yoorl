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

	assert.Equal(t, "YZfD8AAc", shortLink1)
	assert.Equal(t, "SiB56pzL", shortLink2)
}