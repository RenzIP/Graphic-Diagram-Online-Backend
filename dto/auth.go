package dto

type AuthCallbackReq struct {
	AccessToken  string `json:"access_token" validate:"required"`
	RefreshToken string `json:"refresh_token"`
}

type RegisterReq struct {
	Name     string `json:"name" validate:"required"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=72"`
}

type LoginReq struct {
	Identifier string `json:"identifier" validate:"required"` // email or username
	Password   string `json:"password" validate:"required"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6,max=72"`
}

type AuthCallbackResp struct {
	Token string       `json:"token"`
	User  AuthUserResp `json:"user"`
}

type AuthUserResp struct {
	ID        string  `json:"id"`
	Name      *string `json:"name"`
	Username  *string `json:"username"`
	Email     *string `json:"email"`
	Role      string  `json:"role"`
	Provider  string  `json:"provider"`
	Avatar    *string `json:"avatar"`
	Status    string  `json:"status"`

	// Keep existing ones for backward compatibility
	FullName  *string `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type AuthMeResp = AuthUserResp

type UpdateProfileReq struct {
	Name     *string `json:"name"`
	FullName *string `json:"full_name"`
	Username *string `json:"username" validate:"omitempty,min=3,max=50"`
}
