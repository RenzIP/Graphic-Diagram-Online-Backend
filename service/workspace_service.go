package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/repository"
)

// WorkspaceService handles workspace business logic with authorization.
type WorkspaceService struct {
	wsRepo   *repository.WorkspaceRepo
	userRepo *repository.UserRepo
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(wsRepo *repository.WorkspaceRepo, userRepo *repository.UserRepo) *WorkspaceService {
	return &WorkspaceService{wsRepo: wsRepo, userRepo: userRepo}
}

// ListByUser returns paginated workspaces the user belongs to.
func (s *WorkspaceService) ListByUser(ctx context.Context, userID uuid.UUID, pq dto.PaginationQuery) (*dto.WorkspaceListResp, *pkg.AppError) {
	workspaces, total, appErr := s.wsRepo.FindByMember(ctx, userID, pq.PerPage, pq.Offset())
	if appErr != nil {
		return nil, appErr
	}

	items := make([]dto.WorkspaceListItem, 0, len(workspaces))
	for _, ws := range workspaces {
		// Get the user's role in this workspace
		role, roleErr := s.wsRepo.GetMemberRole(ctx, ws.ID, userID)
		if roleErr != nil {
			return nil, roleErr
		}

		// Count members for this workspace
		memberCount, countErr := s.wsRepo.CountMembers(ctx, ws.ID)
		if countErr != nil {
			return nil, countErr
		}

		items = append(items, dto.WorkspaceListItem{
			ID:          ws.ID.String(),
			Name:        ws.Name,
			Slug:        ws.Slug,
			OwnerID:     ws.OwnerID.String(),
			Description: ws.Description,
			Role:        role,
			MemberCount: memberCount,
			CreatedAt:   ws.CreatedAt,
			UpdatedAt:   ws.UpdatedAt,
		})
	}

	meta := dto.NewPaginationMeta(pq, total)
	return &dto.WorkspaceListResp{Data: items, Meta: meta}, nil
}

// Create creates a new workspace with the current user as owner.
// Auto-generates slug from name and inserts owner membership in a transaction.
func (s *WorkspaceService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateWorkspaceReq) (*dto.WorkspaceResp, *pkg.AppError) {
	// Validate request
	if appErr := pkg.Validate(req); appErr != nil {
		return nil, appErr
	}

	slug := pkg.GenerateSlug(req.Name)

	// Check slug uniqueness
	existing, _ := s.wsRepo.FindBySlug(ctx, slug)
	if existing != nil {
		return nil, pkg.ErrConflict.WithMessage("workspace slug already exists: " + slug)
	}

	ws := &model.Workspace{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        slug,
		OwnerID:     userID,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	member := &model.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      userID,
		Role:        "owner",
		JoinedAt:    time.Now(),
	}

	if appErr := s.wsRepo.InsertWithOwner(ctx, ws, member); appErr != nil {
		return nil, appErr
	}

	return toWorkspaceResp(ws), nil
}

// Update modifies workspace name/description. Owner only.
func (s *WorkspaceService) Update(ctx context.Context, userID, workspaceID uuid.UUID, req dto.UpdateWorkspaceReq) (*dto.WorkspaceResp, *pkg.AppError) {
	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return nil, appErr
	}

	// Only the owner can update a workspace
	if ws.OwnerID != userID {
		return nil, pkg.ErrForbidden.WithMessage("only the workspace owner can update it")
	}

	if req.Name != nil {
		ws.Name = *req.Name
		ws.Slug = pkg.GenerateSlug(*req.Name)

		// Check slug uniqueness (excluding current workspace)
		existing, _ := s.wsRepo.FindBySlug(ctx, ws.Slug)
		if existing != nil && existing.ID != ws.ID {
			return nil, pkg.ErrConflict.WithMessage("workspace slug already exists: " + ws.Slug)
		}
	}
	if req.Description != nil {
		ws.Description = req.Description
	}
	ws.UpdatedAt = time.Now()

	if appErr := s.wsRepo.Update(ctx, ws); appErr != nil {
		return nil, appErr
	}

	return toWorkspaceResp(ws), nil
}

// Delete removes a workspace. Owner only. CASCADE removes projects/documents.
func (s *WorkspaceService) Delete(ctx context.Context, userID, workspaceID uuid.UUID) *pkg.AppError {
	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return appErr
	}

	if ws.OwnerID != userID {
		return pkg.ErrForbidden.WithMessage("only the workspace owner can delete it")
	}

	return s.wsRepo.Delete(ctx, workspaceID)
}

// ListMembers returns every member of a workspace. Any member may view the list.
func (s *WorkspaceService) ListMembers(ctx context.Context, userID, workspaceID uuid.UUID) (*dto.MemberListResp, *pkg.AppError) {
	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return nil, appErr
	}
	if _, appErr := s.RequireMembership(ctx, workspaceID, userID); appErr != nil {
		return nil, appErr
	}

	rows, appErr := s.wsRepo.ListMembers(ctx, workspaceID)
	if appErr != nil {
		return nil, appErr
	}

	items := make([]dto.MemberResp, 0, len(rows))
	for _, m := range rows {
		items = append(items, dto.MemberResp{
			UserID:   m.UserID.String(),
			Name:     m.Name,
			Username: m.Username,
			Email:    m.Email,
			Avatar:   m.Avatar,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
			IsOwner:  m.UserID == ws.OwnerID,
		})
	}

	return &dto.MemberListResp{Data: items}, nil
}

// AddMember invites a user (by username or email) to a workspace. Owner only.
// The new member's role is limited to editor or viewer.
func (s *WorkspaceService) AddMember(ctx context.Context, actorID, workspaceID uuid.UUID, req dto.AddMemberReq) (*dto.MemberResp, *pkg.AppError) {
	if appErr := pkg.Validate(req); appErr != nil {
		return nil, appErr
	}

	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return nil, appErr
	}
	if ws.OwnerID != actorID {
		return nil, pkg.ErrForbidden.WithMessage("only the workspace owner can add members")
	}

	invitee, appErr := s.userRepo.FindByUsernameOrEmail(ctx, req.Identifier)
	if appErr != nil {
		return nil, appErr
	}
	if invitee == nil {
		return nil, pkg.ErrNotFound.WithMessage("no user found with that username or email")
	}

	// Reject if the user is already a member.
	existingRole, appErr := s.wsRepo.GetMemberRole(ctx, workspaceID, invitee.ID)
	if appErr != nil {
		return nil, appErr
	}
	if existingRole != "" {
		return nil, pkg.ErrConflict.WithMessage("user is already a member of this workspace")
	}

	member := &model.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      invitee.ID,
		Role:        req.Role,
		JoinedAt:    time.Now(),
	}
	if appErr := s.wsRepo.InsertMember(ctx, member); appErr != nil {
		return nil, appErr
	}

	return &dto.MemberResp{
		UserID:   invitee.ID.String(),
		Name:     invitee.Name,
		Username: invitee.Username,
		Email:    invitee.Email,
		Avatar:   invitee.Avatar,
		Role:     member.Role,
		JoinedAt: member.JoinedAt,
		IsOwner:  false,
	}, nil
}

// UpdateMemberRole changes an existing member's role. Owner only.
// The owner's own role cannot be changed (ownership transfer is out of scope).
func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, actorID, workspaceID, targetID uuid.UUID, req dto.UpdateMemberRoleReq) *pkg.AppError {
	if appErr := pkg.Validate(req); appErr != nil {
		return appErr
	}

	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return appErr
	}
	if ws.OwnerID != actorID {
		return pkg.ErrForbidden.WithMessage("only the workspace owner can change member roles")
	}
	if targetID == ws.OwnerID {
		return pkg.ErrBadRequest.WithMessage("the workspace owner's role cannot be changed")
	}

	role, appErr := s.wsRepo.GetMemberRole(ctx, workspaceID, targetID)
	if appErr != nil {
		return appErr
	}
	if role == "" {
		return pkg.ErrNotFound.WithMessage("user is not a member of this workspace")
	}

	return s.wsRepo.UpdateMemberRole(ctx, workspaceID, targetID, req.Role)
}

// RemoveMember removes a member from a workspace. The owner can remove anyone
// (except themselves); any member can remove themselves (leave the workspace).
func (s *WorkspaceService) RemoveMember(ctx context.Context, actorID, workspaceID, targetID uuid.UUID) *pkg.AppError {
	ws, appErr := s.wsRepo.FindByID(ctx, workspaceID)
	if appErr != nil {
		return appErr
	}

	if targetID == ws.OwnerID {
		return pkg.ErrBadRequest.WithMessage("the workspace owner cannot be removed")
	}

	// Either the owner is removing someone, or a member is removing themselves.
	if actorID != ws.OwnerID && actorID != targetID {
		return pkg.ErrForbidden.WithMessage("you cannot remove another member")
	}

	role, appErr := s.wsRepo.GetMemberRole(ctx, workspaceID, targetID)
	if appErr != nil {
		return appErr
	}
	if role == "" {
		return pkg.ErrNotFound.WithMessage("user is not a member of this workspace")
	}

	return s.wsRepo.RemoveMember(ctx, workspaceID, targetID)
}

// RequireMembership checks that the user is a member of the workspace and returns the role.
// Returns ErrForbidden if not a member.
func (s *WorkspaceService) RequireMembership(ctx context.Context, workspaceID, userID uuid.UUID) (string, *pkg.AppError) {
	role, appErr := s.wsRepo.GetMemberRole(ctx, workspaceID, userID)
	if appErr != nil {
		return "", appErr
	}
	if role == "" {
		return "", pkg.ErrForbidden.WithMessage("you are not a member of this workspace")
	}
	return role, nil
}

func toWorkspaceResp(ws *model.Workspace) *dto.WorkspaceResp {
	return &dto.WorkspaceResp{
		ID:          ws.ID.String(),
		Name:        ws.Name,
		Slug:        ws.Slug,
		OwnerID:     ws.OwnerID.String(),
		Description: ws.Description,
		CreatedAt:   ws.CreatedAt,
		UpdatedAt:   ws.UpdatedAt,
	}
}
