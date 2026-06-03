package handler

import (
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/shx-dow/yoorl/shortener"
	"github.com/shx-dow/yoorl/store"
)

type UrlCreationRequest struct {
	LongUrl string `json:"long_url" binding:"required"`
	UserId  string `json:"user_id" binding:"required"`
}

func CreateShortUrl(c *gin.Context) {
	var creationRequest UrlCreationRequest
	if err := c.ShouldBindJSON(&creationRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := url.ParseRequestURI(creationRequest.LongUrl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url provided"})
		return
	}

	shortUrl := shortener.GenerateShortLink(creationRequest.LongUrl, creationRequest.UserId)
	store.SaveUrlMapping(shortUrl, creationRequest.LongUrl, creationRequest.UserId)

	host := os.Getenv("BASE_URL")
	if host == "" {
		host = "http://localhost:9808/"
	}

	c.JSON(200, gin.H{
		"message":   "short url created successfully",
		"short_url": host + shortUrl,
	})
}

func HandleShortUrlRedirect(c *gin.Context) {
	shortUrl := c.Param("shortUrl")
	initialUrl, err := store.RetrieveInitialUrl(shortUrl)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, initialUrl)
}
