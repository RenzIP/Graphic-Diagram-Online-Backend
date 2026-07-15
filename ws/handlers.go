package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	gws "github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTValidator validates JWT tokens for WebSocket authentication
type JWTValidator struct {
	Secret string
}

// ValidateToken parses and validates a JWT token, returning the user ID, role,
// and username (from the "username" claim; empty if absent).
func (v *JWTValidator) ValidateToken(tokenStr string) (userID uuid.UUID, role, username string, err error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(v.Secret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, "", "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", "", fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return uuid.Nil, "", "", fmt.Errorf("missing sub claim")
	}

	userID, err = uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("invalid user ID in token: %w", err)
	}

	if r, ok := claims["role"].(string); ok {
		role = r
	}
	if u, ok := claims["username"].(string); ok {
		username = u
	}

	return userID, role, username, nil
}

// Authorizer resolves a user's role for a document's workspace and their
// display name. Satisfied structurally by *service.DocumentService.
// AuthorizeDocumentAccess returns the workspace role ("owner"/"editor"/"viewer")
// or an error when the document does not exist or the user is not a member.
// ResolveFirstName returns the user's first name (from their profile) and is
// used to label collaborators when the JWT has no username claim.
type Authorizer interface {
	AuthorizeDocumentAccess(ctx context.Context, userID, docID uuid.UUID) (string, error)
	ResolveFirstName(ctx context.Context, userID uuid.UUID) string
}

// UpgradeMiddleware checks for WebSocket upgrade requests
func UpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if gws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleWebSocket returns a Fiber handler for WebSocket connections.
// authz may be nil (membership checks are then skipped), but in production
// it should be a *service.DocumentService so only workspace members can join.
func HandleWebSocket(hub *Hub, validator *JWTValidator, authz Authorizer) fiber.Handler {
	return gws.New(func(c *gws.Conn) {
		// Extract token from query parameter
		tokenStr := c.Query("token")
		if tokenStr == "" {
			log.Printf("[WS] Connection rejected: missing token")
			c.Close()
			return
		}

		// Validate JWT token
		userID, role, username, err := validator.ValidateToken(tokenStr)
		if err != nil {
			log.Printf("[WS] Connection rejected: invalid token: %v", err)
			c.Close()
			return
		}

		documentID := c.Params("documentId")

		// Authorize: the user must be a member of the document's workspace.
		// wsRole ("owner"/"editor"/"viewer") gates mutations below.
		wsRole := "editor"
		if authz != nil {
			docUUID, err := uuid.Parse(documentID)
			if err != nil {
				log.Printf("[WS] Connection rejected: invalid document id %q", documentID)
				c.Close()
				return
			}
			r, authErr := authz.AuthorizeDocumentAccess(context.Background(), userID, docUUID)
			if authErr != nil {
				log.Printf("[WS] Connection rejected: user %s not authorized for document %s: %v", userID, documentID, authErr)
				c.Close()
				return
			}
			wsRole = r
		}

		clientID := uuid.New().String()

		// Label the collaborator: prefer the JWT username; if the token lacks
		// one, fall back to the user's first name from their profile; finally
		// a synthetic label if neither is available.
		name := username
		if name == "" && authz != nil {
			name = authz.ResolveFirstName(context.Background(), userID)
		}
		if name == "" {
			name = "User-" + clientID[:8]
		}

		client := &Client{
			ID:     clientID,
			UserID: userID.String(),
			Role:   role,
			WsRole: wsRole,
			Name:   name,
			Conn:   c.Conn,
			Room:   documentID,
		}

		room := hub.GetOrCreateRoom(documentID)
		room.AddClient(client)

		log.Printf("[WS] Client %s (user=%s, role=%s) joined room %s", clientID, userID, role, documentID)

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

	// Input validation
	if err := validateMessage(&msg); err != nil {
		log.Printf("[WS] Invalid message from client %s: %v", client.ID, err)
		sendError(client, "Invalid message: "+err.Error())
		return
	}

	// Authorization: viewers may observe (cursor + join only) but not mutate,
	// mirroring the "viewers cannot edit" rule enforced on the REST paths.
	if client.WsRole == "viewer" && isMutation(msg.Type) {
		sendError(client, "viewers cannot edit this document")
		return
	}

	switch msg.Type {
	case TypeLockNode:
		if msg.NodeID == "" {
			sendError(client, "node_id is required for lock_node")
			return
		}
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
		if msg.NodeID == "" {
			sendError(client, "node_id is required for unlock_node")
			return
		}
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
		if msg.NodeID == "" || msg.Changes == nil {
			sendError(client, "node_id and changes are required for update_node")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type:    TypeNodeUpdated,
			NodeID:  msg.NodeID,
			Changes: msg.Changes,
		})

	case TypeAddNode:
		if msg.Node == nil {
			sendError(client, "node is required for add_node")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type: TypeNodeAdded,
			Node: msg.Node,
		})

	case TypeDeleteNode:
		if msg.NodeID == "" {
			sendError(client, "node_id is required for delete_node")
			return
		}
		room.UnlockNode(msg.NodeID, client.ID)
		if hub.Locks != nil {
			_ = hub.Locks.UnlockNode(ctx, client.Room, msg.NodeID, client.ID)
		}
		broadcastJSON(room, client.ID, Message{
			Type:   TypeNodeDeleted,
			NodeID: msg.NodeID,
		})

	case TypeAddEdge:
		if msg.Edge == nil {
			sendError(client, "edge is required for add_edge")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type: TypeEdgeAdded,
			Edge: msg.Edge,
		})

	case TypeUpdateEdge:
		if msg.EdgeID == "" || msg.Changes == nil {
			sendError(client, "edge_id and changes are required for update_edge")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type:    TypeEdgeUpdated,
			EdgeID:  msg.EdgeID,
			Changes: msg.Changes,
		})

	case TypeDeleteEdge:
		if msg.EdgeID == "" {
			sendError(client, "edge_id is required for delete_edge")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type:   TypeEdgeDeleted,
			EdgeID: msg.EdgeID,
		})

	case TypeReplaceDocument:
		if msg.State == nil {
			sendError(client, "state is required for replace_document")
			return
		}
		broadcastJSON(room, client.ID, Message{
			Type:  TypeReplaceDocument,
			State: msg.State,
		})

	case TypeCursorMove:
		// Cursor moves are validated by validateMessage()
		
		// Rate limit cursor updates to prevent spam (max 30 updates per second per client)
		now := time.Now()
		if lastUpdate, ok := hub.cursorRate.Load(client.ID); ok {
			if now.Sub(lastUpdate.(time.Time)) < 33*time.Millisecond {
				// Too frequent, skip this update
				return
			}
		}
		hub.cursorRate.Store(client.ID, now)
		
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
		log.Printf("[WS] Failed to marshal message: %v", err)
		return
	}
	room.Broadcast(data, excludeID)
}

func sendError(client *Client, message string) {
	data, _ := json.Marshal(Message{
		Type:        TypeError,
		MessageText: message,
	})
	if err := client.Send(data); err != nil {
		log.Printf("[WS] Failed to send error to client %s: %v", client.ID, err)
	}
}

// validateMessage performs basic validation on incoming WebSocket messages
func validateMessage(msg *Message) error {
	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}
	
	// Validate message size (prevent DoS via oversized messages)
	// This is already limited by WebSocket frame size, but we add an extra check
	
	// Type-specific validation
	switch msg.Type {
	case TypeCursorMove:
		// Cursor coordinates should be reasonable (not NaN, not infinite)
		// Go's JSON unmarshal will reject NaN/Inf, but we check bounds
		if msg.X < -100000 || msg.X > 100000 || msg.Y < -100000 || msg.Y > 100000 {
			return fmt.Errorf("cursor coordinates out of bounds")
		}
	}
	
	return nil
}

