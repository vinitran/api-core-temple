package auth

import (
	authservice "api-core/internal/service/auth"
	httpx "api-core/pkg/httpx_echo"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *authservice.Service
}

func NewHandler(service *authservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GoogleLogin(c echo.Context) error {
	url, err := h.service.GenerateLoginURL(c.Request().Context())
	return httpx.RestAbort(c, map[string]string{
		"url": url,
	}, err)
}

func (h *Handler) GoogleCallback(c echo.Context) error {
	state := c.QueryParam("state")
	code := c.QueryParam("code")
	resp, err := h.service.HandleCallback(c.Request().Context(), state, code)
	return httpx.RestAbort(c, resp, err)
}
