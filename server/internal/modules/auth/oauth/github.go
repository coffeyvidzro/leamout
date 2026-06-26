package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type GitHub struct {
	config *oauth2.Config
}

func NewGitHub(cfg ProviderConfig) *GitHub {
	return &GitHub{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes: []string{
				"read:user",
				"user:email",
			},
			Endpoint: endpoints.GitHub,
		},
	}
}

func (g *GitHub) Name() string {
	return ProviderGitHub
}

func (g *GitHub) AuthURL(state string) string {
	return g.config.AuthCodeURL(state)
}

func (g *GitHub) Exchange(ctx context.Context, code string) (*Profile, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange github code: %w", err)
	}

	client := g.config.Client(ctx, token)

	user, err := g.fetchUser(client)
	if err != nil {
		return nil, err
	}

	email, verified, err := g.fetchPrimaryEmail(client)
	if err != nil {
		return nil, err
	}

	if email == "" {
		email = user.Email
	}

	return &Profile{
		Provider:       ProviderGitHub,
		ProviderUserID: strconv.FormatInt(user.ID, 10),
		Email:          email,
		EmailVerified:  verified,
		Name:           user.Name,
		AvatarURL:      user.AvatarURL,
	}, nil
}

func (g *GitHub) fetchUser(client *http.Client) (*githubUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("fetch github user: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user endpoint returned status %d", resp.StatusCode)
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode github user: %w", err)
	}

	return &user, nil
}

func (g *GitHub) fetchPrimaryEmail(client *http.Client) (string, bool, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", false, fmt.Errorf("fetch github emails: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github emails endpoint returned status %d", resp.StatusCode)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false, fmt.Errorf("decode github emails: %w", err)
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, email.Verified, nil
		}
	}

	return "", false, nil
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}
