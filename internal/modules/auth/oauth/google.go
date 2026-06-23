package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Google struct {
	config *oauth2.Config
}

func NewGoogle(cfg ProviderConfig) *Google {
	return &Google{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes: []string{
				"openid",
				"profile",
				"email",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (g *Google) Name() string {
	return ProviderGoogle
}

func (g *Google) AuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *Google) Exchange(ctx context.Context, code string) (*Profile, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange google code: %w", err)
	}

	client := g.config.Client(ctx, token)

	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}

	var user googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode google userinfo: %w", err)
	}

	return &Profile{
		Provider:       ProviderGoogle,
		ProviderUserID: user.Sub,
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		Name:           user.Name,
		AvatarURL:      user.Picture,
	}, nil
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}
