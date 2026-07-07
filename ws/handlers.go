package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	gws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// UpgradeMiddleware checks for WebSocket upgrade requests
func UpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if gws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleWebSocket returns a Fiber handler for WebSocket connections
func HandleWebSocket(hub *Hub) fiber.Handler {
	return gws.New(func(c *gws.Conn) {
		documentID := c.Params("documentId")
		clientID := uuid.New().String()

		client := &Client{
			ID:   clientID,
			Name: "User-" + clientID[:8],
			Conn: c.Conn,
			Room: documentID,
		}

		room := hub.GetOrCreateRoom(documentID)
		room.AddClient(client)

		log.Printf("[WS] Client %s joined room %s", clientID, documentID)

		// --- Redis: register presence ---
		ctx := context.Background()
		if hub.Presence != nil {
			if err := hub.Presence.SetPresence(ctx, documentID, clientID, client.Name); err != nil {
				log.Printf("[WS] Redis presence set error: %v", err)
			}
			if err := hub.Presence.SetHeartbeat(ctx, documentID, clientID); err != nil {
				log.Printf("[WS] Redis heartbeat set error: %v", err)
			}
		}

		// --- Heartbeat goroutine: keep Redis presence alive ---
		heartbeatDone := make(chan struct{})
		if hub.Presence != nil {
			go func() {
				ticker := time.NewTicker(15 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-heartbeatDone:
						return
					case <-ticker.C:
						hbCtx := context.Background()
						_ = hub.Presence.SetHeartbeat(hbCtx, documentID, clientID)
					}
				}
			}()
		}

		// Send room state to the new client
		sendRoomState(client, room, hub)

		// Notify others
		broadcastJSON(room, clientID, Message{
			Type: TypeUserJoined,
			User: map[string]interface{}{
				"id":   clientID,
				"name": client.Name,
			},
		})

		// Read loop
		defer func() {
			// Stop heartbeat
			close(heartbeatDone)

			room.RemoveClient(clientID)
			hub.RemoveRoomIfEmpty(documentID)

			// --- Redis: clean up presence and locks ---
			cleanCtx := context.Background()
			if hub.Presence != nil {
				if err := hub.Presence.RemovePresence(cleanCtx, documentID, clientID); err != nil {
					log.Printf("[WS] Redis presence remove error: %v", err)
				}
			}
			if hub.Locks != nil {
				// Release all Redis locks held by this client
				locks, err := hub.Locks.GetRoomLocks(cleanCtx, documentID)
				if err == nil {
					for nodeID, lockerID := range locks {
						if lockerID == clientID {
							_ = hub.Locks.UnlockNode(cleanCtx, documentID, nodeID, clientID)
						}
					}
				}
			}

			broadcastJSON(room, "", Message{
				Type:   TypeUserLeft,
				UserID: clientID,
			})

			log.Printf("[WS] Client %s left room %s", clientID, documentID)
		}()

		for {
			_, msgBytes, err := c.ReadMessage()
			if err != nil {
				break
			}

			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				sendError(client, "Invalid message format")
				continue
			}

			handleMessage(client, room, hub, msg)
		}
	})
}

func handleMessage(client *Client, room *Room, hub *Hub, msg Message) {
	ctx := context.Background()

	switch msg.Type {
	case TypeLockNode:
		// In-memory lock
		if !room.LockNode(msg.NodeID, client.ID) {
			sendError(client, "Node is already locked")
			return
		}
		// Redis lock (best-effort write-through)
		if hub.Locks != nil {
			ok, err := hub.Locks.LockNode(ctx, client.Room, msg.NodeID, client.ID)
			if err != nil {
				log.Printf("[WS] Redis lock error: %v", err)
			} else if !ok {
				// Redis says someone else holds it — rollback in-memory
				room.UnlockNode(msg.NodeID, client.ID)
				sendError(client, "Node is already locked")
				return
			}
		}
		broadcastJSON(room, "", Message{
			Type:   TypeNodeLocked,
			NodeID: msg.NodeID,
			By:     client.ID,
		})

	case TypeUnlockNode:
		room.UnlockNode(msg.NodeID, client.ID)
		if hub.Locks != nil {
			if err := hub.Locks.UnlockNode(ctx, client.Room, msg.NodeID, client.ID); err != nil {
				log.Printf("[WS] Redis unlock error: %v", err)
			}
		}
		broadcastJSON(room, "", Message{
			Type:   TypeNodeUnlocked,
			NodeID: msg.NodeID,
		})

	case TypeUpdateNode:
		broadcastJSON(room, client.ID, Message{
			Type:    TypeNodeUpdated,
			NodeID:  msg.NodeID,
			Changes: msg.Changes,
		})

	case TypeAddNode:
		broadcastJSON(room, client.ID, Message{
			Type: TypeNodeAdded,
			Node: msg.Node,
		})

	case TypeDeleteNode:
		room.UnlockNode(msg.NodeID, client.ID)
		if hub.Locks != nil {
			_ = hub.Locks.UnlockNode(ctx, client.Room, msg.NodeID, client.ID)
		}
		broadcastJSON(room, client.ID, Message{
			Type:   TypeNodeDeleted,
			NodeID: msg.NodeID,
		})

	case TypeAddEdge:
		broadcastJSON(room, client.ID, Message{
			Type: "edge_added",
			Edge: msg.Edge,
		})

	case TypeDeleteEdge:
		broadcastJSON(room, client.ID, Message{
			Type:   "edge_deleted",
			EdgeID: msg.EdgeID,
		})

	case TypeCursorMove:
		broadcastJSON(room, client.ID, Message{
			Type:   TypeCursorUpdate,
			UserID: client.ID,
			X:      msg.X,
			Y:      msg.Y,
		})

	default:
		sendError(client, "Unknown message type: "+msg.Type)
	}
}

func sendRoomState(client *Client, room *Room, hub *Hub) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	users := make([]interface{}, 0, len(room.Clients))
	for _, c := range room.Clients {
		if c.ID != client.ID {
			users = append(users, map[string]interface{}{
				"id":   c.ID,
				"name": c.Name,
			})
		}
	}

	// Merge in-memory locks with Redis locks for the most complete picture
	locks := make(map[string]string, len(room.Locks))
	for k, v := range room.Locks {
		locks[k] = v
	}
	if hub.Locks != nil {
		ctx := context.Background()
		redisLocks, err := hub.Locks.GetRoomLocks(ctx, room.ID)
		if err == nil {
			for k, v := range redisLocks {
				if _, exists := locks[k]; !exists {
					locks[k] = v
				}
			}
		}
	}

	msg := Message{
		Type:  TypeRoomState,
		Users: users,
		Locks: locks,
	}

	data, _ := json.Marshal(msg)
	_ = client.Send(data)
}

func broadcastJSON(room *Room, excludeID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	room.Broadcast(data, excludeID)
}

func sendError(client *Client, message string) {
	data, _ := json.Marshal(Message{
		Type:        TypeError,
		MessageText: message,
	})
	_ = client.Send(data)
}

