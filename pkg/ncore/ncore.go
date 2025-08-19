package ncore

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

var (
	loginURL    = "https://ncore.pro/login.php?honnan=/hitnrun.php"
	activityURL = "https://ncore.pro/hitnrun.php"
)

func chromeExecPath() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	paths := []string{
		`C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe`,
		`C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe`,
		`C:\\Program Files\\Chromium\\Application\\chrome.exe`,
		`C:\\Program Files (x86)\\Chromium\\Application\\chrome.exe`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("chrome.exe"); err == nil {
		return p
	}
	return ""
}

// Run logs in to nCore and downloads torrents with "Stopped" status.
// Status messages about downloaded files are sent to the provided channel.
func Run(ctx context.Context, user, pass, outDir string, logger *log.Logger, status chan<- string) error {
	loginData := struct {
		Nev  string
		Pass string
	}{user, pass}
	outputDir := outDir

	opts := []chromedp.ExecAllocatorOption{}
	if p := chromeExecPath(); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(logger.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 240*time.Second)
	defer cancel()

	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitReady(`#nev`, chromedp.ByID),
		chromedp.SendKeys(`#nev`, loginData.Nev, chromedp.ByID),
		chromedp.SendKeys(`[name="pass"]`, loginData.Pass, chromedp.ByQuery),
		chromedp.Click(`[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitReady(`a[href*="hitnrun"]`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		return err
	}

	if !strings.Contains(body, loginData.Nev) {
		return fmt.Errorf("login failed, username not found on the page")
	}
	logger.Println("Login successful.")

	logger.Println("Opening activity page...")
	err = chromedp.Run(ctx,
		chromedp.Navigate(activityURL),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		return err
	}

	logger.Println("Analyzing HTML to find torrents with 'Stopped' status...")

	var rows []*cdp.Node
	err = chromedp.Run(ctx,
		chromedp.Nodes(`div[class^="hnr_all"]`, &rows, chromedp.ByQueryAll),
	)
	if err != nil {
		return err
	}
	logger.Printf("Found %d rows in total.", len(rows))

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return err
	}
	var matches []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			var sb strings.Builder
			html.Render(&sb, n)
			content := sb.String()
			if containsStopped(n) {
				matches = append(matches, content)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	logger.Printf("Found %d rows with 'Stopped' status.", len(matches))

	for i, match := range matches {
		logger.Printf("Row %d: %s", i+1, match)
		linkRegex := regexp.MustCompile(`<a href="(torrents\.php\?action=details[^"]*)"`)
		linkMatch := linkRegex.FindStringSubmatch(match)
		if len(linkMatch) > 1 {
			torrentLink := linkMatch[1]
			torrentURL := "https://ncore.pro/" + strings.ReplaceAll(torrentLink, "&amp;", "&")
			downloadTorrent(ctx, torrentURL, match, logger, outputDir, status)
		}
	}
	return nil
}

func downloadTorrent(ctx context.Context, torrentURL, match string, logger *log.Logger, outputDir string, status chan<- string) {
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(torrentURL),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		logger.Println("Error opening the page:", err)
		return
	}

	fileNameRegex := regexp.MustCompile(`<a[^>]*title="([^"]+)"`)
	fileName := fileNameRegex.FindStringSubmatch(match)

	linkRegex := regexp.MustCompile(`<div class="download">.*?<a [^>]*href="(torrents\.php\?action=download[^"]*)"`)
	linkMatch := linkRegex.FindStringSubmatch(body)
	if len(linkMatch) > 1 {
		downloadLink := linkMatch[1]
		downloadURL := "https://ncore.pro/" + strings.ReplaceAll(downloadLink, "&amp;", "&")
		downloadFile(downloadURL, logger, fileName[len(fileName)-1]+".torrent", outputDir, status)
	}
}

func downloadFile(downloadURL string, logger *log.Logger, fileName string, outputDir string, status chan<- string) {
	sanitizeFileName := func(name string) string {
		name = strings.ReplaceAll(name, "?", "_")
		name = strings.ReplaceAll(name, "&", "_")
		name = strings.ReplaceAll(name, "=", "_")
		return name
	}
	logger.Println("Downloading file:", downloadURL)
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		logger.Println("Error:", err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Println("Error:", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Println("Error:", err)
		return
	}

	outputPath := filepath.Join(outputDir, sanitizeFileName(fileName))
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		logger.Println("Error:", err)
		return
	}
	if err := os.WriteFile(outputPath, body, 0644); err != nil {
		logger.Println("Error:", err)
		return
	}
	logger.Println("File successfully downloaded and saved:", outputPath)
	if status != nil {
		status <- fmt.Sprintf("Downloaded %s", fileName)
	}
}

func containsStopped(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "span" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == "stopped" {
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode && strings.TrimSpace(n.FirstChild.Data) == "Stopped" {
					return true
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if containsStopped(c) {
			return true
		}
	}
	return false
}
