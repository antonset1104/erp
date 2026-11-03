package events

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DocumentPresence records an active user viewing or editing a specific ERP document
type DocumentPresence struct {
	DocumentType string    `json:"document_type"` // "sale_order", "purchase_order", "account_move", "mrp_production"
	DocumentID   string    `json:"document_id"`
	UserID       uint      `json:"user_id"`
	UserName     string    `json:"user_name"`
	UserEmail    string    `json:"user_email"`
	Action       string    `json:"action"` // "viewing", "editing", "approving"
	LastActive   time.Time `json:"last_active"`
}

// PresenceRequest is the payload for POST /api/v1/events/presence
type PresenceRequest struct {
	DocumentType string `json:"document_type" binding:"required"`
	DocumentID   string `json:"document_id" binding:"required"`
	Action       string `json:"action"` // "viewing", "editing"
}

// Broker manages real-time Server-Sent Events (SSE) connections and document presence
type Broker struct {
	mu           sync.RWMutex
	clients      map[chan string]bool
	presence     map[string]*DocumentPresence // key = docType:docID:userID
	presenceLock sync.RWMutex
}

var globalBroker *Broker
var brokerOnce sync.Once

// GetBroker returns the singleton event broker
func GetBroker() *Broker {
	brokerOnce.Do(func() {
		globalBroker = &Broker{
			clients:  make(map[chan string]bool),
			presence: make(map[string]*DocumentPresence),
		}
		// Background presence cleaner
		go globalBroker.cleanExpiredPresence()
	})
	return globalBroker
}

func (b *Broker) cleanExpiredPresence() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		b.presenceLock.Lock()
		now := time.Now()
		for key, p := range b.presence {
			if now.Sub(p.LastActive) > 45*time.Second {
				delete(b.presence, key)
			}
		}
		b.presenceLock.Unlock()
	}
}

// Subscribe adds a client channel
func (b *Broker) Subscribe() chan string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan string, 32)
	b.clients[ch] = true
	return ch
}

// Unsubscribe removes a client channel
func (b *Broker) Unsubscribe(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
}

// Broadcast sends a formatted Server-Sent Event to all active SSE subscribers
func (b *Broker) Broadcast(eventName string, payload interface{}) {
	dataBytes, _ := json.Marshal(payload)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, string(dataBytes))

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Non-blocking drop if client buffer is full
		}
	}
}

// UpdatePresence records user activity on a document and broadcasts a presence event
func (b *Broker) UpdatePresence(docType, docID string, userID uint, userName, userEmail, action string) {
	b.presenceLock.Lock()
	key := fmt.Sprintf("%s:%s:%d", docType, docID, userID)
	if action == "" {
		action = "viewing"
	}
	p := &DocumentPresence{
		DocumentType: docType,
		DocumentID:   docID,
		UserID:       userID,
		UserName:     userName,
		UserEmail:    userEmail,
		Action:       action,
		LastActive:   time.Now(),
	}
	b.presence[key] = p
	b.presenceLock.Unlock()

	b.Broadcast("presence_update", p)
}

// GetDocumentPresences returns all active users on a specific document
func (b *Broker) GetDocumentPresences(docType, docID string) []*DocumentPresence {
	b.presenceLock.RLock()
	defer b.presenceLock.RUnlock()
	now := time.Now()
	res := []*DocumentPresence{}
	for _, p := range b.presence {
		if p.DocumentType == docType && p.DocumentID == docID && now.Sub(p.LastActive) <= 45*time.Second {
			res = append(res, p)
		}
	}
	return res
}

// RegisterEventRoutes registers SSE streaming and presence endpoints on Gin
func RegisterEventRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	broker := GetBroker()
	group := router.Group("/events")
	{
		// GET /api/v1/events/stream - Server-Sent Events (SSE) stream
		group.GET("/stream", func(c *gin.Context) {
			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("Connection", "keep-alive")
			c.Writer.Header().Set("Transfer-Encoding", "chunked")
			c.Writer.Header().Set("X-Accel-Buffering", "no")

			clientChan := broker.Subscribe()
			defer broker.Unsubscribe(clientChan)

			// Initial welcome event
			c.Writer.WriteString(fmt.Sprintf("event: connected\ndata: {\"status\":\"connected\",\"timestamp\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339)))
			c.Writer.Flush()

			notify := c.Request.Context().Done()
			ticker := time.NewTicker(15 * time.Second) // Keepalive heartbeat
			defer ticker.Stop()

			c.Stream(func(w io.Writer) bool {
				select {
				case <-notify:
					return false
				case <-ticker.C:
					c.Writer.WriteString(": keepalive\n\n")
					c.Writer.Flush()
					return true
				case msg, ok := <-clientChan:
					if !ok {
						return false
					}
					c.Writer.WriteString(msg)
					c.Writer.Flush()
					return true
				}
			})
		})

		// Authenticated presence and broadcast endpoints
		authed := group.Group("")
		authed.Use(authMiddleware)
		{
			// POST /api/v1/events/presence - Report editing or viewing activity
			authed.POST("/presence", func(c *gin.Context) {
				var req PresenceRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				userName := c.GetString("user_name")
				if userName == "" {
					userName = "User"
				}
				userEmail := c.GetString("user_email")
				userID := c.GetUint("user_id")
				if userID == 0 {
					userID = 1
				}

				broker.UpdatePresence(req.DocumentType, req.DocumentID, userID, userName, userEmail, req.Action)
				active := broker.GetDocumentPresences(req.DocumentType, req.DocumentID)
				c.JSON(http.StatusOK, gin.H{
					"success":      true,
					"active_users": active,
				})
			})

			// GET /api/v1/events/presence - Query active users on document
			authed.GET("/presence", func(c *gin.Context) {
				docType := c.Query("document_type")
				docID := c.Query("document_id")
				if docType == "" || docID == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "document_type and document_id query params required"})
					return
				}
				active := broker.GetDocumentPresences(docType, docID)
				c.JSON(http.StatusOK, gin.H{
					"document_type": docType,
					"document_id":   docID,
					"total_active":  len(active),
					"users":         active,
				})
			})

			// POST /api/v1/events/publish - Broadcast custom domain event
			authed.POST("/publish", func(c *gin.Context) {
				var payload struct {
					Event string                 `json:"event" binding:"required"`
					Data  map[string]interface{} `json:"data" binding:"required"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				broker.Broadcast(payload.Event, payload.Data)
				c.JSON(http.StatusOK, gin.H{
					"success":   true,
					"event":     payload.Event,
					"broadcast": "delivered_to_subscribers",
				})
			})
		}
	}
}
