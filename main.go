package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"ncore_automation/pkg/ncore"
	"ncore_automation/pkg/web"
)

func main() {
	debug := flag.Bool("d", false, "Enable debug logging to log.txt")
	user := flag.String("u", "", "nCore username")
	pass := flag.String("p", "", "nCore password")
	outDir := flag.String("o", "", "Directory to store downloaded torrents")
	port := flag.String("port", "8080", "Web interface port")
	flag.Parse()

	if *user == "" || *pass == "" || *outDir == "" {
		fmt.Println("username, password and output directory are required")
		flag.Usage()
		return
	}

	logBuf := web.NewLogBuffer(1000)
	logger := log.New(io.MultiWriter(os.Stdout, logBuf), "", log.LstdFlags)
	if *debug {
		logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Println("Error opening log file:", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logger.SetOutput(io.MultiWriter(os.Stdout, logFile, logBuf))
	}

	statusChan := make(chan string, 100)
	statusStore := web.NewStatusStore()
	go func() {
		for s := range statusChan {
			statusStore.Add(s)
		}
	}()

	ctx := context.Background()
	go func() {
		if err := ncore.Run(ctx, *user, *pass, *outDir, logger, statusChan); err != nil {
			logger.Println("ncore run error:", err)
		}
	}()

	portChan := make(chan string)
	go web.Run(*port, logger, logBuf, statusStore, portChan)

	select {}
}
