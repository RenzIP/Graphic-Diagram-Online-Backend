package ws

// Message types matching frontend WSMessageType from lib/ws/client.ts
const (
	// Client → Server
	TypeJoinRoom   = "join_room"
	TypeLockNode   = "lock_node"
	TypeUnlockNode = "unlock_node"
	TypeUpdateNode      = "update_node"
	TypeAddNode         = "add_node"
	TypeDeleteNode      = "delete_node"
	TypeAddEdge         = "add_edge"
	TypeUpdateEdge      = "update_edge"
	TypeDeleteEdge      = "delete_edge"
	TypeReplaceDocument = "replace_document"
	TypeCursorMove      = "cursor_move"

	// Server → Client
	TypeRoomState    = "room_state"
	TypeUserJoined   = "user_joined"
	TypeUserLeft     = "user_left"
	TypeNodeLocked   = "node_locked"
	TypeNodeUnlocked = "node_unlocked"
	TypeNodeUpdated  = "node_updated"
	TypeNodeAdded    = "node_added"
	TypeNodeDeleted  = "node_deleted"
	TypeEdgeAdded    = "edge_added"
	TypeEdgeUpdated  = "edge_updated"
	TypeEdgeDeleted  = "edge_deleted"
	TypeCursorUpdate = "cursor_update"
	TypeError        = "error"
)

// isMutation reports whether a client→server message type modifies document
// state (as opposed to observing it). Viewers are allowed everything else
// (join_room, cursor_move) but blocked from these, mirroring the REST rule
// that viewers cannot edit.
func isMutation(msgType string) bool {
	switch msgType {
	case TypeLockNode, TypeUnlockNode, TypeUpdateNode,
		TypeAddNode, TypeDeleteNode,
		TypeAddEdge, TypeUpdateEdge, TypeDeleteEdge,
		TypeReplaceDocument:
		return true
	default:
		return false
	}
}

// Message is a generic WebSocket message
type Message struct {
	Type string `json:"type"`

	// join_room
	RoomID string `json:"room_id,omitempty"`

	// node operations
	NodeID  string                 `json:"node_id,omitempty"`
	Node    map[string]interface{} `json:"node,omitempty"`
	Changes map[string]interface{} `json:"changes,omitempty"`

	// edge operations
	EdgeID string                 `json:"edge_id,omitempty"`
	Edge   map[string]interface{} `json:"edge,omitempty"`

	// cursor
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`

	// whole-document replace (import / version restore)
	State map[string]interface{} `json:"state,omitempty"`

	// server → client context
	UserID string                 `json:"user_id,omitempty"`
	By     string                 `json:"by,omitempty"`
	User   map[string]interface{} `json:"user,omitempty"`
	Users  []interface{}          `json:"users,omitempty"`
	Locks  map[string]string      `json:"locks,omitempty"`

	// error
	MessageText string `json:"message,omitempty"`
}
