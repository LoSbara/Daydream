package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"daydream/internal/auth"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS gestito da Gin
	},
}

// Announcement è un messaggio broadcast a tutti i client WS connessi.
type Announcement struct {
	Type  string `json:"type"`  // "announcement"
	Text  string `json:"text"`
	Level string `json:"level"` // "info" | "warning" | "special"
}

// client rappresenta una singola connessione WebSocket autenticata.
type client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

// Hub gestisce l'insieme dei client WS connessi e il broadcast.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*client]bool
	broadcast  chan []byte
	register   chan *client
	unregister chan *client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *client, 16),
		unregister: make(chan *client, 16),
	}
}

// Run avvia la goroutine del hub. Chiamare con go hub.Run().
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			log.Printf("[ws] client connesso: %s (totale: %d)", c.userID, h.count())

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			log.Printf("[ws] client disconnesso: %s (totale: %d)", c.userID, h.count())

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Buffer pieno: disconnetti il client
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, c)
					close(c.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast invia un Announcement a tutti i client connessi.
func (h *Hub) Broadcast(a Announcement) {
	data, err := json.Marshal(a)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
		log.Println("[ws] broadcast channel pieno, messaggio scartato")
	}
}

// ConnectedCount restituisce il numero di client connessi.
func (h *Hub) ConnectedCount() int { return h.count() }

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS è il Gin handler per GET /ws?token=<jwt>.
func (h *Hub) ServeWS(c *gin.Context) {
	// Autenticazione via query param (browser WS non supporta header Authorization)
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token mancante"})
		return
	}
	claims, err := auth.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token non valido"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade fallito: %v", err)
		return
	}

	cl := &client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 64),
		userID: claims.Subject,
	}
	h.register <- cl

	go cl.writePump()
	go cl.readPump()
}

// readPump legge messaggi dal client (solo ping/pong, il client non invia dati).
func (cl *client) readPump() {
	defer func() {
		cl.hub.unregister <- cl
		cl.conn.Close()
	}()
	cl.conn.SetReadLimit(maxMessageSize)
	cl.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint
	cl.conn.SetPongHandler(func(string) error {
		cl.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint
		return nil
	})
	for {
		_, _, err := cl.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error %s: %v", cl.userID, err)
			}
			break
		}
	}
}

// writePump invia messaggi al client.
func (cl *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		cl.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-cl.send:
			cl.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint
			if !ok {
				cl.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint
				return
			}
			if err := cl.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			cl.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint
			if err := cl.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
