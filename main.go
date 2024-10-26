package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

var (
	loginUrl    = "https://ncore.pro/login.php?honnan=/hitnrun.php"
	activityUrl = "https://ncore.pro/hitnrun.php"
	loginData   = struct {
		Nev  string
		Pass string
	}{
		Nev:  "",
		Pass: "",
	}
	outputDir = ""
)

func main() {
	debug := flag.Bool("d", false, "Enable debug logging to log.txt")
	flag.Parse()

	var logFile *os.File
	var err error
	if *debug {
		logFile, err = os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Println("Error opening log file: ", err)
			os.Exit(1)
		}
		defer logFile.Close()
	}

	log := func(v ...interface{}) {
		fmt.Println(v...)
		if *debug {
			fmt.Fprintln(logFile, v...)
		}
	}

	// Start Chrome
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(func(s string, i ...interface{}) { log(fmt.Sprintf(s, i...)) }))
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
		log("Error: ", err)
		os.Exit(1)
	}

	// Verify if login was successful
	if !strings.Contains(body, loginData.Nev) {
		log("Login failed, username not found on the page.")
		os.Exit(1)
	}
	log("Login successful.")

	// Open activity page
	log("Opening activity page...")
	err = chromedp.Run(ctx,
		chromedp.Navigate(activityUrl),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	if *debug {
		log("HTML content: ")
		log(body)
	}

	// Step 3: Find torrents with "Stopped" status and click on them
	log("Analyzing HTML to find torrents with 'Stopped' status...")

	var rows []*cdp.Node
	//var urlsToDownload []string
	err = chromedp.Run(ctx,
		chromedp.Nodes(`div[class^="hnr_all"]`, &rows, chromedp.ByQueryAll),
	)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	log(fmt.Sprintf("Found %d rows in total.", len(rows)))

	stoppedRegex := regexp.MustCompile(`<div class="[^"]*hnr_all[^"]*"[^>]*>[\s\S]*?<div class="[^"]*hnr_tseed[^"]*"[^>]*>[\s\S]*?<span class="stopped">Stopped<\/span>[\s\S]*?<\/div>`)

	matches := stoppedRegex.FindAllString(body, -1)

	log(fmt.Sprintf("Found %d rows with 'Stopped' status.", len(matches)))

	for i, match := range matches {
		log(fmt.Sprintf("Row %d: %s", i+1, match))

		linkRegex := regexp.MustCompile(`<a href="(torrents\.php\?action=details[^"]*)"`)
		linkMatch := linkRegex.FindStringSubmatch(match)
		fileNameRegex := regexp.MustCompile(`<a[^>]*title="([^"]+)"`)
		fileName := fileNameRegex.FindStringSubmatch(match)
		if len(linkMatch) > 1 {
			torrentLink := linkMatch[1]
			torrentUrl := "https://ncore.pro/" + strings.ReplaceAll(torrentLink, "&amp;", "&")
			log("Opening torrent page: ", torrentUrl, "File name: ", fileName)
			downloadTorrent(ctx, torrentUrl, fileName[len(fileName)-1], log)
		}
	}
}

func downloadTorrent(ctx context.Context, torrentUrl string, fileName string, log func(v ...interface{})) {
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(torrentUrl),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery),
	)
	if err != nil {
		log("Error opening the page: ", err)
		return
	}

	// Search and download torrent link
	linkRegex := regexp.MustCompile(`<div class="download">.*?<a [^>]*href="(torrents\.php\?action=download[^"]*)"`)
	linkMatch := linkRegex.FindStringSubmatch(body)

	if len(linkMatch) > 1 {
		downloadLink := linkMatch[1]
		downloadUrl := "https://ncore.pro/" + strings.ReplaceAll(downloadLink, "&amp;", "&")
		log("Torrent download link: ", downloadUrl)
		downloadFile(downloadUrl, log, fileName+".torrent")
	}
}

func downloadFile(downloadUrl string, log func(v ...interface{}), fileName string) {
	sanitizeFileName := func(name string) string {
		name = strings.ReplaceAll(name, "?", "_")
		name = strings.ReplaceAll(name, "&", "_")
		name = strings.ReplaceAll(name, "=", "_")
		return name
	}
	log("Downloading file: ", downloadUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log("Error: ", resp.StatusCode)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	// Save downloaded file to the specified directory
	outputPath := filepath.Join(outputDir, sanitizeFileName(fileName))

	// Create output directory if it does not exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	// Write file
	if err := ioutil.WriteFile(outputPath, body, 0644); err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	log("File successfully downloaded and saved: ", outputPath)
}
