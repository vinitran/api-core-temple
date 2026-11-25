// Package handler builds the HTTP router.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	httpx "api-core/pkg/httpx_echo"
)

const bodyLimit = "2M"

// Module is a feature that mounts its own routes. Adding a feature means
// adding a Module in the container; the router itself does not change.
type Module interface {
	RegisterRoutes(api *echo.Group, requireAuth echo.MiddlewareFunc)
}

type Config struct {
	AllowedOrigins []string
	Verifier       httpx.TokenVerifier
	Modules        []Module
}

func New(cfg Config) (http.Handler, error) {
	if cfg.Verifier == nil {
		return nil, errors.New("handler: token verifier is required")
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.JSONSerializer = httpx.SegmentJSONSerializer{}
	e.HTTPErrorHandler = httpx.ErrorHandler
	e.IPExtractor = echo.ExtractIPFromXFFHeader()

	e.Use(middleware.RequestID())
	e.Use(requestLogger())
	e.Use(middleware.Recover())
	e.Use(corsMiddleware(cfg.AllowedOrigins))
	e.Use(middleware.BodyLimit(bodyLimit))

	api := e.Group("/api/v1")
	api.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	requireAuth := httpx.Authn(cfg.Verifier)
	for _, m := range cfg.Modules {
		m.RegisterRoutes(api, requireAuth)
	}

	return e, nil
}

// corsMiddleware allows the configured origins. Credentials are only allowed
// for an explicit origin list: echo reflects the request origin for "*", so
// "*" + credentials would let any site make authenticated requests.
func corsMiddleware(origins []string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: !slices.Contains(origins, "*"),
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
			http.MethodDelete, http.MethodHead, http.MethodOptions,
		},
		MaxAge: 60 * 60,
	})
}

func requestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURIPath:   true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogRemoteIP:  true,
		LogError:     true,
		HandleError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method,
				"path", v.URIPath,
				"status", v.Status,
				"latency", v.Latency,
				"request_id", v.RequestID,
				"remote_ip", v.RemoteIP,
			}
			if v.Error != nil {
				attrs = append(attrs, "error", v.Error)
			}
			slog.InfoContext(c.Request().Context(), "http request", attrs...)
			return nil
		},
	})
}
