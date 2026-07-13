package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/middleware"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
)

// DocumentHandler handles document CRUD and recent endpoints.
type DocumentHandler struct {
	docSvc *service.DocumentService
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(docSvc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{docSvc: docSvc}
}

// ListByProject godoc
// @Summary      Get all documents by project
// @Description  Retrieves a paginated list of documents inside a specific project
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Project ID"
// @Param        page query int false "Page number"
// @Param        per_page query int false "Items per page"
// @Param        diagram_type query string false "Filter by diagram type"
// @Success      200  {object}  dto.DocumentListResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /projects/{id}/documents [get]
func (h *DocumentHandler) ListByProject(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	projID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid project ID"))
	}

	pq := dto.ParsePagination(c.Query("page"), c.Query("per_page"))
	diagramType := c.Query("diagram_type")
	sortBy := c.Query("sort_by", "updated_at")
	sortOrder := c.Query("sort_order", "desc")

	resp, appErr := h.docSvc.ListByProject(c.UserContext(), userID, projID, pq, diagramType, sortBy, sortOrder)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WritePaginated(c, resp.Data, resp.Meta.Page, resp.Meta.PerPage, resp.Meta.Total)
}

// GetByID godoc
// @Summary      Get document detail
// @Description  Retrieves the full content and metadata of a specific document by its ID
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Success      200  {object}  dto.DocumentResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/{id} [get]
func (h *DocumentHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	resp, appErr := h.docSvc.GetByID(c.UserContext(), userID, docID)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// Create handles POST /api/documents — create a new document.
func (h *DocumentHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.CreateDocumentReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.docSvc.Create(c.UserContext(), userID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusCreated, resp)
}

// Update handles PUT /api/documents/:id — update document.
func (h *DocumentHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	var req dto.UpdateDocumentReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	resp, appErr := h.docSvc.Update(c.UserContext(), userID, docID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// Delete handles DELETE /api/documents/:id — delete document.
func (h *DocumentHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	if appErr := h.docSvc.Delete(c.UserContext(), userID, docID); appErr != nil {
		return handleError(c, appErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Recent handles GET /api/documents/recent — recently updated documents dashboard widget.
func (h *DocumentHandler) Recent(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	limit := 10
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}

	resp, appErr := h.docSvc.ListRecent(c.UserContext(), userID, limit)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// ListVersions handles GET /api/documents/:id/versions.
func (h *DocumentHandler) ListVersions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	resp, appErr := h.docSvc.ListVersions(c.UserContext(), userID, docID)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// RestoreVersion handles POST /api/documents/:id/versions/:version/restore.
func (h *DocumentHandler) RestoreVersion(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	version, err := strconv.Atoi(c.Params("version"))
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid version number"))
	}

	resp, appErr := h.docSvc.RestoreVersion(c.UserContext(), userID, docID, version)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}
