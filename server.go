package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	// WebSocket upgrader
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity in this local tool
		},
	}

	// Connected clients
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

// BroadcastUpdate sends the latest history to all connected clients
func BroadcastUpdate() {
	log.Println("Broadcasting update to clients...")
	hm, err := NewHistoryManager()
	if err != nil {
		log.Println("Failed to load history for broadcast:", err)
		return
	}

	records := hm.GetRecords()
	data, err := json.Marshal(records)
	if err != nil {
		log.Println("Failed to marshal history:", err)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func startWebServer(port int) {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on %s...", addr)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/history", handleHistory)
	http.HandleFunc("/ws", handleConnections)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	clientsMu.Lock()
	clients[ws] = true
	clientsMu.Unlock()

	log.Println("New client connected via WebSocket")

	// Keep connection alive or listen for close
	for {
		// Read message to detect close, but we don't expect client messages
		_, _, err := ws.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, ws)
			clientsMu.Unlock()
			break
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	hm, err := NewHistoryManager()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hm.GetRecords())
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>nCore Seed Checker</title>
    <style>
        :root {
            --bg-color: #0f172a;
            --card-bg: #1e293b;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent: #38bdf8;
            --success: #22c55e;
            --error: #ef4444;
            --border: #334155;
        }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            margin: 0;
            padding: 2rem;
            line-height: 1.5;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
        }
        header {
            margin-bottom: 2rem;
            border-bottom: 1px solid var(--border);
            padding-bottom: 1rem;
        }
        h1 { margin: 0; font-size: 1.8rem; font-weight: 700; color: var(--accent); }
        
        .card {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            margin-bottom: 2rem;
            border: 1px solid var(--border);
        }
        
        table {
            width: 1000px;
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }
        th {
            background: rgba(255,255,255,0.05);
            padding: 12px;
            color: var(--text-secondary);
            font-weight: 600;
            font-size: 0.9rem;
            border-bottom: 1px solid var(--border);
        }
        td {
            padding: 12px;
            border-bottom: 1px solid var(--border);
            font-size: 0.95rem;
        }
        tr:last-child td { border-bottom: none; }
        
        .status-badge {
            display: inline-flex;
            align-items: center;
            padding: 4px 8px;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
        }
        .status-success { background: rgba(34, 197, 94, 0.1); color: var(--success); }
        .status-error { background: rgba(239, 68, 68, 0.1); color: var(--error); }
        
        .downloads-list {
            list-style: none;
            padding: 0;
            margin: 0;
            font-size: 0.85rem;
            color: var(--text-secondary);
        }
        .downloads-list li { margin-bottom: 2px; }

        .time-cell { color: var(--text-secondary); }
		
		#connection-status {
			font-size: 0.8rem;
			margin-left: 1rem;
			color: var(--text-secondary);
		}
		.connected { color: var(--success) !important; }
		.disconnected { color: var(--error) !important; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>nCore Seed Checker History <span id="connection-status">Connecting...</span></h1>
        </header>

        <div class="card">
            <h2 style="margin-top:0">Run History</h2>
            <div id="loading">Loading...</div>
            <table id="historyTable" style="display:none">
                <thead>
                    <tr>
                        <th width="20%">Timestamp</th>
                        <th width="10%">Duration</th>
                        <th width="10%">Found</th>
                        <th width="10%">Downloaded</th>
                        <th width="15%">Status</th>
                        <th>Downloads</th>
                    </tr>
                </thead>
                <tbody id="historyBody"></tbody>
            </table>
        </div>
    </div>

    <script>
        function renderTable(data) {
             const tbody = document.getElementById('historyBody');
             tbody.innerHTML = '';
             
             if (!data || data.length === 0) {
                 document.getElementById('loading').textContent = 'No history found.';
                 document.getElementById('loading').style.display = 'block';
                 document.getElementById('historyTable').style.display = 'none';
                 return;
             }

             data.forEach(run => {
                 const row = document.createElement('tr');
                 const date = new Date(run.timestamp).toLocaleString();
                 const statusClass = run.log_summary === 'Success' ? 'status-success' : 'status-error';
                 const downloads = run.downloads ? run.downloads.map(d => '<li>' + d + '</li>').join('') : '-';

                 row.innerHTML = '<td>' + date + '</td>' +
                     '<td>' + run.duration + '</td>' +
                     '<td>' + run.items_found + '</td>' +
                     '<td>' + run.items_downloaded + '</td>' +
                     '<td><span class="status-badge ' + statusClass + '">' + run.log_summary + '</span></td>' +
                     '<td><ul class="downloads-list">' + downloads + '</ul></td>';
                 tbody.appendChild(row);
             });
             
             document.getElementById('loading').style.display = 'none';
             document.getElementById('historyTable').style.display = 'table';
        }

        async function loadHistory() {
            try {
                const response = await fetch('/api/history');
                const data = await response.json();
                renderTable(data);
            } catch (err) {
                console.error(err);
                document.getElementById('loading').textContent = 'Error loading history.';
            }
        }
        
        function connectWebSocket() {
            const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(proto + '//' + window.location.host + '/ws');
			const statusEl = document.getElementById('connection-status');

            ws.onopen = function() {
                console.log('WebSocket connected');
				statusEl.textContent = '● Live';
				statusEl.classList.add('connected');
				statusEl.classList.remove('disconnected');
            };

            ws.onmessage = function(event) {
                console.log('Update received via WebSocket');
                try {
                    const data = JSON.parse(event.data);
                    renderTable(data);
                } catch (e) {
                    console.error('Failed to parse WebSocket message', e);
                }
            };

            ws.onclose = function() {
                console.log('WebSocket disconnected. Reconnecting in 5s...');
				statusEl.textContent = '○ Disconnected';
				statusEl.classList.remove('connected');
				statusEl.classList.add('disconnected');
                setTimeout(connectWebSocket, 5000);
            };
        }
        
        // Initial load
        loadHistory();
        // Connect WS
        connectWebSocket();
    </script>
</body>
</html>
`
