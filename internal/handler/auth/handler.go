// Package auth exposes the authentication endpoints over HTTP.
package auth

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"api-core/internal/domain"
	authservice "api-core/internal/service/auth"
	appauth "api-core/pkg/auth"
	httpx "api-core/pkg/httpx_echo"
)

// Service is what the handler needs from the auth service.
type Service interface {
	GenerateLoginURL(ctx context.Context) (string, error)
	HandleCallback(ctx context.Context, state, code string) (*authservice.AuthResult, error)
	CurrentUser(ctx context.Context, subject string) (*domain.User, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts the auth endpoints under /auth.
func (h *Handler) RegisterRoutes(api *echo.Group, requireAuth echo.MiddlewareFunc) {
	g := api.Group("/auth")
	g.GET("/google/login", h.GoogleLogin)
	g.GET("/google/callback", h.GoogleCallback)
	g.GET("/me", h.Me, requireAuth)
}

type loginURLResponse struct {
	URL string `json:"url"`
}

type authResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
	User        *userResponse `json:"user"`
}

type userResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Name          *string    `json:"name"`
	Picture       *string    `json:"picture"`
	Locale        *string    `json:"locale"`
	VerifiedEmail bool       `json:"verified_email"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

func toUserResponse(u *domain.User) *userResponse {
	if u == nil {
		return nil
	}

	return &userResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Picture:       u.Picture,
		Locale:        u.Locale,
		VerifiedEmail: u.VerifiedEmail,
		CreatedAt:     u.CreatedAt,
		LastLoginAt:   u.LastLoginAt,
	}
}

// GoogleLogin returns the Google authorization URL the client should open.
func (h *Handler) GoogleLogin(c echo.Context) error {
	url, err := h.service.GenerateLoginURL(c.Request().Context())
	if err != nil {
		return httpx.Fail(c, err)
	}

	return httpx.Success(c, loginURLResponse{URL: url})
}

// GoogleCallback completes sign-in and returns an access token.
func (h *Handler) GoogleCallback(c echo.Context) error {
	res, err := h.service.HandleCallback(c.Request().Context(), c.QueryParam("state"), c.QueryParam("code"))
	if err != nil {
		return httpx.Fail(c, err)
	}

	return httpx.Success(c, authResponse{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   res.ExpiresAt,
		User:        toUserResponse(res.User),
	})
}

// Me returns the authenticated user.
func (h *Handler) Me(c echo.Context) error {
	ctx := c.Request().Context()

	subject, err := appauth.ResolveValidSubject(ctx)
	if err != nil {
		return httpx.Fail(c, err)
	}

	user, err := h.service.CurrentUser(ctx, subject)
	if err != nil {
		return httpx.Fail(c, err)
	}

	return httpx.Success(c, toUserResponse(user))
}
