package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

type apiResponse struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func baseURL() string {
	if v := os.Getenv("YOORL_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBaseURL
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "create":
		cmdCreate(args)
	case "delete":
		cmdDelete(args)
	case "update":
		cmdUpdate(args)
	case "analytics":
		cmdAnalytics(args)
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
	fmt.Println("  create [--alias,-a <alias>] [--user,-u <id>] <url>")
	fmt.Println("                           Create a short URL")
	fmt.Println("  delete <short-url>     Delete a short URL")
	fmt.Println("  update <short-url> <new-url>")
	fmt.Println("                           Update a short URL's destination")
	fmt.Println("  analytics <short-url>  Get click analytics")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  YOORL_BASE_URL    API base URL (default: http://localhost:8080)")
}

func cmdCreate(args []string) {
	var aliasVal, userVal, target string
	userVal = "cli"

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
		case args[i] == "--user" || args[i] == "-user":
			if i+1 < len(args) {
				i++
				userVal = args[i]
			}
		case args[i] == "-u":
			if i+1 < len(args) {
				i++
				userVal = args[i]
			}
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

	body := map[string]string{
		"long_url": target,
		"user_id":  userVal,
	}
	if aliasVal != "" {
		body["custom_alias"] = aliasVal
	}

	resp := doRequest("POST", "/v1/urls", body)
	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Error)
		os.Exit(1)
	}

	var data struct {
		ShortUrl string `json:"short_url"`
		LongUrl  string `json:"long_url"`
	}
	json.Unmarshal(resp.Data, &data)

	fmt.Printf("Created: %s -> %s\n", data.ShortUrl, data.LongUrl)
}

func cmdDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: short-url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl delete <short-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	resp := doRequest("DELETE", "/v1/urls/"+shortUrl, nil)
	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Error)
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

	body := map[string]string{"long_url": newUrl}
	resp := doRequest("PUT", "/v1/urls/"+shortUrl, body)
	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Error)
		os.Exit(1)
	}

	var data struct {
		ShortUrl string `json:"short_url"`
		LongUrl  string `json:"long_url"`
	}
	json.Unmarshal(resp.Data, &data)

	fmt.Printf("Updated: %s -> %s\n", data.ShortUrl, data.LongUrl)
}

func cmdAnalytics(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: short-url is required")
		fmt.Fprintln(os.Stderr, "Usage: yoorl analytics <short-url>")
		os.Exit(1)
	}

	shortUrl := strings.TrimLeft(args[0], "/")
	resp := doRequest("GET", "/v1/urls/"+shortUrl+"/analytics", nil)
	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "error: %s\n", resp.Error)
		os.Exit(1)
	}

	var data struct {
		ShortUrl     string `json:"short_url"`
		TotalClicks  int64  `json:"total_clicks"`
		RecentClicks []struct {
			Timestamp time.Time `json:"timestamp"`
			IP        string    `json:"ip"`
			UserAgent string    `json:"user_agent"`
			Referer   string    `json:"referer"`
		} `json:"recent_clicks"`
	}
	json.Unmarshal(resp.Data, &data)

	fmt.Printf("Analytics for %s:\n", data.ShortUrl)
	fmt.Printf("  Total clicks: %d\n", data.TotalClicks)
	fmt.Println("  Recent visits:")
	for _, c := range data.RecentClicks {
		fmt.Printf("    %s | %s | %s | %s\n",
			c.Timestamp.Format(time.RFC3339),
			c.IP,
			truncate(c.UserAgent, 50),
			c.Referer,
		)
	}
	if len(data.RecentClicks) == 0 {
		fmt.Println("    (none)")
	}
}

func doRequest(method, path string, body interface{}) apiResponse {
	url := baseURL() + path

	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, _ := http.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var apiResp apiResponse
	json.Unmarshal(respBody, &apiResp)

	if resp.StatusCode >= 400 && apiResp.Error == "" {
		apiResp.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return apiResp
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
