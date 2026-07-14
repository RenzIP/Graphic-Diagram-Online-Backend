package model

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Name            *string    `json:"name,omitempty" gorm:"column:name"`
	Username        *string    `json:"username,omitempty" gorm:"unique"`
	Email           *string    `json:"email,omitempty" gorm:"unique"`
	Password        *string    `json:"-" gorm:"column:password"`
	Provider        string     `json:"provider" gorm:"default:local"`
	ProviderID      *string    `json:"provider_id,omitempty"`
	Avatar          *string    `json:"avatar,omitempty" gorm:"column:avatar"`
	Role            string     `json:"role" gorm:"not null;default:user"`
	Status          string     `json:"status" gorm:"default:active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	RememberToken   *string    `json:"-"`
	LastLogin       *time.Time `json:"last_login,omitempty"`
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;default:now()"`

	// Keep existing fields for backward compatibility if needed in some parts
	FullName  *string `json:"full_name,omitempty" gorm:"column:full_name"`
	AvatarURL *string `json:"avatar_url,omitempty" gorm:"column:avatar_url"`
}
