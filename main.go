package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

var (
	loginUrl    = "https://ncore.pro/login.php?honnan=/hitnrun.php"
	activityUrl = "https://ncore.pro/hitnrun.php"
	loginData   = struct {
		Nev  string
		Pass string
	}{}
	outputDir string
)

func main() {
	debug := flag.Bool("d", false, "Enable debug logging to log.txt")
	user := flag.String("u", "", "nCore username")
	pass := flag.String("p", "", "nCore password")
	outDir := flag.String("o", "", "Directory to store downloaded torrents")
	flag.Parse()

	if *user == "" || *pass == "" || *outDir == "" {
		fmt.Println("username, password and output directory are required")
		flag.Usage()
		return
	}

	loginData.Nev = *user
	loginData.Pass = *pass
	outputDir = *outDir

	var err error
	logger := log.New(os.Stdout, "", log.LstdFlags)
	if *debug {
		logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Println("Error opening log file:", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	// Start Chrome
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(logger.Printf))
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 240*time.Second)
	defer cancel()

	// Log in to nCore
	var body string
	err = chromedp.Run(ctx,
		chromedp.Navigate(loginUrl),
		chromedp.WaitReady(`#nev`, chromedp.ByID),
		chromedp.SendKeys(`#nev`, loginData.Nev, chromedp.ByID),
		chromedp.SendKeys(`[name="pass"]`, loginData.Pass, chromedp.ByQuery),
		chromedp.Click(`[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitReady(`a[href*="hitnrun"]`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	// Verify if login was successful
	if !strings.Contains(body, loginData.Nev) {
		logger.Println("Login failed, username not found on the page.")
		os.Exit(1)
	}
	logger.Println("Login successful.")

	// Open activity page
	logger.Println("Opening activity page...")
	err = chromedp.Run(ctx,
		chromedp.Navigate(activityUrl),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	if *debug {
		logger.Println("HTML content:")
		logger.Println(body)
	}

	// Step 3: Find torrents with "Stopped" status and click on them
	logger.Println("Analyzing HTML to find torrents with 'Stopped' status...")

	var rows []*cdp.Node
	//var urlsToDownload []string
	err = chromedp.Run(ctx,
		chromedp.Nodes(`div[class^="hnr_all"]`, &rows, chromedp.ByQueryAll),
	)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	logger.Printf("Found %d rows in total.", len(rows))
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		panic(err)
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
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	for i, div := range matches {
		fmt.Printf("Found div #%d:\n%s\n\n", i+1, div)
	}

	logger.Printf("Found %d rows with 'Stopped' status.", len(matches))

	for i, match := range matches {
		logger.Printf("Row %d: %s", i+1, match)

		linkRegex := regexp.MustCompile(`<a href="(torrents\.php\?action=details[^"]*)"`)
		linkMatch := linkRegex.FindStringSubmatch(match)

		if len(linkMatch) > 1 {
			logger.Println("Opening torrent page:", linkMatch[1], "Match:", match)
			torrentLink := linkMatch[1]
			torrentUrl := "https://ncore.pro/" + strings.ReplaceAll(torrentLink, "&amp;", "&")

			downloadTorrent(ctx, torrentUrl, match, logger)
		}
	}
}

func downloadTorrent(ctx context.Context, torrentUrl string, match string, logger *log.Logger) {
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(torrentUrl),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		logger.Println("Error opening the page:", err)
		return
	}

	// Search and download torrent link
	fileNameRegex := regexp.MustCompile(`<a[^>]*title="([^"]+)"`)
	fileName := fileNameRegex.FindStringSubmatch(match)

	linkRegex := regexp.MustCompile(`<div class="download">.*?<a [^>]*href="(torrents\.php\?action=download[^"]*)"`)
	linkMatch := linkRegex.FindStringSubmatch(body)
	logger.Println("Link match:", len(linkMatch))
	if len(linkMatch) > 1 {
		downloadLink := linkMatch[1]
		downloadUrl := "https://ncore.pro/" + strings.ReplaceAll(downloadLink, "&amp;", "&")
		logger.Println("Opening torrent page:", torrentUrl, "Torrent download link:", downloadUrl, "file name:", fileName)
		downloadFile(downloadUrl, logger, fileName[len(fileName)-1]+".torrent")
	}
}

func downloadFile(downloadUrl string, logger *log.Logger, fileName string) {
	sanitizeFileName := func(name string) string {
		name = strings.ReplaceAll(name, "?", "_")
		name = strings.ReplaceAll(name, "&", "_")
		name = strings.ReplaceAll(name, "=", "_")
		return name
	}
	logger.Println("Downloading file:", downloadUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Println("Error:", resp.StatusCode)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	// Save downloaded file to the specified directory
	outputPath := filepath.Join(outputDir, sanitizeFileName(fileName))

	// Create output directory if it does not exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	// Write file
	if err := os.WriteFile(outputPath, body, 0644); err != nil {
		logger.Println("Error:", err)
		os.Exit(1)
	}

	logger.Println("File successfully downloaded and saved:", outputPath)
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
	// Rekurzívan bejárjuk a gyermek node-okat
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if containsStopped(c) {
			return true
		}
	}
	return false
}
