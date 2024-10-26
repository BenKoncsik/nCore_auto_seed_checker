package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
	var urlsToDownload []string
	err = chromedp.Run(ctx,
		chromedp.Nodes(`div[class^="hnr_all"]`, &rows, chromedp.ByQueryAll),
	)
	if err != nil {
		log("Error: ", err)
		os.Exit(1)
	}

	log(fmt.Sprintf("Found %d rows in total.", len(rows)))

	for i := 0; i < len(rows); i++ {
		var rowHTML string
		err = chromedp.Run(ctx, chromedp.OuterHTML(`div[class^="hnr_all"]`, &rowHTML, chromedp.FromNode(rows[i])))

		if err != nil {
			log("Error: ", err)
			continue
		}

		if strings.Contains(rowHTML, "Stopped") {
			log(fmt.Sprintf("Extracting link from row %d...", i+1))
			var torrentLink string
			err = chromedp.Run(ctx, chromedp.AttributeValue(`a[href^="torrents.php?"]`, "href", &torrentLink, nil, chromedp.ByQuery, chromedp.FromNode(rows[i])))
			if err != nil {
				log("Error extracting torrent link: ", err)
				continue
			}
			if torrentLink != "" {
				torrentUrl := "https://ncore.pro/" + strings.ReplaceAll(torrentLink, "&amp;", "&")
				log("Adding torrent URL: ", torrentUrl)
				urlsToDownload = append(urlsToDownload, torrentUrl)
			}
		}
	}
	// Step 4. Process and download torrents from the URLs
	for _, torrentUrl := range urlsToDownload {
		log("Opening torrent page: ", torrentUrl)
		err = chromedp.Run(ctx, chromedp.Navigate(torrentUrl))
		if err != nil {
			log("Error while clicking: ", err)
			continue
		}

		chromedp.OuterHTML(`html`, &body, chromedp.ByQuery)
		if *debug {
			log("Loaded HTML page: ")
			log(body)
		}

		// Step 4: Find and download the torrent link
		var downloadLink string
		var torrentName string
		err = chromedp.Run(ctx,
			chromedp.WaitReady(`div.download a[href*="action=download"]`, chromedp.ByQuery),
			chromedp.AttributeValue(`div.download a[href*="action=download"]`, "href", &downloadLink, nil, chromedp.ByQuery),
			chromedp.Text(`div.torrent_reszletek_cim`, &torrentName, chromedp.ByQuery),
		)
		if err != nil {
			log("Error finding download link or torrent name: ", err)
			continue
		}

		if downloadLink != "" {
			downloadUrl := "https://ncore.pro/" + strings.ReplaceAll(downloadLink, "&amp;", "&")
			log("Torrent download link: ", downloadUrl)
			downloadFile(downloadUrl, log, torrentName+".torrent")
		}
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
