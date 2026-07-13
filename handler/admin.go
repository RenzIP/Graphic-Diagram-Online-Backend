package handler

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RenzIP/Graphic-Diagram-Online/model"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// GetOverview godoc
// @Summary      Get system overview metrics
// @Description  Returns system-wide counts of users, workspaces, projects, and documents
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]int64
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/overview [get]
func (h *AdminHandler) GetOverview(c *fiber.Ctx) error {
	var userCount int64
	var wsCount int64
	var projectCount int64
	var docCount int64

	if err := h.db.WithContext(c.UserContext()).Model(&model.UserProfile{}).Count(&userCount).Error; err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to count users").WithDetails(err.Error()))
	}
	if err := h.db.WithContext(c.UserContext()).Model(&model.Workspace{}).Count(&wsCount).Error; err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to count workspaces").WithDetails(err.Error()))
	}
	if err := h.db.WithContext(c.UserContext()).Model(&model.Project{}).Count(&projectCount).Error; err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to count projects").WithDetails(err.Error()))
	}
	if err := h.db.WithContext(c.UserContext()).Model(&model.Document{}).Count(&docCount).Error; err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to count documents").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{
		"users":      userCount,
		"workspaces": wsCount,
		"projects":   projectCount,
		"documents":  docCount,
	})
}

// ListUsers godoc
// @Summary      List all users
// @Description  Lists all user profiles registered in the system
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   model.UserProfile
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/users [get]
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	var users []model.UserProfile
	if err := h.db.WithContext(c.UserContext()).Order("created_at desc").Find(&users).Error; err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to list users").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, users)
}

// UpdateUserRole godoc
// @Summary      Update user role
// @Description  Updates a specific user's role to 'admin' or 'user'
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID"
// @Param        role body      map[string]string true "Target Role (e.g. {"role": "admin"})"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      404  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/users/{id}/role [put]
func (h *AdminHandler) UpdateUserRole(c *fiber.Ctx) error {
	idStr := c.Params("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid user ID"))
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role != "admin" && role != "user" {
		return handleError(c, pkg.ErrBadRequest.WithMessage("role must be 'admin' or 'user'"))
	}

	result := h.db.WithContext(c.UserContext()).Model(&model.UserProfile{}).Where("id = ?", userID).Update("role", role)
	if result.Error != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to update user role").WithDetails(result.Error.Error()))
	}
	if result.RowsAffected == 0 {
		return handleError(c, pkg.ErrNotFound.WithMessage("user not found"))
	}

	log.Printf("[Admin] User %s role updated to %s", userID, role)
	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{"message": "User role updated successfully"})
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Deletes a user and cleans up their workspace memberships from the platform
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid user ID"))
	}

	// Delete workspace memberships, profiles, etc.
	err = h.db.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
		// Delete user workspaces membership
		if err := tx.Exec("DELETE FROM workspace_members WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		// Set documents created_by to NULL
		if err := tx.Exec("UPDATE documents SET created_by = NULL WHERE created_by = ?", userID).Error; err != nil {
			return err
		}
		// Delete user profile
		if err := tx.Delete(&model.UserProfile{}, "id = ?", userID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to delete user").WithDetails(err.Error()))
	}

	log.Printf("[Admin] User %s deleted", userID)
	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{"message": "User deleted successfully"})
}

// ListWorkspaces godoc
// @Summary      List all workspaces
// @Description  Lists all workspaces in the system along with their owner details
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   map[string]interface{}
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/workspaces [get]
func (h *AdminHandler) ListWorkspaces(c *fiber.Ctx) error {
	type WorkspaceDetail struct {
		model.Workspace
		OwnerName  string `json:"owner_name"`
		OwnerEmail string `json:"owner_email"`
	}

	var results []WorkspaceDetail
	err := h.db.WithContext(c.UserContext()).
		Table("workspaces").
		Select("workspaces.*, user_profiles.username as owner_name, user_profiles.email as owner_email").
		Joins("left join user_profiles on user_profiles.id = workspaces.owner_id").
		Order("workspaces.created_at desc").
		Scan(&results).Error

	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to list workspaces").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, results)
}

// DeleteWorkspace godoc
// @Summary      Delete workspace
// @Description  Deletes a workspace along with all its child projects and documents
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Workspace ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/workspaces/{id} [delete]
func (h *AdminHandler) DeleteWorkspace(c *fiber.Ctx) error {
	idStr := c.Params("id")
	wsID, err := uuid.Parse(idStr)
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid workspace ID"))
	}

	err = h.db.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
		// Delete document versions associated with documents in the workspace
		if err := tx.Exec("DELETE FROM document_versions WHERE document_id IN (SELECT id FROM documents WHERE workspace_id = ?)", wsID).Error; err != nil {
			return err
		}
		// Delete documents
		if err := tx.Exec("DELETE FROM documents WHERE workspace_id = ?", wsID).Error; err != nil {
			return err
		}
		// Delete projects
		if err := tx.Exec("DELETE FROM projects WHERE workspace_id = ?", wsID).Error; err != nil {
			return err
		}
		// Delete members
		if err := tx.Exec("DELETE FROM workspace_members WHERE workspace_id = ?", wsID).Error; err != nil {
			return err
		}
		// Delete workspace
		if err := tx.Delete(&model.Workspace{}, "id = ?", wsID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to delete workspace").WithDetails(err.Error()))
	}

	log.Printf("[Admin] Workspace %s deleted", wsID)
	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{"message": "Workspace and all its assets deleted successfully"})
}

// ListDocuments godoc
// @Summary      List all diagrams
// @Description  Lists all diagrams in the system along with workspace and project names
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   map[string]interface{}
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/documents [get]
func (h *AdminHandler) ListDocuments(c *fiber.Ctx) error {
	type DocDetail struct {
		model.Document
		WorkspaceName string `json:"workspace_name"`
		ProjectName   string `json:"project_name"`
		CreatorName   string `json:"creator_name"`
	}

	var results []DocDetail
	err := h.db.WithContext(c.UserContext()).
		Table("documents").
		Select("documents.*, workspaces.name as workspace_name, projects.name as project_name, user_profiles.username as creator_name").
		Joins("left join workspaces on workspaces.id = documents.workspace_id").
		Joins("left join projects on projects.id = documents.project_id").
		Joins("left join user_profiles on user_profiles.id = documents.created_by").
		Order("documents.created_at desc").
		Scan(&results).Error

	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to list documents").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, results)
}

// DeleteDocument godoc
// @Summary      Delete diagram
// @Description  Deletes a diagram and all its historical versions
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      403  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /admin/documents/{id} [delete]
func (h *AdminHandler) DeleteDocument(c *fiber.Ctx) error {
	idStr := c.Params("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid document ID"))
	}

	err = h.db.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
		// Delete versions
		if err := tx.Exec("DELETE FROM document_versions WHERE document_id = ?", docID).Error; err != nil {
			return err
		}
		// Delete document
		if err := tx.Delete(&model.Document{}, "id = ?", docID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to delete document").WithDetails(err.Error()))
	}

	log.Printf("[Admin] Document %s deleted", docID)
	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{"message": "Document deleted successfully"})
}
