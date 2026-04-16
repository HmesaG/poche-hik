package realtime

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

// Message represents a WebSocket message with type
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan interface{}),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Debug().Msg("WebSocket client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Debug().Msg("WebSocket client unregistered")

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Warn().Err(err).Msg("WebSocket write error")
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastAttendanceEvent broadcasts an attendance event to all connected clients
func (h *Hub) BroadcastAttendanceEvent(employeeNo, employeeName, deviceID string, timestamp time.Time) {
	msg := Message{
		Type: "attendance",
		Data: map[string]interface{}{
			"employeeNo":  employeeNo,
			"employeeName": employeeName,
			"deviceId":    deviceID,
			"timestamp":   timestamp,
			"dateTime":    timestamp.Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
	h.Broadcast(msg)
}

func (h *Hub) Broadcast(msg interface{}) {
	h.broadcast <- msg
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
