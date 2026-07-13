package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/middleware"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
)

// ProjectHandler handles project CRUD endpoints.
type ProjectHandler struct {
	projSvc *service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projSvc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projSvc: projSvc}
}

// ListByWorkspace godoc
// @Summary      Get list of projects in a workspace
// @Description  Retrieves all projects for a given workspace ID
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true   "Workspace ID"
// @Param        page    query     int     false  "Page number"
// @Param        per_page query    int     false  "Items per page"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /workspaces/{id}/projects [get]
func (h *ProjectHandler) ListByWorkspace(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	wsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	pq := dto.ParsePagination(c.Query("page"), c.Query("per_page"))

	resp, appErr := h.projSvc.ListByWorkspace(c.UserContext(), userID, wsID, pq)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WritePaginated(c, resp.Data, resp.Meta.Page, resp.Meta.PerPage, resp.Meta.Total)
}

// Create godoc
// @Summary      Create a new project
// @Description  Creates a new project in a workspace
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateProjectReq true "Project Details"
// @Success      201  {object}  model.Project
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /projects [post]
func (h *ProjectHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateProjectReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.projSvc.Create(c.UserContext(), userID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusCreated, resp)
}

// Update godoc
// @Summary      Update a project
// @Description  Updates a project's details
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true   "Project ID"
// @Param        request body dto.UpdateProjectReq true "Update Details"
// @Success      200  {object}  model.Project
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /projects/{id} [put]
func (h *ProjectHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	projID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid project ID"))
	}

	var req dto.UpdateProjectReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.projSvc.Update(c.UserContext(), userID, projID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete a project
// @Description  Deletes a project and its documents
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true   "Project ID"
// @Success      204  "No Content"
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /projects/{id} [delete]
func (h *ProjectHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	projID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid project ID"))
	}

	if appErr := h.projSvc.Delete(c.UserContext(), userID, projID); appErr != nil {
		return handleError(c, appErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
