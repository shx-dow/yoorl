package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/shx-dow/yoorl/handler"
	"github.com/shx-dow/yoorl/store"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "URL Shortener",
		})
	})

	r.POST("/create-short-url", func(c *gin.Context) {
		handler.CreateShortUrl(c)
	})

	r.GET("/:shortUrl", func(c *gin.Context) {
		handler.HandleShortUrlRedirect(c)
	})

	store.InitializeStore()

	port := os.Getenv("PORT")
	if port == "" {
		port = "9808"
	}

	err := r.Run(":" + port)
	if err != nil {
		panic(fmt.Sprintf("Failed to start the web server - Error: &v", err))
	}
}
