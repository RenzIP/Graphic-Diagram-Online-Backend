package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RenzIP/Graphic-Diagram-Online/model"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.UserProfile, *pkg.AppError) {
	var user model.UserProfile
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkg.ErrInternal.WithMessage("failed to find user")
	}

	return &user, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*model.UserProfile, *pkg.AppError) {
	var user model.UserProfile
	if err := r.db.WithContext(ctx).
		First(&user, "LOWER(username) = ?", strings.ToLower(username)).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkg.ErrInternal.WithMessage("failed to find user")
	}

	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.UserProfile, *pkg.AppError) {
	var user model.UserProfile
	if err := r.db.WithContext(ctx).
		First(&user, "LOWER(email) = ?", strings.ToLower(email)).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkg.ErrInternal.WithMessage("failed to find user")
	}

	return &user, nil
}

func (r *UserRepo) FindByUsernameOrEmail(ctx context.Context, identifier string) (*model.UserProfile, *pkg.AppError) {
	normalized := strings.ToLower(strings.TrimSpace(identifier))

	var user model.UserProfile
	if err := r.db.WithContext(ctx).
		First(&user, "LOWER(username) = ? OR LOWER(email) = ?", normalized, normalized).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkg.ErrInternal.WithMessage("failed to find user")
	}

	return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.UserProfile) *pkg.AppError {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isDuplicateError(err) {
			return pkg.ErrConflict.WithMessage("username already registered")
		}
		return pkg.ErrInternal.WithMessage("failed to create user: " + err.Error())
	}

	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, password string) *pkg.AppError {
	result := r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("id = ?", id).
		Update("password", password)
	if result.Error != nil {
		return pkg.ErrInternal.WithMessage("failed to update password")
	}
	if result.RowsAffected == 0 {
		return pkg.ErrNotFound.WithMessage("user not found")
	}

	return nil
}

func (r *UserRepo) Upsert(ctx context.Context, user *model.UserProfile) (*model.UserProfile, *pkg.AppError) {
	// Check if user exists first
	existing, err := r.FindByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		// New user - create with default role
		user.Role = "user"
		if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
			if isDuplicateError(err) {
				return nil, pkg.ErrConflict.WithMessage("username already registered")
			}
			return nil, pkg.ErrInternal.WithMessage("failed to create user")
		}
		return user, nil
	}

	// Existing user - update profile but preserve role
	existing.Username = user.Username
	existing.Email = user.Email
	existing.FullName = user.FullName
	existing.Name = user.Name
	existing.AvatarURL = user.AvatarURL
	existing.Avatar = user.Avatar
	// Do NOT update Role - preserve existing role from DB

	if err := r.db.WithContext(ctx).Save(existing).Error; err != nil {
		return nil, pkg.ErrInternal.WithMessage("failed to update user profile")
	}

	return existing, nil
}

func isDuplicateError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
