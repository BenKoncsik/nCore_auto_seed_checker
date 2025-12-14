package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	loginUrl    = "https://ncore.pro/login.php?honnan=/hitnrun.php"
	activityUrl = "https://ncore.pro/hitnrun.php"
)

func main() {
	debug := flag.Bool("d", false, "Enable debug logging to log.txt")
	user := flag.String("u", "", "nCore username")
	pass := flag.String("p", "", "nCore password")
	outDir := flag.String("o", "", "Directory to store downloaded torrents")
	webMode := flag.Bool("web", false, "Start web interface")
	interval := flag.Duration("interval", 0, "Check interval (e.g. 10m). If 0, runs once.")
	port := flag.Int("port", 8080, "Web server port")
	flag.Parse()

	if (*user == "" || *pass == "" || *outDir == "") && !*webMode {
		fmt.Println("username, password and output directory are required")
		flag.Usage()
		return
	}

	// If just web mode, we might need config passed in, or we just show history.
	// For now, let's assume we want to run the checker UNLESS web mode is on?
	// Or maybe web mode is a separate blocking call.
	if *webMode {
		go startWebServer(*port) // Run web server in a goroutine
	}

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

	checker := &Checker{
		User:      *user,
		Pass:      *pass,
		OutputDir: *outDir,
		Debug:     *debug,
		Logger:    logger,
	}

	logger.Println("Starting application...")

	// Helper to run one check
	runCheck := func() {
		record := checker.Run()
		hm, err := NewHistoryManager()
		if err != nil {
			logger.Println("Failed to load history manager:", err)
		} else {
			if err := hm.SaveRecord(record); err != nil {
				logger.Println("Failed to save run record:", err)
			} else {
				logger.Println("Run recorded to history.json")
			}
		}
	}

	// Always run once immediately
	runCheck()

	if *interval > 0 {
		logger.Printf("Running in loop mode. Checking every %v...", *interval)
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for range ticker.C {
			runCheck()
		}
	} else if *webMode {
		// If web mode is on but no interval, run once then block to keep server alive
		select {}
	}
}

// Helper needed for both
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
