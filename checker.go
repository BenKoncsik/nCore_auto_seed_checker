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
	return "cookies.txt"
}

func (c *Checker) runLynx(args ...string) (string, error) {
	// Add default args for non-interactive mode
	// -accept_all_cookies: automatically accept cookies
	// -cookie_save_file: where to save cookies
	// -cookie_file: where to read cookies from
	cookieFile := c.getCookieFile()
	baseArgs := []string{
		"-accept_all_cookies",
		"-cookie_save_file=" + cookieFile,
		"-cookie_file=" + cookieFile,
		"-width=200", // Ensure wide output to avoid wrapping issues in dumps if we used -dump, less relevant for -source
	}

	finalArgs := append(baseArgs, args...)

	if c.Debug {
		c.Logger.Printf("Running lynx with args: %v", finalArgs)
	}

	cmd := exec.Command("lynx", finalArgs...)

	// If we are posting data, we need to handle it separately, but here we assume args capture most needs.
	// For raw POST data, we might need to pipe to stdin. See login method.

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.Logger.Printf("Lynx error info: %s", stderr.String())
		return "", fmt.Errorf("lynx execution failed: %w", err)
	}

	return out.String(), nil
}

func (c *Checker) login() error {
	c.Logger.Println("Attempting to log in...")

	// Construct the post data
	// nCore login usually expects 'nev' and 'pass'
	postData := fmt.Sprintf("nev=%s&pass=%s&ne_leptessen_ki=1", c.User, c.Pass)

	cookieFile := c.getCookieFile()

	// We use -post_data. Lynx expects the data on stdin.
	args := []string{
		"-accept_all_cookies",
		"-cookie_save_file=" + cookieFile,
		"-cookie_file=" + cookieFile,
		"-post_data",
		"-source", // We want the source HTML to check if login succeeded
		loginUrl,
	}

	cmd := exec.Command("lynx", args...)
	cmd.Stdin = strings.NewReader(postData)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		c.Logger.Printf("Lynx login error info: %s", stderr.String())
		return fmt.Errorf("login command failed: %w", err)
	}

	body := out.String()

	if !strings.Contains(body, c.User) {
		// Just debug check, sometimes raw source might be different, but usually nCore shows username on top right
		// c.Logger.Println("Login response body preview:", body[:500])
		// Actually, let's look for "kilépés" or something that indicates we are logged in.
		// Or just trust the cookies?
		// Let's stick to the previous check:
		// if !strings.Contains(body, c.User) { ... }
		// But note that lynx -source returns the raw HTML.

		// If login redirects, lynx might follow it?
		// Standard lynx behavior with post_data:
		// "lynx -post_data ... URL"
		// If it's a 302, Lynx might follow or show "Data transfer complete".

		// Let's relax the check slightly or look for the redirect/success indicator.
		// If we are on hitnrun.php immediately (because of honnan param), we might see "Aktivitás" or similar.
		// Or just check that we DIDNT get the login page again.
		if strings.Contains(body, "name=\"pass\"") { // Login form still present
			c.Logger.Println("Login failed, form still present.")
			return fmt.Errorf("login failed, found login form")
		}
	}

	c.Logger.Println("Login successfully executed (cookies saved).")
	return nil
}

func (c *Checker) checkActivity() ([]string, error) {
	c.Logger.Println("Opening activity page...")

	body, err := c.runLynx("-source", activityUrl)
	if err != nil {
		return nil, err
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

	// Log matches
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
	body, err := c.runLynx("-source", torrentUrl)
	if err != nil {
		c.Logger.Println("Error opening the page:", err)
		return "", err
	}

	// Search for filename in the match first (from activity page), or in the details page
	fileNameRegex := regexp.MustCompile(`<a[^>]*title="([^"]+)"`)
	fileNameMatch := fileNameRegex.FindStringSubmatch(match)

	rawFileName := "unknown_torrent"
	if len(fileNameMatch) >= 2 {
		rawFileName = fileNameMatch[1]
	} else {
		// Try to find it in the details page body as a backup
		// Looking for <div class="torrent_reszletek_cim">Title</div> or similar
		// But let's trust the regex for now or fallback
		c.Logger.Println("Could not find filename in match snippet, using generic name")
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

	c.Logger.Println("Downloading file via Lynx to stdout...", downloadUrl)

	// Use lynx -source to get the binary content
	content, err := c.runLynx("-source", downloadUrl)
	if err != nil {
		return err
	}

	// Create output directory
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
