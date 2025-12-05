package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"api-core/internal/config"
	"api-core/internal/datastore/userstore"
	appauth "api-core/pkg/auth"
	"api-core/pkg/jwtx"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

type Service struct {
	stateTTL    time.Duration
	statePrefix string

	userStore   userstore.Store
	googleOAuth *appauth.GoogleOAuth
	tokenIssuer *jwtx.HMACIssuer
	redis       *redis.Client
	cfg         config.GoogleConfig
}

type AuthResponse struct {
	Token string          `json:"token"`
	User  *userstore.User `json:"user"`
}

func NewService(
	userStore userstore.Store,
	googleOAuth *appauth.GoogleOAuth,
	tokenIssuer *jwtx.HMACIssuer,
	redis *redis.Client,
	googleCfg config.GoogleConfig,
) *Service {
	return &Service{
		stateTTL:    5 * time.Minute,
		statePrefix: "oauth_state:",
		userStore:   userStore,
		googleOAuth: googleOAuth,
		tokenIssuer: tokenIssuer,
		redis:       redis,
		cfg:         googleCfg,
	}
}

func (s *Service) stateKey(state string) string {
	return s.statePrefix + state
}

func (s *Service) GenerateLoginURL(ctx context.Context) (string, error) {
	if s.googleOAuth == nil {
		return "", errors.New("google oauth not configured")
	}

	state := uuid.NewString()
	if err := s.redis.Set(ctx, s.stateKey(state), "1", s.stateTTL).Err(); err != nil {
		return "", fmt.Errorf("store oauth state: %w", err)
	}
	url := s.googleOAuth.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "code"))
	log.Printf("auth: generated google login url: %s", url)
	return url, nil
}

func (s *Service) HandleCallback(ctx context.Context, state, code string) (*AuthResponse, error) {
	if state == "" || code == "" {
		return nil, errors.New("missing state or code")
	}

	ok, err := s.consumeState(ctx, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("invalid state")
	}

	token, err := s.googleOAuth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}

	profile, err := s.googleOAuth.FetchUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}

	loginAt := time.Now().UTC()
	user, err := s.userStore.UpsertGoogleUser(ctx, userstore.UpsertGoogleUserParams{
		GoogleID:      profile.ID,
		Email:         profile.Email,
		Name:          &profile.Name,
		Picture:       &profile.Picture,
		Locale:        &profile.Locale,
		VerifiedEmail: profile.VerifiedEmail,
		LoginAt:       loginAt,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	tokenStr, err := s.tokenIssuer.Issue(fmt.Sprintf("%d", user.ID), user.Email)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &AuthResponse{
		Token: tokenStr,
		User:  user,
	}, nil
}

func (s *Service) consumeState(ctx context.Context, state string) (bool, error) {
	key := s.stateKey(state)
	val, err := s.redis.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consume state: %w", err)
	}
	return val != "", nil
}
