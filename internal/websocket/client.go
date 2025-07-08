package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// ClientMessage represents a message received from a client
type ClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	ID   string      `json:"id,omitempty"`
}

// SubscriptionMessage represents a subscription request
type SubscriptionMessage struct {
	Action string   `json:"action"` // subscribe, unsubscribe
	Topics []string `json:"topics"`
}

// Client represents a WebSocket client
type Client struct {
	id            string
	hub           *Hub
	conn          *websocket.Conn
	send          chan Message
	subscriptions map[string]bool
	mutex         sync.RWMutex
	logger        *logrus.Logger
	userID        *uuid.UUID
	authenticated bool
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, logger *logrus.Logger) *Client {
	return &Client{
		id:            uuid.New().String(),
		hub:           hub,
		conn:          conn,
		send:          make(chan Message, 256),
		subscriptions: make(map[string]bool),
		logger:        logger,
		authenticated: false,
	}
}

// GetID returns the client ID
func (c *Client) GetID() string {
	return c.id
}

// GetUserID returns the authenticated user ID
func (c *Client) GetUserID() *uuid.UUID {
	return c.userID
}

// IsAuthenticated returns whether the client is authenticated
func (c *Client) IsAuthenticated() bool {
	return c.authenticated
}

// SetAuthenticated sets the authentication status and user ID
func (c *Client) SetAuthenticated(userID uuid.UUID) {
	c.userID = &userID
	c.authenticated = true
}

// IsSubscribed checks if the client is subscribed to a topic
func (c *Client) IsSubscribed(topic string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.subscriptions[topic]
}

// Subscribe adds a subscription for the client
func (c *Client) Subscribe(topic string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.subscriptions[topic] = true
	c.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"topic":     topic,
	}).Debug("Client subscribed to topic")
}

// Unsubscribe removes a subscription for the client
func (c *Client) Unsubscribe(topic string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.subscriptions, topic)
	c.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"topic":     topic,
	}).Debug("Client unsubscribed from topic")
}

// GetSubscriptions returns all client subscriptions
func (c *Client) GetSubscriptions() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	return topics
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(messageBytes, &clientMsg); err != nil {
			c.logger.WithError(err).Warn("Failed to parse client message")
			c.sendError("Invalid message format")
			continue
		}

		// Handle the message
		c.handleMessage(clientMsg)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message
			if err := c.conn.WriteJSON(message); err != nil {
				c.logger.WithError(err).Error("Failed to write message")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Error("Failed to write ping")
				return
			}
		}
	}
}

// handleMessage processes messages received from the client
func (c *Client) handleMessage(msg ClientMessage) {
	c.logger.WithFields(logrus.Fields{
		"client_id":    c.id,
		"message_type": msg.Type,
		"message_id":   msg.ID,
	}).Debug("Received client message")

	switch msg.Type {
	case "ping":
		c.sendPong(msg.ID)

	case "subscribe":
		c.handleSubscription(msg)

	case "unsubscribe":
		c.handleUnsubscription(msg)

	case "auth":
		c.handleAuthentication(msg)

	case "get_status":
		c.handleStatusRequest(msg)

	default:
		c.logger.WithField("message_type", msg.Type).Warn("Unknown message type")
		c.sendError("Unknown message type")
	}
}

// handleSubscription handles subscription requests
func (c *Client) handleSubscription(msg ClientMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError("Invalid subscription data")
		return
	}

	topicsInterface, ok := data["topics"]
	if !ok {
		c.sendError("Missing topics in subscription")
		return
	}

	topics, ok := topicsInterface.([]interface{})
	if !ok {
		c.sendError("Invalid topics format")
		return
	}

	// Subscribe to each topic
	for _, topicInterface := range topics {
		topic, ok := topicInterface.(string)
		if !ok {
			continue
		}
		c.Subscribe(topic)
	}

	// Send confirmation
	c.sendSubscriptionConfirmation(msg.ID, c.GetSubscriptions())
}

// handleUnsubscription handles unsubscription requests
func (c *Client) handleUnsubscription(msg ClientMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError("Invalid unsubscription data")
		return
	}

	topicsInterface, ok := data["topics"]
	if !ok {
		c.sendError("Missing topics in unsubscription")
		return
	}

	topics, ok := topicsInterface.([]interface{})
	if !ok {
		c.sendError("Invalid topics format")
		return
	}

	// Unsubscribe from each topic
	for _, topicInterface := range topics {
		topic, ok := topicInterface.(string)
		if !ok {
			continue
		}
		c.Unsubscribe(topic)
	}

	// Send confirmation
	c.sendSubscriptionConfirmation(msg.ID, c.GetSubscriptions())
}

// handleAuthentication handles authentication requests
func (c *Client) handleAuthentication(msg ClientMessage) {
	// TODO: Implement JWT token validation
	// For now, we'll accept any authentication attempt
	c.SetAuthenticated(uuid.New())

	c.sendAuthConfirmation(msg.ID)
}

// handleStatusRequest handles status requests
func (c *Client) handleStatusRequest(msg ClientMessage) {
	status := map[string]interface{}{
		"client_id":     c.id,
		"authenticated": c.authenticated,
		"subscriptions": c.GetSubscriptions(),
		"timestamp":     c.getTimestamp(),
	}

	response := Message{
		Type:      MessageTypeSystemStatus,
		Data:      status,
		Timestamp: c.getTimestamp(),
		ID:        msg.ID,
	}

	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send status response: channel full")
	}
}

// sendPong sends a pong response
func (c *Client) sendPong(messageID string) {
	response := Message{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": c.getTimestamp(),
		},
		Timestamp: c.getTimestamp(),
		ID:        messageID,
	}

	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send pong: channel full")
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMsg string) {
	response := Message{
		Type: MessageTypeError,
		Data: map[string]interface{}{
			"error": errorMsg,
		},
		Timestamp: c.getTimestamp(),
	}

	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send error: channel full")
	}
}

// sendSubscriptionConfirmation sends subscription confirmation
func (c *Client) sendSubscriptionConfirmation(messageID string, subscriptions []string) {
	response := Message{
		Type: "subscription_confirmed",
		Data: map[string]interface{}{
			"subscriptions": subscriptions,
		},
		Timestamp: c.getTimestamp(),
		ID:        messageID,
	}

	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send subscription confirmation: channel full")
	}
}

// sendAuthConfirmation sends authentication confirmation
func (c *Client) sendAuthConfirmation(messageID string) {
	response := Message{
		Type: "auth_confirmed",
		Data: map[string]interface{}{
			"authenticated": true,
			"user_id":       c.userID,
		},
		Timestamp: c.getTimestamp(),
		ID:        messageID,
	}

	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send auth confirmation: channel full")
	}
}

// getTimestamp returns current timestamp in milliseconds
func (c *Client) getTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
