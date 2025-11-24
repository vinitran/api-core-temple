## Setup

Run api
```
go run ./cmd api --addr 0.0.0.0:3030
```

Run migration
```
go run ./cmd migrate --action up   # apply new migrations
go run ./cmd migrate --action down # rollback last batch
go run ./cmd migrate --action status
```

Generate bob (ORM stubs are written to `internal/service/bobmodel`, keeping manual code untouched)
```
go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c ./internal/config/bobgen.yaml
```

Clean and lint code
```
gofumpt -l -w .
golangci-lint run ./...
```

## Google OAuth 2.0 helper

The package `pkg/auth/google_oauth.go` wraps the OAuth2 flow for Google sign-in.

1. Configure via environment variables (recommended):
   ```
   GOOGLE_OAUTH_CLIENT_ID=<client id>
   GOOGLE_OAUTH_CLIENT_SECRET=<client secret>
   GOOGLE_OAUTH_REDIRECT_URL=https://your.app/oauth/google/callback
   GOOGLE_OAUTH_SCOPES=https://www.googleapis.com/auth/userinfo.email,https://www.googleapis.com/auth/userinfo.profile
   ```
2. Instantiate in code:
   ```go
   googleOAuth, err := auth.NewGoogleOAuth(auth.GoogleOAuthConfig{
       ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
       ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
       RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
   })
   ```
3. Build auth URL & exchange code:
   ```go
   url := googleOAuth.AuthCodeURL(state)
   token, err := googleOAuth.Exchange(ctx, code)
   user, err := googleOAuth.FetchUserInfo(ctx, token)
   ```

`GoogleOAuth` supplies helpers to refresh tokens and fetch user profile data, so handlers only need to manage session state.