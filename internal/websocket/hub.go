package websocket

import (
	"context"
	"encoding/json"
	"sync"

	gws "github.com/gorilla/websocket" // Alias para evitar colisão
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	// Registered clients (user_id -> connections)
	clients map[string]map[*Client]bool

	// Room subscriptions (task_id -> clients)
	rooms map[string]map[*Client]bool

	// Register/Unregister requests
	Register   chan *Client
	Unregister chan *Client

	// Broadcast messages (Exportado para ser usado pelo Handler de Tasks)
	Broadcast chan *Message

	redis *redis.Client
	mu    sync.RWMutex
}

type Client struct {
	ID     string
	UserID string
	Conn   *gws.Conn // Usa o tipo da lib externa via alias
	Hub    *Hub
	Send   chan []byte
	Rooms  map[string]bool
}

type Message struct {
	Type      string          `json:"type"`
	TaskID    string          `json:"task_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	UserID    string          `json:"user_id"`
	Timestamp string          `json:"timestamp"` // String para facilitar serialização
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message, 256),
		redis:      redisClient,
	}
}

func (h *Hub) Run(ctx context.Context) {
	// Goroutine para Redis Pub/Sub (simplificada para o exemplo)
	pubsub := h.redis.Subscribe(ctx, "task_events")
	defer pubsub.Close()

	go func() {
		for msg := range pubsub.Channel() {
			var message Message
			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				continue
			}
			h.Broadcast <- &message
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; !ok {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.clients, client.UserID)
				}
			}
			for roomID := range client.Rooms {
				h.LeaveRoom(client, roomID)
			}
			close(client.Send)
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			// Envia para quem está na sala da Task
			if message.TaskID != "" {
				if room, ok := h.rooms[message.TaskID]; ok {
					data, _ := json.Marshal(message)
					for client := range room {
						select {
						case client.Send <- data:
						default:
							close(client.Send)
							delete(room, client)
						}
					}
				}
			}

			// ============================================================
			// NOVA LÓGICA (Adicione isto): Envia para o Dashboard do Usuário
			// ============================================================
			if message.UserID != "" {
				// Busca todas as conexões deste usuário
				if userClients, ok := h.clients[message.UserID]; ok {
					data, _ := json.Marshal(message)
					for client := range userClients {
						// Otimização opcional: Se o cliente já recebeu via "room",
						// aqui ele receberia de novo. O ideal é o frontend lidar com duplicatas
						// ou fazermos um mapa de "sentClients" aqui.
						// Para este teste, vamos enviar mesmo que duplique.

						select {
						case client.Send <- data:
						default:
							close(client.Send)
							delete(userClients, client)
						}
					}
				}
			}
			// ============================================================

			h.mu.RUnlock()

			// Aqui você publicaria no Redis para outras instâncias
			// data, _ := json.Marshal(message)
			// h.redis.Publish(ctx, "task_events", data)
		}
	}
}

func (h *Hub) JoinRoom(client *Client, taskID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[taskID]; !ok {
		h.rooms[taskID] = make(map[*Client]bool)
	}
	h.rooms[taskID][client] = true
	client.Rooms[taskID] = true
}

func (h *Hub) LeaveRoom(client *Client, taskID string) {
	// Nota: Assumindo que o chamador já tratou locks globais se necessário,
	// mas para segurança interna:
	if _, ok := h.rooms[taskID]; ok {
		delete(h.rooms[taskID], client)
		if len(h.rooms[taskID]) == 0 {
			delete(h.rooms, taskID)
		}
	}
	delete(client.Rooms, taskID)
}
