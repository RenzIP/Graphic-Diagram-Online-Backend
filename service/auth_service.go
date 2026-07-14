package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/repository"
)

type AuthService struct {
	userRepo *repository.UserRepo
}

func NewAuthService(userRepo *repository.UserRepo) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.AuthMeResp, *pkg.AppError) {
	user, appErr := s.userRepo.FindByID(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}
	if user == nil {
		return nil, pkg.ErrNotFound.WithMessage("user not found")
	}

	return userResponse(user), nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileReq) (*dto.AuthMeResp, *pkg.AppError) {
	if appErr := pkg.Validate(req); appErr != nil {
		return nil, appErr
	}

	user, appErr := s.userRepo.FindByID(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}
	if user == nil {
		return nil, pkg.ErrNotFound.WithMessage("user not found")
	}

	if req.Username != nil {
		newUsername := strings.ToLower(strings.TrimSpace(*req.Username))
		if user.Username == nil || newUsername != *user.Username {
			// check for conflict
			existing, appErr := s.userRepo.FindByUsername(ctx, newUsername)
			if appErr != nil {
				return nil, appErr
			}
			if existing != nil {
				return nil, pkg.ErrConflict.WithMessage("username already taken")
			}
			user.Username = &newUsername
		}
	}
	if req.FullName != nil {
		user.FullName = req.FullName
	}

	if _, appErr := s.userRepo.Upsert(ctx, user); appErr != nil {
		return nil, appErr
	}

	return userResponse(user), nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterReq) (*model.UserProfile, *pkg.AppError) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	if appErr := pkg.Validate(req); appErr != nil {
		return nil, appErr
	}

	existingUser, appErr := s.userRepo.FindByUsernameOrEmail(ctx, req.Username)
	if appErr != nil {
		return nil, appErr
	}
	if existingUser != nil {
		return nil, pkg.ErrConflict.WithMessage("username or email already registered")
	}
	
	existingEmail, appErr := s.userRepo.FindByEmail(ctx, req.Email)
	if appErr != nil {
		return nil, appErr
	}
	if existingEmail != nil {
		return nil, pkg.ErrConflict.WithMessage("username or email already registered")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, pkg.ErrInternal.WithMessage("failed to hash password")
	}

	hash := string(passwordHash)
	user := &model.UserProfile{
		ID:        uuid.New(),
		Name:      &req.Name,
		Username:  &req.Username,
		Email:     &req.Email,
		Password:  &hash,
		Provider:  "local",
		Role:      "user", // Always force to "user" - never accept from client
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if appErr := s.userRepo.Create(ctx, user); appErr != nil {
		return nil, appErr
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.UserProfile, *pkg.AppError) {
	req.Identifier = strings.ToLower(strings.TrimSpace(req.Identifier))
	if appErr := pkg.Validate(req); appErr != nil {
		return nil, appErr
	}

	user, appErr := s.userRepo.FindByUsernameOrEmail(ctx, req.Identifier)
	if appErr != nil {
		return nil, appErr
	}
	if user == nil {
		return nil, pkg.ErrNotFound.WithMessage("account_not_found")
	}
	if user.Password == nil || *user.Password == "" {
		return nil, pkg.ErrUnauthorized.WithMessage("invalid username/email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.Password)); err != nil {
		return nil, pkg.ErrUnauthorized.WithMessage("invalid username/email or password")
	}

	return user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordReq) *pkg.AppError {
	if appErr := pkg.Validate(req); appErr != nil {
		return appErr
	}

	user, appErr := s.userRepo.FindByID(ctx, userID)
	if appErr != nil {
		return appErr
	}
	if user == nil {
		return pkg.ErrNotFound.WithMessage("user not found")
	}
	if user.Password == nil || *user.Password == "" {
		return pkg.ErrBadRequest.WithMessage("password login is not available for this account")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.CurrentPassword)); err != nil {
		return pkg.ErrUnauthorized.WithMessage("current password is invalid")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return pkg.ErrInternal.WithMessage("failed to hash password")
	}

	return s.userRepo.UpdatePassword(ctx, userID, string(passwordHash))
}

func (s *AuthService) UserResponse(user *model.UserProfile) dto.AuthUserResp {
	return *userResponse(user)
}

func (s *AuthService) UpsertProfile(ctx context.Context, userID uuid.UUID, email string, fullName, avatarURL *string) (*model.UserProfile, *pkg.AppError) {
	email = strings.ToLower(strings.TrimSpace(email))

	var existing *model.UserProfile
	var err *pkg.AppError

	if email != "" {
		existing, err = s.userRepo.FindByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
	}

	if existing == nil {
		existing, err = s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	var userToUpsert *model.UserProfile
	if existing == nil {
		// New user: use oauth generated username and full name
		username := usernameFromOAuth(email, fullName, userID)
		userToUpsert = &model.UserProfile{
			ID:        userID,
			Name:      fullName,
			Username:  &username,
			Email:     optionalString(email),
			FullName:  fullName,
			Avatar:    avatarURL,
			AvatarURL: avatarURL,
			Provider:  "oauth",
			Role:      "user",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	} else {
		// Existing user: preserve their edited username and full name
		userToUpsert = existing
		
		// ALWAYS sync email if it was missing
		if email != "" && (userToUpsert.Email == nil || *userToUpsert.Email == "") {
			userToUpsert.Email = &email
		}
		// ALWAYS sync FullName/Name if it was missing
		if fullName != nil && *fullName != "" {
			if userToUpsert.FullName == nil || *userToUpsert.FullName == "" {
				userToUpsert.FullName = fullName
			}
			if userToUpsert.Name == nil || *userToUpsert.Name == "" {
				userToUpsert.Name = fullName
			}
		}

		if avatarURL != nil && (userToUpsert.AvatarURL == nil || *userToUpsert.AvatarURL == "") {
			userToUpsert.AvatarURL = avatarURL
			userToUpsert.Avatar = avatarURL
		}
		if userToUpsert.Provider == "local" {
			userToUpsert.Provider = "oauth_linked"
		}
	}

	return s.userRepo.Upsert(ctx, userToUpsert)
}

func userResponse(user *model.UserProfile) *dto.AuthMeResp {
	return &dto.AuthMeResp{
		ID:        user.ID.String(),
		Name:      user.Name,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Provider:  user.Provider,
		Avatar:    user.Avatar,
		Status:    user.Status,
		FullName:  user.FullName,
		AvatarURL: user.AvatarURL,
	}
}

func usernameFromOAuth(email string, fullName *string, userID uuid.UUID) string {
	suffix := strings.ReplaceAll(userID.String(), "-", "")
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	if email != "" {
		base := strings.Split(email, "@")[0]
		base = strings.ToLower(strings.TrimSpace(base))
		if base != "" {
			return base + "_" + suffix
		}
	}
	if fullName != nil {
		base := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(*fullName), " ", "_"))
		if base != "" {
			return base + "_" + suffix
		}
	}

	return "user_" + strings.ReplaceAll(userID.String(), "-", "")
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
