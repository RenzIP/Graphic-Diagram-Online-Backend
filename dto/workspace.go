package dto

import "time"

// CreateWorkspaceReq is the body for POST /api/workspaces.
type CreateWorkspaceReq struct {
	Name        string  `json:"name"        validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// UpdateWorkspaceReq is the body for PUT /api/workspaces/:id.
type UpdateWorkspaceReq struct {
	Name        *string `json:"name"        validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// WorkspaceResp is the response for a single workspace.
type WorkspaceResp struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerID     string    `json:"owner_id"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceListItem extends WorkspaceResp with membership context for list responses.
type WorkspaceListItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerID     string    `json:"owner_id"`
	Description *string   `json:"description"`
	Role        string    `json:"role"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceListResp is the paginated response for GET /api/workspaces.
type WorkspaceListResp struct {
	Data []WorkspaceListItem `json:"data"`
	Meta PaginationMeta      `json:"meta"`
}

// AddMemberReq is the body for POST /api/workspaces/:id/members.
// Identifier accepts either the invitee's username or email.
type AddMemberReq struct {
	Identifier string `json:"identifier" validate:"required,min=1,max=255"`
	Role       string `json:"role"       validate:"required,oneof=editor viewer"`
}

// UpdateMemberRoleReq is the body for PUT /api/workspaces/:id/members/:userId.
type UpdateMemberRoleReq struct {
	Role string `json:"role" validate:"required,oneof=owner editor viewer"`
}

// MemberResp describes a single workspace member with user details.
type MemberResp struct {
	UserID   string    `json:"user_id"`
	Name     *string   `json:"name"`
	Username *string   `json:"username"`
	Email    *string   `json:"email"`
	Avatar   *string   `json:"avatar"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	IsOwner  bool      `json:"is_owner"`
}

// MemberListResp is the response for GET /api/workspaces/:id/members.
type MemberListResp struct {
	Data []MemberResp `json:"data"`
}
