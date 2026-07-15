package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func makeToken(t *testing.T, secret, sub string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": sub,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"role": "user",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestAuth_AcceptsRawAndBearer(t *testing.T) {
	secret := "dev-secret-change-me"
	sub := "1ce5b0ab-44be-4100-b5e8-548658cab427"

	app := fiber.New()
	app.Use(Auth(secret))
	app.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"userId": GetUserID(c).String()})
	})

	cases := []struct {
		name  string
		auth  string
		want  int
	}{
		{"raw token (Swagger/curl style)", makeToken(t, secret, sub), 200},
		{"Bearer prefixed", "Bearer " + makeToken(t, secret, sub), 200},
		{"empty", "", 401},
		{"garbage", "not-a-token", 401},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("test: %v", err)
			}
			if resp.StatusCode != tc.want {
				t.Errorf("%s: status=%d want=%d (auth=%q)", tc.name, resp.StatusCode, tc.want, tc.auth)
			}
		})
	}
}
