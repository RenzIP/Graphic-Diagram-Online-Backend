package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/middleware"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
)

// WorkspaceHandler handles workspace CRUD endpoints.
type WorkspaceHandler struct {
	wsSvc *service.WorkspaceService
}

// NewWorkspaceHandler creates a new WorkspaceHandler.
func NewWorkspaceHandler(wsSvc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{wsSvc: wsSvc}
}

// List godoc
// @Summary      Get list of workspaces
// @Description  Retrieves all workspaces the current user is a member of
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int  false  "Page number"
// @Param        per_page  query     int  false  "Items per page"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /workspaces [get]
func (h *WorkspaceHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	pq := dto.ParsePagination(c.Query("page"), c.Query("per_page"))

	resp, appErr := h.wsSvc.ListByUser(c.UserContext(), userID, pq)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WritePaginated(c, resp.Data, resp.Meta.Page, resp.Meta.PerPage, resp.Meta.Total)
}

// Create godoc
// @Summary      Create a new workspace
// @Description  Creates a new workspace and sets the current user as owner
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateWorkspaceReq true "Workspace Details"
// @Success      201  {object}  model.Workspace
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /workspaces [post]
func (h *WorkspaceHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateWorkspaceReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.wsSvc.Create(c.UserContext(), userID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusCreated, resp)
}

// Update godoc
// @Summary      Update a workspace
// @Description  Updates workspace details (must be owner or editor)
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Workspace ID"
// @Param        request body dto.UpdateWorkspaceReq true "Update Details"
// @Success      200  {object}  model.Workspace
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /workspaces/{id} [put]
func (h *WorkspaceHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	var req dto.UpdateWorkspaceReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.wsSvc.Update(c.UserContext(), userID, wsID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete a workspace
// @Description  Deletes a workspace and all its contents (must be owner)
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Workspace ID"
// @Success      204  "No Content"
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /workspaces/{id} [delete]
func (h *WorkspaceHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	if appErr := h.wsSvc.Delete(c.UserContext(), userID, wsID); appErr != nil {
		return handleError(c, appErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListMembers godoc
// @Summary      List workspace members
// @Description  Lists all members of a workspace (must be a member)
// @Tags         workspaces
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Workspace ID"
// @Success      200  {object}  dto.MemberListResp
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Router       /workspaces/{id}/members [get]
func (h *WorkspaceHandler) ListMembers(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	resp, appErr := h.wsSvc.ListMembers(c.UserContext(), userID, wsID)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// AddMember godoc
// @Summary      Add a workspace member
// @Description  Invites a user by username or email (must be owner)
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Workspace ID"
// @Param        request body dto.AddMemberReq true "Member Details"
// @Success      201  {object}  dto.MemberResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      409  {object}  pkg.AppError
// @Router       /workspaces/{id}/members [post]
func (h *WorkspaceHandler) AddMember(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	var req dto.AddMemberReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.wsSvc.AddMember(c.UserContext(), userID, wsID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusCreated, resp)
}

// UpdateMemberRole godoc
// @Summary      Update a member's role
// @Description  Changes a member's role (must be owner)
// @Tags         workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true  "Workspace ID"
// @Param        userId  path      string  true  "Target User ID"
// @Param        request body dto.UpdateMemberRoleReq true "Role"
// @Success      204  "No Content"
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Router       /workspaces/{id}/members/{userId} [put]
func (h *WorkspaceHandler) UpdateMemberRole(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid user ID"))
	}

	var req dto.UpdateMemberRoleReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	if appErr := h.wsSvc.UpdateMemberRole(c.UserContext(), userID, wsID, targetID, req); appErr != nil {
		return handleError(c, appErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveMember godoc
// @Summary      Remove a workspace member
// @Description  Removes a member (owner removes anyone; members can leave)
// @Tags         workspaces
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true  "Workspace ID"
// @Param        userId  path      string  true  "Target User ID"
// @Success      204  "No Content"
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Router       /workspaces/{id}/members/{userId} [delete]
func (h *WorkspaceHandler) RemoveMember(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	targetID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid user ID"))
	}

	if appErr := h.wsSvc.RemoveMember(c.UserContext(), userID, wsID, targetID); appErr != nil {
		return handleError(c, appErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
