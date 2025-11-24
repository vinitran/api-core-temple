package handler

import (
	"errors"
	"net/http"
	"otp-core/pkg/auth"
	"otp-core/pkg/errorx"
	httpx "otp-core/pkg/httpx_echo"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/do"
)

type Config struct {
	Container *do.Injector

	Origins []string
}

func New(cfg *Config) (http.Handler, error) {
	r := echo.New()

	r.IPExtractor = echo.ExtractIPFromXFFHeader()
	r.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339}\t${method}\t${uri}\t${status}\t${latency_human}\n",
	}))
	r.Use(middleware.Recover())

	cors := middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.Origins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions},
		MaxAge:           60 * 60,
	})

	//guard, err := do.Invoke[*auth.Guard](cfg.Container)
	//if err != nil {
	//	return nil, err
	//}

	//authorized := httpx.Authn(guard)

	routesAPIv1 := r.Group("/api/v1")
	{
		routesAPIv1.Use(cors)
		routesAPIv1.GET("/ping", func(c echo.Context) error {
			return c.String(http.StatusOK, "hello world")
		})
	}

	//groupUser := &GroupUser{cfg}
	//{
	//	routesAPIv1.GET("/me", authorized(groupUser.ShowMe))
	//}

	return r, nil
}

func restAbort(c echo.Context, v any, err error) error {
	if errors.Is(err, auth.ErrInvalidSession) {
		return httpx.Abort(c, errorx.Wrap(err, errorx.Authn))
	}

	if _, ok := err.(*errorx.Error); ok {
		return httpx.Abort(c, err)
	}

	if err != nil {
		return httpx.Abort(c, errorx.Wrap(err, errorx.Service))
	}

	return httpx.Abort(c, v)
}

func queryParamInt(c echo.Context, name string, val int) int {
	v := c.QueryParam(name)
	if v == "" {
		return val
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return val
	}

	return i
}
