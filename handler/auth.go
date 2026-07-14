package handler

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/dto"
	"github.com/RenzIP/Graphic-Diagram-Online/middleware"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	cfg     *config.Config
}

func NewAuthHandler(authSvc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, cfg: cfg}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account and returns an auth token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterReq true "Registration Details"
// @Success      201  {object}  dto.AuthCallbackResp
// @Failure      400  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	user, appErr := h.authSvc.Register(c.UserContext(), req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	resp, err := h.buildAuthSession(user)
	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to sign auth token").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusCreated, resp)
}

// Login godoc
// @Summary      Login user
// @Description  Authenticates a user and returns an auth token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginReq true "Login Credentials"
// @Success      200  {object}  dto.AuthCallbackResp
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	user, appErr := h.authSvc.Login(c.UserContext(), req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	resp, err := h.buildAuthSession(user)
	if err != nil {
		return handleError(c, pkg.ErrInternal.WithMessage("failed to sign auth token").WithDetails(err.Error()))
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, resp)
}

// ChangePassword godoc
// @Summary      Change Password
// @Description  Allows an authenticated user to change their password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.ChangePasswordReq true "Password Data"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  pkg.AppError
// @Failure      401  {object}  pkg.AppError
// @Failure      500  {object}  pkg.AppError
// @Router       /change-password [post]
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.ChangePasswordReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	if appErr := h.authSvc.ChangePassword(c.UserContext(), userID, req); appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, fiber.Map{
		"message": "password changed successfully",
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	profile, appErr := h.authSvc.GetProfile(c.UserContext(), userID)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, profile)
}

func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req dto.UpdateProfileReq
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, pkg.ErrBadRequest.WithMessage("invalid request body"))
	}

	profile, appErr := h.authSvc.UpdateProfile(c.UserContext(), userID, req)
	if appErr != nil {
		return handleError(c, appErr)
	}

	return pkg.WriteSuccess(c, fiber.StatusOK, profile)
}

func (h *AuthHandler) GoogleLogin(c *fiber.Ctx) error {
	redirectURI := h.oauthRedirectURI("google")
	
	// Generate CSRF state parameter
	state := uuid.New().String()
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   600, // 10 minutes
	})
	
	url := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid%%20email%%20profile&access_type=offline&prompt=consent&state=%s",
		h.cfg.GoogleClientID,
		redirectURI,
		state,
	)
	return c.Redirect(url, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(c *fiber.Ctx) error {
	log.Println("[Auth] GoogleCallback handler entered")
	
	// Verify CSRF state parameter
	state := c.Query("state")
	cookieState := c.Cookies("oauth_state")
	if state == "" || cookieState == "" || state != cookieState {
		log.Printf("[Auth] Google callback CSRF validation failed: state=%s, cookie=%s", state, cookieState)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=csrf_validation_failed", fiber.StatusTemporaryRedirect)
	}
	// Clear the state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})
	
	code := c.Query("code")
	if code == "" {
		log.Println("[Auth] Google callback error: missing authorization code")
		return c.Redirect(h.cfg.FrontendURL+"/login?error=missing_code", fiber.StatusTemporaryRedirect)
	}

	redirectURI := h.oauthRedirectURI("google")
	log.Printf("[Auth] Exchanging Google code. redirect_uri=%s", redirectURI)
	tokenResp, err := exchangeGoogleCode(code, h.cfg.GoogleClientID, h.cfg.GoogleClientSecret, redirectURI)
	if err != nil {
		log.Printf("[Auth] Google code exchange failed: %v", err)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=exchange_failed", fiber.StatusTemporaryRedirect)
	}

	log.Println("[Auth] Fetching Google user info")
	userInfo, err := fetchGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		log.Printf("[Auth] Google user info failed: %v", err)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=userinfo_failed", fiber.StatusTemporaryRedirect)
	}

	log.Printf("[Auth] Completing Google OAuth for sub=%s, email=%s", userInfo.Sub, userInfo.Email)
	return h.completeOAuth(c, userInfo.Sub, userInfo.Email, userInfo.Name, userInfo.Picture)
}

func (h *AuthHandler) GitHubLogin(c *fiber.Ctx) error {
	redirectURI := h.oauthRedirectURI("github")
	
	// Generate CSRF state parameter
	state := uuid.New().String()
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   600, // 10 minutes
	})
	
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s",
		h.cfg.GitHubClientID,
		redirectURI,
		state,
	)
	return c.Redirect(url, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) GitHubCallback(c *fiber.Ctx) error {
	// Verify CSRF state parameter
	state := c.Query("state")
	cookieState := c.Cookies("oauth_state")
	if state == "" || cookieState == "" || state != cookieState {
		log.Printf("[Auth] GitHub callback CSRF validation failed: state=%s, cookie=%s", state, cookieState)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=csrf_validation_failed", fiber.StatusTemporaryRedirect)
	}
	// Clear the state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})
	
	code := c.Query("code")
	if code == "" {
		return c.Redirect(h.cfg.FrontendURL+"/login?error=missing_code", fiber.StatusTemporaryRedirect)
	}

	redirectURI := h.oauthRedirectURI("github")
	accessToken, err := exchangeGitHubCode(code, h.cfg.GitHubClientID, h.cfg.GitHubClientSecret, redirectURI)
	if err != nil {
		log.Printf("[Auth] GitHub code exchange failed: %v", err)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=exchange_failed", fiber.StatusTemporaryRedirect)
	}

	userInfo, err := fetchGitHubUserInfo(accessToken)
	if err != nil {
		log.Printf("[Auth] GitHub user info failed: %v", err)
		return c.Redirect(h.cfg.FrontendURL+"/login?error=userinfo_failed", fiber.StatusTemporaryRedirect)
	}

	userUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("github:"+fmt.Sprint(userInfo.ID)))
	return h.completeOAuth(c, userUUID.String(), userInfo.Email, userInfo.Name, userInfo.AvatarURL)
}

func (h *AuthHandler) completeOAuth(c *fiber.Ctx, providerUserID, email, fullName, avatarURL string) error {
	userID, err := uuid.Parse(providerUserID)
	if err != nil {
		userID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(providerUserID))
	}

	// Upsert user and get the profile with role from DB
	user, appErr := h.authSvc.UpsertProfile(c.UserContext(), userID, email, strPtr(fullName), strPtr(avatarURL))
	if appErr != nil {
		return c.Redirect(h.cfg.FrontendURL+"/login?error=profile_failed", fiber.StatusTemporaryRedirect)
	}

	// Use role from database, not hardcoded "user"
	username := usernameForToken(email, fullName, userID)
	token, err := h.signJWT(userID, username, user.Role)
	if err != nil {
		return c.Redirect(h.cfg.FrontendURL+"/login?error=token_failed", fiber.StatusTemporaryRedirect)
	}

	callbackURL := fmt.Sprintf("%s/auth/callback#token=%s", h.cfg.FrontendURL, token)
	return c.Redirect(callbackURL, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) buildAuthSession(user *model.UserProfile) (*dto.AuthCallbackResp, error) {
	var username string
	if user.Username != nil {
		username = *user.Username
	}
	token, err := h.signJWT(user.ID, username, user.Role)
	if err != nil {
		return nil, err
	}

	return &dto.AuthCallbackResp{
		Token: token,
		User:  h.authSvc.UserResponse(user),
	}, nil
}

func (h *AuthHandler) signJWT(userID uuid.UUID, username, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID.String(),
		"username": username,
		"role":     role,
		"iat":      now.Unix(),
		"exp":      now.Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

func (h *AuthHandler) oauthRedirectURI(provider string) string {
	return fmt.Sprintf("%s/api/auth/%s/callback", h.cfg.BackendURL, provider)
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func usernameForToken(email, fullName string, userID uuid.UUID) string {
	if email != "" {
		return email
	}
	if fullName != "" {
		return fullName
	}
	return userID.String()
}
