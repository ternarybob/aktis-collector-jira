package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ternarybob/arbor"
)

// WebSocketHub manages active WebSocket connections and log streaming
type WebSocketHub struct {
	clients     map[*websocket.Conn]bool
	broadcast   chan []byte
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
	mutex       sync.RWMutex
	logger      arbor.ILogger
	lastLogTime time.Time
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger arbor.ILogger) *WebSocketHub {
	hub := &WebSocketHub{
		clients:     make(map[*websocket.Conn]bool),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *websocket.Conn),
		unregister:  make(chan *websocket.Conn),
		logger:      logger,
		lastLogTime: time.Now(),
	}
	go hub.run()
	return hub
}

// run manages client connections and broadcasts
func (h *WebSocketHub) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			h.logger.Debug().Msg("WebSocket client connected")

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()
			h.logger.Debug().Msg("WebSocket client disconnected")

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					h.logger.Warn().Err(err).Msg("Failed to send WebSocket message")
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()

		case <-ticker.C:
			// Send heartbeat to all clients
			h.SendStatus("online")

			// Stream new logs to clients
			h.streamLogs()
		}
	}
}

// SendStatus broadcasts server status to all clients
func (h *WebSocketHub) SendStatus(status string) {
	msg := map[string]interface{}{
		"type":      "status",
		"status":    status,
		"timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// SendCollectionUpdate broadcasts collection updates to all clients
func (h *WebSocketHub) SendCollectionUpdate(eventType string, data interface{}) {
	msg := map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}
	jsonData, _ := json.Marshal(msg)
	h.broadcast <- jsonData
}

// streamLogs sends new logs to all connected clients
// TODO: Implement server log streaming when Arbor v1.4.45 WebSocket API is clarified
// The memory writer and log store interfaces need to be verified with source code access
func (h *WebSocketHub) streamLogs() {
	// Placeholder for future implementation
	// Arbor v1.4.45 has memory writer and WebSocket capabilities,
	// but the exact API for retrieving logs from the store needs verification
}

// Upgrader for WebSocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for Chrome extension
	},
}

// WebSocketHandler handles WebSocket connection requests
func (h *WebSocketHub) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	h.register <- conn

	// Keep connection alive and handle messages
	go func() {
		defer func() {
			h.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
