package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeSystemStatus   MessageType = "system_status"
	MessageTypeProjectUpdate  MessageType = "project_update"
	MessageTypeTaskUpdate     MessageType = "task_update"
	MessageTypeAgentUpdate    MessageType = "agent_update"
	MessageTypeUserUpdate     MessageType = "user_update"
	MessageTypeMetricsUpdate  MessageType = "metrics_update"
	MessageTypeActivityUpdate MessageType = "activity_update"
	MessageTypeNotification   MessageType = "notification"
	MessageTypeError          MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	logger     *logrus.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Start starts the hub and handles client registration/unregistration and message broadcasting
func (h *Hub) Start() {
	h.logger.Info("Starting WebSocket hub")

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Stop stops the hub and closes all connections
func (h *Hub) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.logger.Info("Stopping WebSocket hub")

	// Close all client connections
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}

	// Close channels
	close(h.broadcast)
	close(h.register)
	close(h.unregister)
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	select {
	case h.register <- client:
	default:
		h.logger.Warn("Failed to register client: register channel full")
	}
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	select {
	case h.unregister <- client:
	default:
		h.logger.Warn("Failed to unregister client: unregister channel full")
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message Message) {
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("Failed to broadcast message: broadcast channel full")
	}
}

// BroadcastToSubscribed sends a message to clients subscribed to a specific topic
func (h *Hub) BroadcastToSubscribed(message Message, topic string) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.IsSubscribed(topic) {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, close it
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetClients returns a slice of all connected clients
func (h *Hub) GetClients() []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// registerClient handles client registration
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	h.clients[client] = true
	h.mutex.Unlock()

	h.logger.WithFields(logrus.Fields{
		"client_id":     client.id,
		"remote_addr":   client.conn.RemoteAddr().String(),
		"total_clients": len(h.clients),
	}).Info("Client connected")

	// Send welcome message
	welcomeMsg := Message{
		Type: MessageTypeNotification,
		Data: map[string]interface{}{
			"message":   "Connected to KaskManager R&D Platform",
			"client_id": client.id,
		},
		Timestamp: client.getTimestamp(),
	}

	select {
	case client.send <- welcomeMsg:
	default:
		close(client.send)
		delete(h.clients, client)
	}
}

// unregisterClient handles client unregistration
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mutex.Unlock()

	h.logger.WithFields(logrus.Fields{
		"client_id":     client.id,
		"remote_addr":   client.conn.RemoteAddr().String(),
		"total_clients": len(h.clients),
	}).Info("Client disconnected")
}

// broadcastMessage sends a message to all connected clients
func (h *Hub) broadcastMessage(message Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Serialize message once for all clients
	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	// Track failed sends
	var failedClients []*Client

	for client := range h.clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// Client's send channel is full, mark for removal
			failedClients = append(failedClients, client)
		}
	}

	// Remove failed clients
	for _, client := range failedClients {
		close(client.send)
		delete(h.clients, client)
		h.logger.WithField("client_id", client.id).Warn("Removed unresponsive client")
	}

	h.logger.WithFields(logrus.Fields{
		"message_type":   message.Type,
		"clients_sent":   len(h.clients),
		"clients_failed": len(failedClients),
		"message_size":   len(messageBytes),
	}).Debug("Broadcast message sent")
}

// BroadcastSystemStatus broadcasts system status update
func (h *Hub) BroadcastSystemStatus(status interface{}) {
	message := Message{
		Type:      MessageTypeSystemStatus,
		Data:      status,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastProjectUpdate broadcasts project update
func (h *Hub) BroadcastProjectUpdate(project interface{}) {
	message := Message{
		Type:      MessageTypeProjectUpdate,
		Data:      project,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastTaskUpdate broadcasts task update
func (h *Hub) BroadcastTaskUpdate(task interface{}) {
	message := Message{
		Type:      MessageTypeTaskUpdate,
		Data:      task,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastAgentUpdate broadcasts agent update
func (h *Hub) BroadcastAgentUpdate(agent interface{}) {
	message := Message{
		Type:      MessageTypeAgentUpdate,
		Data:      agent,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastUserUpdate broadcasts user update
func (h *Hub) BroadcastUserUpdate(user interface{}) {
	message := Message{
		Type:      MessageTypeUserUpdate,
		Data:      user,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastMetricsUpdate broadcasts metrics update
func (h *Hub) BroadcastMetricsUpdate(metrics interface{}) {
	message := Message{
		Type:      MessageTypeMetricsUpdate,
		Data:      metrics,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastActivityUpdate broadcasts activity update
func (h *Hub) BroadcastActivityUpdate(activity interface{}) {
	message := Message{
		Type:      MessageTypeActivityUpdate,
		Data:      activity,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastNotification broadcasts a notification
func (h *Hub) BroadcastNotification(notification interface{}) {
	message := Message{
		Type:      MessageTypeNotification,
		Data:      notification,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// BroadcastError broadcasts an error message
func (h *Hub) BroadcastError(error interface{}) {
	message := Message{
		Type:      MessageTypeError,
		Data:      error,
		Timestamp: getTimestamp(),
	}
	h.Broadcast(message)
}

// getTimestamp returns current Unix timestamp in milliseconds
func getTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
