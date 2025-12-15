package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Checker struct {
	User      string
	Pass      string
	OutputDir string
	Debug     bool
	Logger    *log.Logger
}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func (c *Checker) Run() RunRecord {
	startTime := time.Now()
	record := RunRecord{
		Timestamp: startTime,
		Downloads: []string{},
	}

	// Login
	if err := c.login(); err != nil {
		c.Logger.Println("Login failed:", err)
		record.LogSummary = fmt.Sprintf("Login failed: %v", err)
		record.Duration = time.Since(startTime).String()
		return record
	}

	// Check activity
	matches, err := c.checkActivity()
	if err != nil {
		c.Logger.Println("Activity check failed:", err)
		record.LogSummary = fmt.Sprintf("Activity check failed: %v", err)
		record.Duration = time.Since(startTime).String()
		return record
	}
	record.ItemsFound = len(matches)

	// Download
	downloaded := c.processMatches(matches)
	record.Downloads = downloaded
	record.ItemsDownloaded = len(downloaded)

	record.LogSummary = "Success"
	record.Duration = time.Since(startTime).String()
	return record
}

func (c *Checker) getCookieFile() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "cookies.txt")
}

func (c *Checker) runCurl(args ...string) (string, error) {
	// Base args for curl
	// -s: silent (no progress bar)
	// -L: follow redirects
	// -A: user agent
	baseArgs := []string{
		"-s",
		"-L",
		"-A", userAgent,
	}

	finalArgs := append(baseArgs, args...)

	if c.Debug {
		c.Logger.Printf("Running curl with args: %v", finalArgs)
	}

	cmd := exec.Command("curl", finalArgs...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.Logger.Printf("Curl error info: %s", stderr.String())
		return "", fmt.Errorf("curl execution failed: %w", err)
	}

	return out.String(), nil
}

func (c *Checker) login() error {
	c.Logger.Println("Attempting to log in via cURL...")

	cookieFile := c.getCookieFile()
	postData := fmt.Sprintf("nev=%s&pass=%s&ne_leptessen_ki=1", c.User, c.Pass)

	// -c: save cookies (cookie jar)
	// -d: post data
	args := []string{
		"-c", cookieFile,
		"-d", postData,
		loginUrl,
	}

	// Login usually follows redirect to index.php or similar
	body, err := c.runCurl(args...)
	if err != nil {
		return err
	}

	// Basic check if login succeeded
	if strings.Contains(body, "name=\"pass\"") { // Login form still present
		c.Logger.Println("Login failed, form still present in response.")
		return fmt.Errorf("login failed (bad credentials?)")
	}

	c.Logger.Println("Login request completed.")
	return nil
}

func (c *Checker) checkActivity() ([]string, error) {
	c.Logger.Println("Opening activity page...")

	cookieFile := c.getCookieFile()
	// -b: read cookies
	// -c: write cookies (update session if needed)
	args := []string{
		"-b", cookieFile,
		"-c", cookieFile,
		activityUrl,
	}

	body, err := c.runCurl(args...)
	if err != nil {
		return nil, err
	}

	if strings.Contains(body, "name=\"pass\"") {
		c.Logger.Println("Activity page shows login form. Session lost.")
		if c.Debug {
			snippet := body
			if len(snippet) > 500 {
				snippet = snippet[:500]
			}
			c.Logger.Printf("DEBUG: HTML Snippet:\n%s", snippet)
		}
		return nil, fmt.Errorf("authentication failed")
	}

	c.Logger.Println("Analyzing HTML to find torrents with 'Stopped' status...")

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	var matches []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && (strings.HasPrefix(attr.Val, "hnr_all") || strings.HasPrefix(attr.Val, "hnr_all2")) {
					if containsStopped(n) {
						var buf strings.Builder
						html.Render(&buf, n)
						matches = append(matches, buf.String())
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			f(child)
		}
	}
	f(doc)

	c.Logger.Printf("Found %d rows with 'Stopped' status.", len(matches))
	for i, div := range matches {
		c.Logger.Printf("Found div #%d:\n%s\n\n", i+1, div)
	}

	return matches, nil
}

func (c *Checker) processMatches(matches []string) []string {
	var downloaded []string
	for i, match := range matches {
		c.Logger.Printf("Processing row %d...", i+1)

		linkRegex := regexp.MustCompile(`<a href="(torrents\.php\?action=details[^"]*)"`)
		linkMatch := linkRegex.FindStringSubmatch(match)

		if len(linkMatch) > 1 {
			c.Logger.Println("Opening torrent page:", linkMatch[1])
			torrentLink := linkMatch[1]
			torrentUrl := "https://ncore.pro/" + strings.ReplaceAll(torrentLink, "&amp;", "&")

			fileName, err := c.downloadTorrent(torrentUrl, match)
			if err == nil && fileName != "" {
				downloaded = append(downloaded, fileName)
			} else if err != nil {
				c.Logger.Println("Failed to download torrent:", err)
			}
		}
	}
	return downloaded
}

func (c *Checker) downloadTorrent(torrentUrl string, match string) (string, error) {
	cookieFile := c.getCookieFile()
	args := []string{
		"-b", cookieFile,
		"-c", cookieFile,
		torrentUrl,
	}

	body, err := c.runCurl(args...)
	if err != nil {
		c.Logger.Println("Error opening the page:", err)
		return "", err
	}

	fileNameRegex := regexp.MustCompile(`<a[^>]*title="([^"]+)"`)
	fileNameMatch := fileNameRegex.FindStringSubmatch(match)

	rawFileName := "unknown_torrent"
	if len(fileNameMatch) >= 2 {
		rawFileName = fileNameMatch[1]
	}

	linkRegex := regexp.MustCompile(`<div class="download">.*?<a [^>]*href="(torrents\.php\?action=download[^"]*)"`)
	linkMatch := linkRegex.FindStringSubmatch(body)

	if len(linkMatch) > 1 {
		downloadLink := linkMatch[1]
		downloadUrl := "https://ncore.pro/" + strings.ReplaceAll(downloadLink, "&amp;", "&")
		c.Logger.Println("Found download link:", downloadUrl)

		finalName := rawFileName + ".torrent"
		if err := c.downloadFile(downloadUrl, finalName); err != nil {
			return "", err
		}
		return finalName, nil
	}
	if strings.Contains(body, "name=\"pass\"") {
		return "", fmt.Errorf("authentication failed on details page")
	}
	return "", fmt.Errorf("download link not found")
}

func (c *Checker) downloadFile(downloadUrl string, fileName string) error {
	sanitizeFileName := func(name string) string {
		name = strings.ReplaceAll(name, "?", "_")
		name = strings.ReplaceAll(name, "&", "_")
		name = strings.ReplaceAll(name, "=", "_")
		name = strings.ReplaceAll(name, "/", "_")
		return name
	}

	c.Logger.Println("Downloading file via cURL...", downloadUrl)

	cookieFile := c.getCookieFile()
	// Download only, write to stdout for capture
	args := []string{
		"-b", cookieFile,
		"-c", cookieFile,
		downloadUrl,
	}

	content, err := c.runCurl(args...)
	if err != nil {
		return err
	}

	if strings.HasPrefix(strings.TrimSpace(content), "<!DOCTYPE") || strings.Contains(content, "<html") {
		return fmt.Errorf("downloaded content appears to be HTML, likely failed auth or invalid link")
	}

	if err := os.MkdirAll(c.OutputDir, os.ModePerm); err != nil {
		return err
	}

	outputPath := filepath.Join(c.OutputDir, sanitizeFileName(fileName))

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		c.Logger.Println("Error writing file:", err)
		return err
	}

	c.Logger.Println("File successfully downloaded and saved:", outputPath)
	return nil
}
