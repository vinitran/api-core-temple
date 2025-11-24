package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GoogleLogin(c echo.Context) error {
	url, err := h.service.GenerateLoginURL(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"url": url,
	})
}

func (h *Handler) GoogleCallback(c echo.Context) error {
	state := c.QueryParam("state")
	code := c.QueryParam("code")
	resp, err := h.service.HandleCallback(c.Request().Context(), state, code)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, resp)
}
