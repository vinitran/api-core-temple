package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"api-core/pkg/auth"
	"api-core/pkg/errorx"
	"api-core/pkg/jwtx"
)

const (
	captchaVerifyURL   = "https://www.google.com/recaptcha/api/siteverify"
	captchaTimeout     = 10 * time.Second
	captchaMaxBodySize = 1 << 20
)

// TokenVerifier verifies a bearer token and returns its claims.
// jwtx.HMAC and auth.CognitoVerifier satisfy it.
type TokenVerifier interface {
	Verify(token string) (*jwtx.Claims, error)
}

// Authn rejects requests without a valid bearer token and stores the verified
// claims in the request context (see auth.ResolveValidSubject).
func Authn(verifier TokenVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if !ok {
				return Fail(c, errorx.New(errorx.Authn, "missing or malformed bearer token"))
			}

			claims, err := verifier.Verify(token)
			if err != nil {
				return Fail(c, errorx.WrapMsg(err, errorx.Authn, "invalid access token"))
			}

			ctx := auth.WithAuthJWT(c.Request().Context(), token)
			ctx = auth.WithAuthClaims(ctx, claims)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(token)
	return token, token != ""
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

// CaptchaValid verifies a reCAPTCHA v3 token sent as {"captcha": "..."} in the
// JSON body. The body is restored so the handler can still bind it.
func CaptchaValid(secret string, minScore float64, action string) echo.MiddlewareFunc {
	client := &http.Client{Timeout: captchaTimeout}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if secret == "" {
				return Fail(c, errorx.New(errorx.Service, "missing captcha secret"))
			}

			req := c.Request()
			raw, err := io.ReadAll(io.LimitReader(req.Body, captchaMaxBodySize))
			if err != nil {
				return Fail(c, errorx.WrapMsg(err, errorx.Invalid, "unable to read request body"))
			}
			req.Body = io.NopCloser(bytes.NewReader(raw))

			var payload CaptchaPayload
			if err := json.Unmarshal(raw, &payload); err != nil || payload.Captcha == "" {
				return Fail(c, errorx.New(errorx.Captcha, "missing captcha"))
			}

			body, err := verifyCaptcha(c, client, secret, payload.Captcha)
			if err != nil {
				return Fail(c, errorx.WrapMsg(err, errorx.Service, "captcha verification failed"))
			}

			if !body.Success || body.Score < minScore || (action != "" && body.Action != action) {
				return Fail(c, errorx.New(errorx.Captcha, "captcha failed"))
			}

			return next(c)
		}
	}
}

func verifyCaptcha(c echo.Context, client *http.Client, secret, token string) (*CaptchaVerifyResponse, error) {
	form := url.Values{"secret": {secret}, "response": {token}}
	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, captchaVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("captcha verify: unexpected status %d", resp.StatusCode)
	}

	var body CaptchaVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, errors.Join(errors.New("captcha verify: decode response"), err)
	}

	return &body, nil
}
