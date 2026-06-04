package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/shx-dow/yoorl/handler"
	"github.com/shx-dow/yoorl/internal/middleware"
	"github.com/shx-dow/yoorl/store"
)

func main() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().
		Timestamp().
		Logger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store.InitializeStore()

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		gin.Logger(),
		middleware.Recovery(log),
		middleware.CORS(),
	)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	{
		v1.POST("/urls", handler.CreateShortUrl)
		v1.DELETE("/urls/:shortUrl", handler.DeleteShortUrl)
	}

	r.GET("/r/:shortUrl", handler.HandleShortUrlRedirect)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Info().Str("port", port).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed to start")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}
