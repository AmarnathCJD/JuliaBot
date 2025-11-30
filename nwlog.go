package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

type NetworkLogger struct {
	clients     map[chan string]bool
	mu          sync.RWMutex
	port        string
	server      *http.Server
	logFile     *os.File
	logFilePath string
	maxSize     int64
}

func NewNetworkLogger(port string) *NetworkLogger {
	logFilePath := "logs_stream.txt"
	maxSize := int64(5 * 1024 * 1024)

	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		logFile = nil
	}

	nl := &NetworkLogger{
		clients:     make(map[chan string]bool),
		port:        port,
		logFile:     logFile,
		logFilePath: logFilePath,
		maxSize:     maxSize,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/logs", nl.handleSSE)
	mux.HandleFunc("/logs/history", nl.handleHistory)
	mux.HandleFunc("/", nl.handleIndex)

	nl.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return nl
}

func (nl *NetworkLogger) Start() error {
	go func() {
		fmt.Printf("Network logger listening on http://localhost:%s\n", nl.port)
		if err := nl.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Network logger server error: %v\n", err)
		}
	}()
	return nil
}

func (nl *NetworkLogger) Stop() error {
	if nl.server != nil {
		return nl.server.Close()
	}
	return nil
}

func (nl *NetworkLogger) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))

	if nl.logFile != nil {
		nl.checkAndRotateLog()
		if _, err := nl.logFile.WriteString(msg + "\n"); err != nil {
			fmt.Printf("Failed to write to log file: %v\n", err)
		}
		nl.logFile.Sync()
	}

	nl.mu.Lock()
	for client := range nl.clients {
		select {
		case client <- msg:
		default:

			delete(nl.clients, client)
			close(client)
		}
	}
	nl.mu.Unlock()

	return len(p), nil
}

func (nl *NetworkLogger) checkAndRotateLog() {
	if nl.logFile == nil {
		return
	}

	stat, err := nl.logFile.Stat()
	if err != nil {
		return
	}

	if stat.Size() > nl.maxSize {

		nl.logFile.Close()

		backupPath := nl.logFilePath + ".backup"
		os.Rename(nl.logFilePath, backupPath)

		newFile, err := os.OpenFile(nl.logFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("Failed to create new log file: %v\n", err)
			nl.logFile = nil
			return
		}
		nl.logFile = newFile
	}
}

func (nl *NetworkLogger) addClient() chan string {
	client := make(chan string, 100)

	nl.mu.Lock()
	nl.clients[client] = true
	nl.mu.Unlock()

	return client
}

func (nl *NetworkLogger) removeClient(client chan string) {
	nl.mu.Lock()
	if _, exists := nl.clients[client]; exists {
		delete(nl.clients, client)
		close(client)
	}
	nl.mu.Unlock()
}

func (nl *NetworkLogger) handleSSE(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	client := nl.addClient()
	defer nl.removeClient(client)

	fmt.Fprintf(w, "data: Connected to JuliaBot log stream\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	for {
		select {
		case msg := <-client:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (nl *NetworkLogger) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JuliaBot Logs</title>
    <style>
        body {
            font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
            background: linear-gradient(135deg, #1a1a2e, #16213e);
            color: #e0e0e0;
            margin: 0;
            padding: 5px;
            font-size: 13px;
            line-height: 1.3;
        }
        .header {
            text-align: center;
            margin-bottom: 5px;
        }
        h1 {
            font-size: 16px;
            margin: 0;
            color: #00d4ff;
        }
        .status {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 3px 5px;
            background: rgba(0, 0, 0, 0.3);
            border-radius: 3px;
            margin-bottom: 5px;
            border: 1px solid #333;
        }
        .status-text {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        .status-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background: #00ff88;
        }
        .status.disconnected .status-dot { background: #ff4444; }
        .controls {
            display: flex;
            gap: 3px;
        }
        .control-btn {
            padding: 2px 6px;
            background: #2a2a2a;
            color: #e0e0e0;
            border: 1px solid #555;
            border-radius: 3px;
            cursor: pointer;
            font-size: 11px;
            transition: background 0.2s;
        }
        .control-btn:hover { background: #3a3a3a; }
        .control-btn.active { background: #4a4a4a; border-color: #777; }
        .logs-container {
            background: rgba(0, 0, 0, 0.5);
            border: 1px solid #444;
            height: calc(100vh - 60px);
            overflow: hidden;
            border-radius: 3px;
        }
        #logs {
            height: 100%;
            overflow-y: auto;
            padding: 3px;
            font-size: 12px;
            line-height: 1.4;
        }
        .log-entry {
            margin-bottom: 1px;
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .log-entry.error { color: #ff6b6b; }
        .log-entry.warn { color: #ffd93d; }
        .log-entry.info { color: #6bcf7f; }
        .log-entry.debug { color: #a8e6cf; }
        .log-entry.panic { color: #ff3838; font-weight: bold; background: rgba(255, 56, 56, 0.1); padding: 1px 2px; border-radius: 2px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>JuliaBot Logs</h1>
    </div>

    <div class="status connected" id="status">
        <div class="status-text">
            <div class="status-dot"></div>
            <span>Connected</span>
        </div>
        <div class="controls">
            <button class="control-btn active" id="autoScrollBtn" onclick="toggleAutoScroll()">Auto-scroll</button>
            <button class="control-btn" onclick="clearLogs()">Clear</button>
            <button class="control-btn" onclick="scrollToTop()">Top</button>
            <button class="control-btn" onclick="scrollToBottom()">Bottom</button>
        </div>
    </div>

    <div class="logs-container">
        <div id="logs"></div>
    </div>

    <script>
        const logsDiv = document.getElementById('logs');
        const statusDiv = document.getElementById('status');
        const autoScrollBtn = document.getElementById('autoScrollBtn');
        let autoScroll = true;
        let isAtBottom = true;

        async function loadExistingLogs() {
            try {
                const response = await fetch('/logs/history');
                if (response.ok) {
                    const history = await response.text();
                    const lines = history.split('\n').filter(line => line.trim());
                    lines.forEach(line => addLogEntry(line, false));
                }
            } catch (e) {
                console.log('No history available');
            }
        }

        function addLogEntry(logLine, scroll = true) {
            const logElement = document.createElement('div');
            logElement.className = 'log-entry';

            if (logLine.includes('PANIC')) {
                logElement.classList.add('panic');
            } else if (logLine.includes('ERROR')) {
                logElement.classList.add('error');
            } else if (logLine.includes('WARN')) {
                logElement.classList.add('warn');
            } else if (logLine.includes('INFO')) {
                logElement.classList.add('info');
            } else if (logLine.includes('DEBUG')) {
                logElement.classList.add('debug');
            }

            logElement.textContent = logLine;
            logsDiv.appendChild(logElement);

            if (scroll && autoScroll && isAtBottom) {
                setTimeout(() => logsDiv.scrollTop = logsDiv.scrollHeight, 10);
            }
        }

        function toggleAutoScroll() {
            autoScroll = !autoScroll;
            autoScrollBtn.classList.toggle('active', autoScroll);
            if (autoScroll && isAtBottom) {
                scrollToBottom();
            }
        }

        function clearLogs() {
            logsDiv.innerHTML = '';
        }

        function scrollToTop() {
            logsDiv.scrollTop = 0;
        }

        function scrollToBottom() {
            logsDiv.scrollTop = logsDiv.scrollHeight;
        }

        logsDiv.addEventListener('scroll', () => {
            const threshold = 50;
            isAtBottom = logsDiv.scrollTop + logsDiv.clientHeight >= logsDiv.scrollHeight - threshold;
        });

        const eventSource = new EventSource('/logs');

        eventSource.onmessage = function(event) {
            addLogEntry(event.data);
        };

        eventSource.onerror = function() {
            statusDiv.classList.remove('connected');
            statusDiv.classList.add('disconnected');
            statusDiv.querySelector('span').textContent = 'Disconnected';
        };

        eventSource.onopen = function() {
            statusDiv.classList.remove('disconnected');
            statusDiv.classList.add('connected');
            statusDiv.querySelector('span').textContent = 'Connected';
            loadExistingLogs();
            setTimeout(() => scrollToBottom(), 100);
        };

        loadExistingLogs();
        setTimeout(() => scrollToBottom(), 100);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func (nl *NetworkLogger) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if nl.logFile == nil {
		return
	}

	data, err := os.ReadFile(nl.logFilePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 1000 {
		lines = lines[len(lines)-1000:]
	}

	fmt.Fprint(w, strings.Join(lines, "\n"))
}
