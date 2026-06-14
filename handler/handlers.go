package handler

import (
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shx-dow/yoorl/internal/analytics"
	"github.com/shx-dow/yoorl/shortener"
	"github.com/shx-dow/yoorl/store"
)

var clickTracker *analytics.Tracker

func SetTracker(t *analytics.Tracker) {
	clickTracker = t
}

type createUrlRequest struct {
	LongUrl     string `json:"long_url" binding:"required"`
	UserId      string `json:"user_id" binding:"required"`
	CustomAlias string `json:"custom_alias,omitempty"`
}

type createUrlResponse struct {
	ShortUrl string `json:"short_url"`
	LongUrl  string `json:"long_url"`
}

func CreateShortUrl(c *gin.Context) {
	var req createUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if _, err := url.ParseRequestURI(req.LongUrl); err != nil {
		badRequest(c, "invalid URL provided")
		return
	}

	var shortUrl string
	if req.CustomAlias != "" {
		if len(req.CustomAlias) < 3 || len(req.CustomAlias) > 32 {
			badRequest(c, "custom alias must be between 3 and 32 characters")
			return
		}

		if existing, _ := store.RetrieveInitialUrl(req.CustomAlias); existing != "" {
			conflict(c, "custom alias already in use")
			return
		}
		shortUrl = req.CustomAlias
	} else {
		shortUrl = shortener.GenerateShortLink(req.LongUrl, req.UserId)
	}

	store.SaveUrlMapping(shortUrl, req.LongUrl, req.UserId)

	host := os.Getenv("BASE_URL")
	if host == "" {
		host = "http://localhost:8080/"
	}

	created(c, "short url created successfully", createUrlResponse{
		ShortUrl: host + "r/" + shortUrl,
		LongUrl:  req.LongUrl,
	})
}

func HandleShortUrlRedirect(c *gin.Context) {
	shortUrl := c.Param("shortUrl")
	initialUrl, err := store.RetrieveInitialUrl(shortUrl)
	if err != nil {
		notFound(c, "short URL not found")
		return
	}

	if clickTracker != nil {
		clickTracker.Track(shortUrl, store.ClickEvent{
			Timestamp: time.Now(),
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Referer:   c.Request.Referer(),
		})
	}

	c.Redirect(302, initialUrl)
}

func DeleteShortUrl(c *gin.Context) {
	shortUrl := c.Param("shortUrl")
	store.DeleteUrlMapping(shortUrl)
	success(c, 200, "short url deleted successfully", nil)
}

type updateUrlRequest struct {
	LongUrl string `json:"long_url" binding:"required"`
}

func UpdateShortUrl(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	if _, err := store.RetrieveInitialUrl(shortUrl); err != nil {
		notFound(c, "short URL not found")
		return
	}

	var req updateUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	if _, err := url.ParseRequestURI(req.LongUrl); err != nil {
		badRequest(c, "invalid URL provided")
		return
	}

	store.SaveUrlMapping(shortUrl, req.LongUrl, "")

	success(c, 200, "short url updated successfully", createUrlResponse{
		ShortUrl: os.Getenv("BASE_URL") + "r/" + shortUrl,
		LongUrl:  req.LongUrl,
	})
}

func HandleListUrls(c *gin.Context) {
	userId, _ := c.Get("user_id")
	userIdStr, _ := userId.(string)

	if qp := c.Query("user_id"); qp != "" {
		userIdStr = qp
	}

	urls, err := store.ListUrls(userIdStr)
	if err != nil {
		internalError(c, "failed to list URLs")
		return
	}

	success(c, 200, "", urls)
}

func HandleGetAnalytics(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	if _, err := store.RetrieveInitialUrl(shortUrl); err != nil {
		notFound(c, "short URL not found")
		return
	}

	stats, err := store.GetAnalytics(shortUrl)
	if err != nil {
		internalError(c, "failed to retrieve analytics")
		return
	}

	success(c, 200, "", stats)
}
