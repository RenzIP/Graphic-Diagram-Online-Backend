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

// Create godoc
// @Summary      Create a new document
// @Description  Creates a new document in a workspace/project
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateDocumentReq true "Document Details"
// @Success      201  {object}  dto.DocumentResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents [post]
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

// Update godoc
// @Summary      Update a document
// @Description  Updates a document's content and metadata
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Param        request body dto.UpdateDocumentReq true "Document Details"
// @Success      200  {object}  dto.DocumentResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/{id} [put]
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

// Delete godoc
// @Summary      Delete a document
// @Description  Deletes a document by ID
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Success      204  "No Content"
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/{id} [delete]
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

// Recent godoc
// @Summary      Get recent documents
// @Description  Retrieves the user's recently accessed/updated documents
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of documents to retrieve"
// @Success      200  {array}   model.Document
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/recent [get]
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

// ListVersions godoc
// @Summary      Get document versions
// @Description  Retrieves the history of saved versions for a document
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Success      200  {array}   model.DocumentVersion
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/{id}/versions [get]
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

// RestoreVersion godoc
// @Summary      Restore document version
// @Description  Restores a document to a specific previous version
// @Tags         documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Param        version path int true "Version number"
// @Success      200  {object}  dto.DocumentResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /documents/{id}/versions/{version}/restore [post]
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
