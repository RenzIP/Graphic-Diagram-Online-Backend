package middleware

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RenzIP/Graphic-Diagram-Online/pkg"
)

// Auth returns a Fiber middleware that validates self-signed HS256 JWT tokens.
// On success, it sets ctx.Locals("userId") to the UUID from the `sub` claim
// and ctx.Locals("email") if present.
func Auth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract the Bearer token from the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("missing Authorization header"))
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("invalid Authorization header format"))
		}
		tokenStr := parts[1]

		// Parse and validate the JWT — HS256 only
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			log.Printf("[Auth] JWT validation failed: %v", err)
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("invalid or expired token"))
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("invalid token claims"))
		}

		// Extract user ID from the `sub` claim
		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("missing sub claim in token"))
		}

		userID, err := uuid.Parse(sub)
		if err != nil {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("invalid user ID in token"))
		}

		// Set user context for downstream handlers
		c.Locals("userId", userID)

		// Optionally extract email if present
		if email, ok := claims["email"].(string); ok {
			c.Locals("email", email)
		}

		// Extract role if present
		if role, ok := claims["role"].(string); ok {
			c.Locals("role", role)
		}

		return c.Next()
	}
}

// GetUserID extracts the authenticated user's UUID from ctx.Locals.
// Returns uuid.Nil if not set (should not happen behind Auth middleware).
func GetUserID(c *fiber.Ctx) uuid.UUID {
	if id, ok := c.Locals("userId").(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetRole extracts the authenticated user's role from ctx.Locals.
func GetRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return ""
}

// RequireRole returns a middleware that restricts access to specific roles.
// It assumes Auth middleware has already run and populated ctx.Locals("role").
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := GetRole(c)
		if userRole == "" {
			return pkg.WriteError(c, pkg.ErrUnauthorized.WithMessage("missing role information"))
		}

		for _, allowed := range allowedRoles {
			if strings.EqualFold(userRole, allowed) {
				return c.Next()
			}
		}

		return pkg.WriteError(c, pkg.ErrForbidden.WithMessage("you do not have permission to perform this action"))
	}
}
