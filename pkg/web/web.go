package web

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LogBuffer stores recent log lines.
type LogBuffer struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func NewLogBuffer(max int) *LogBuffer {
	return &LogBuffer{max: max}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := string(p)
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		b.lines = append(b.lines, line)
		if len(b.lines) > b.max {
			b.lines = b.lines[len(b.lines)-b.max:]
		}
	}
	return len(p), nil
}

func (b *LogBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.lines...)
}

// StatusStore keeps track of status messages.
type StatusStore struct {
	mu      sync.Mutex
	entries []string
}

func NewStatusStore() *StatusStore { return &StatusStore{} }

func (s *StatusStore) Add(entry string) {
	s.mu.Lock()
	s.entries = append(s.entries, entry)
	s.mu.Unlock()
}

func (s *StatusStore) Entries() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.entries...)
}

func localIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"127.0.0.1"}
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	if len(ips) == 0 {
		ips = append(ips, "127.0.0.1")
	}
	return ips
}

// Run starts the HTTP server and listens for port change requests via portChan.
func Run(port string, logger *log.Logger, logBuf *LogBuffer, status *StatusStore, portChan chan string) {
	currentPort := port
	for {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `<html><body><h1>nCore Auto Seed Checker</h1>
<p>Running on port %s</p>
<form action="/set-port" method="POST">
<input name="port" value="%s" />
<input type="submit" value="Change Port" />
</form>
<p><a href="/logs">Logs</a></p>
<p><a href="/status">Upload Status</a></p>
</body></html>`, currentPort, currentPort)
		})
		mux.HandleFunc("/set-port", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			newPort := r.FormValue("port")
			if newPort == "" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			fmt.Fprintf(w, "Port change to %s initiated, reloading...", newPort)
			go func() { portChan <- newPort }()
		})
		mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintln(w, "<pre>")
			for _, line := range logBuf.Lines() {
				fmt.Fprintln(w, line)
			}
			fmt.Fprintln(w, "</pre>")
		})
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintln(w, "<ul>")
			for _, e := range status.Entries() {
				fmt.Fprintf(w, "<li>%s</li>", e)
			}
			fmt.Fprintln(w, "</ul>")
		})

		srv := &http.Server{Addr: ":" + currentPort, Handler: mux}
		go func() {
			logger.Println("Starting web server on port", currentPort)
			for _, ip := range localIPs() {
				logger.Printf("Web UI available at http://%s:%s\n", ip, currentPort)
			}
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Println("HTTP server error:", err)
			}
		}()
		newPort := <-portChan
		logger.Println("Port change requested to", newPort)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		srv.Shutdown(ctx)
		cancel()
		currentPort = newPort
	}
}
