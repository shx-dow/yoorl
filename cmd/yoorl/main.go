package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/shx-dow/yoorl/handler"
	"github.com/shx-dow/yoorl/internal/analytics"
	"github.com/shx-dow/yoorl/internal/client"
	"github.com/shx-dow/yoorl/internal/middleware"
	"github.com/shx-dow/yoorl/store"
	"github.com/shx-dow/yoorl/internal/tui"
	qrcode "github.com/skip2/go-qrcode"
)

const defaultBaseURL = "http://localhost:8080"

func baseURL() string {
	if v := os.Getenv("YOORL_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBaseURL
}

func newClient() *client.Client {
	return client.New(baseURL(), os.Getenv("YOORL_API_KEY"))
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "server":
		runServer()
	case "create":
		cmdCreate(args)
	case "delete":
		cmdDelete(args)
	case "update":
		cmdUpdate(args)
	case "analytics":
		cmdAnalytics(args)
	case "qr":
		cmdQr(args)
	case "tui":
		apiKey := os.Getenv("YOORL_API_KEY")
		if err := tui.StartTUI(baseURL(), apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: yoorl <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  server                Start the HTTP server")
	fmt.Println("  create [--alias,-a <alias>] [--user,-u <id>] <url>")
	fmt.Println("                           Create a short URL")
	fmt.Println("  delete <short-url>     Delete a short URL")
	fmt.Println("  update <short-url> <new-url>")
	fmt.Println("                           Update a short URL's destination")
	fmt.Println("  analytics <short-url>  Get click analytics")
	fmt.Println("  qr <short-url>         Generate QR code")
	fmt.Println("  tui                   Terminal UI dashboard")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  YOORL_BASE_URL    API base URL (default: http://localhost:8080)")
	fmt.Println("  YOORL_API_KEY     API key for authentication (key only)")
	fmt.Println("  PORT              Server port (default: 8080)")
}

func runServer() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().
		Timestamp().
		Logger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := store.InitializeStore(); err != nil {
		log.Warn().Err(err).Msg("Redis unavailable, using in-memory store")
		store.SetStore(store.NewMemoryStore())
	}

	tracker := analytics.NewTracker()
	tracker.Start(log)
	handler.SetTracker(tracker)

	limiter := middleware.NewTokenBucket(100, 50)
	limiter.Start()

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
	v1.Use(middleware.APIKeyAuth(), middleware.RateLimit(limiter))
	{
		v1.GET("/urls", handler.HandleListUrls)
		v1.POST("/urls", handler.CreateShortUrl)
		v1.DELETE("/urls/:shortUrl", handler.DeleteShortUrl)
		v1.PUT("/urls/:shortUrl", handler.UpdateShortUrl)
		v1.GET("/urls/:shortUrl/analytics", handler.HandleGetAnalytics)
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

	limiter.Stop()
	tracker.Stop()
	log.Info().Msg("server exited")
}

func cmdCreate(args []string) {
	var aliasVal, target string

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--alias" || args[i] == "-alias":
			if i+1 < len(args) {
				i++
				aliasVal = args[i]
			}
		case args[i] == "-a":
			if i+1 < len(args) {
				i++
				aliasVal = args[i]
			}
		case args[i] == "--user" || args[i] == "-user", args[i] == "-u":
			i++
		default:
			if target == "" {
				target = args[i]
			}
		}
	}

	if target == "" {
		fmt.Fprintln(os.Stderr, "error: url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl create [--alias <alias>] [--user <id>] <url>")
		os.Exit(1)
	}

	created, err := newClient().CreateURL(target, aliasVal, "cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created: %s -> %s\n", created.ShortURL, created.LongURL)
}

func cmdDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: short-url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl delete <short-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	if err := newClient().DeleteURL(shortUrl); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted: %s\n", shortUrl)
}

func cmdUpdate(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "error: short-url and url are required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl update <short-url> <new-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	newUrl := args[1]

	if _, err := url.ParseRequestURI(newUrl); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid URL: %s\n", newUrl)
		os.Exit(1)
	}

	updated, err := newClient().UpdateURL(shortUrl, newUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated: %s -> %s\n", updated.ShortURL, updated.LongURL)
}

func cmdAnalytics(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: short-url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl analytics <short-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	stats, err := newClient().GetAnalytics(shortUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analytics for %s:\n", stats.ShortUrl)
	fmt.Printf("  Total clicks: %d\n", stats.TotalClicks)
	fmt.Println("  Recent visits:")
	for _, c := range stats.RecentClicks {
		fmt.Printf("    %s | %s | %s | %s\n",
			c.Timestamp.Format(time.RFC3339),
			c.IP,
			truncate(c.UserAgent, 50),
			c.Referer,
		)
	}
	if len(stats.RecentClicks) == 0 {
		fmt.Println("    (none)")
	}
}

func cmdQr(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: short-url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl qr <short-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	fullURL := baseURL() + "/r/" + shortUrl

	qr, err := qrcode.New(fullURL, qrcode.Medium)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating QR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(qr.ToSmallString(false))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
