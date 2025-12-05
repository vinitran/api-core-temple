package httpx

import (
	"api-core/pkg/auth"
	"api-core/pkg/errorx"
	"api-core/pkg/jwtx"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type Guard interface {
	AuthenticateJWT(tokenStr string) (*jwt.Token, error)
}

func Authn(guard Guard) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return Abort(c, errorx.Wrap(errors.New("missing access token"), errorx.Authn), -1)
			}

			parts := strings.Split(header, "Bearer")
			if len(parts) != 2 {
				return Abort(c, errorx.Wrap(errors.New("invalid access token"), errorx.Authn), -1)
			}

			token := strings.TrimSpace(parts[1])
			if len(token) == 0 {
				return Abort(c, errorx.Wrap(errors.New("invalid access token"), errorx.Authn), -1)
			}

			claims, err := guard.AuthenticateJWT(token)
			if err != nil {
				// although it's a client error, we don't want to detailed information
				//nolint:errcheck
				return Abort(c, errorx.Wrap(err, errorx.Authn), -1)
			}

			ctx := c.Request().Context()
			ctx = auth.WithAuthJWT(ctx, claims.Raw)

			jwtClaims, err := jwtx.NewJWTClaims(claims.Claims)
			if err != nil {
				return Abort(c, errorx.Wrap(fmt.Errorf("invalid access token: %v", err), errorx.Authn), -1)
			}

			ctx = auth.WithAuthClaims(ctx, jwtClaims)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

type CaptchaPayload struct {
	Captcha string `json:"captcha"`
}

type CaptchaVerifyResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func CaptchaValid(secret string, score float64, action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if secret == "" {
				return Abort(c, errorx.Wrap(errors.New("missing captcha secret"), errorx.Service))
			}

			var payload CaptchaPayload
			if err := c.Bind(&payload); err != nil {
				return Abort(c, errorx.Wrap(err, errorx.Invalid))
			}

			resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
				"secret":   {secret},
				"response": {payload.Captcha},
			})
			if err != nil {
				return Abort(c, errorx.Wrap(err, errorx.Service))
			}
			defer resp.Body.Close()

			var body CaptchaVerifyResponse
			if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
				return Abort(c, errorx.Wrap(err, errorx.Service))
			}

			if action != "" && body.Action != action {
				return Abort(c, errorx.Wrap(errors.New("captcha failed"), errorx.Authz))
			}

			if !body.Success || body.Score < score {
				return Abort(c, errorx.Wrap(errors.New("captcha failed"), errorx.Authz))
			}

			return next(c)
		}
	}
}
