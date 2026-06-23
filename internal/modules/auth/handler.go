package auth

import (
	"errors"
	"net/http"

	"github.com/cuffeyvidzro/leamout/internal/http/middleware"
	"github.com/gin-gonic/gin"
)

const (
	sessionCookieMaxAge = 30 * 24 * 60 * 60
	oauthStateMaxAge    = 10 * 60
)

type Handler struct {
	service     *Service
	development bool
}

func NewHandler(service *Service, development bool) *Handler {
	return &Handler{
		service:     service,
		development: development,
	}
}

func (h *Handler) Google(c *gin.Context) {
	h.oauthLogin(c, "google")
}

func (h *Handler) GoogleCallback(c *gin.Context) {
	h.oauthCallback(c, "google")
}

func (h *Handler) GitHub(c *gin.Context) {
	h.oauthLogin(c, "github")
}

func (h *Handler) GitHubCallback(c *gin.Context) {
	h.oauthCallback(c, "github")
}

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(middleware.SessionCookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to log out",
		})
		return
	}

	h.clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
}

func (h *Handler) oauthLogin(c *gin.Context, provider string) {
	state, err := h.service.NewOAuthState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start " + provider + " login",
		})
		return
	}

	redirectURL, err := h.service.OAuthURL(provider, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start " + provider + " login",
		})
		return
	}

	h.setOAuthStateCookie(c, provider, state)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) oauthCallback(c *gin.Context, provider string) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing code or state",
		})
		return
	}
	if !h.validOAuthState(c, provider, state) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid oauth state",
		})
		return
	}

	response, err := h.service.CompleteOAuthLogin(c.Request.Context(), OAuthLoginRequest{
		Provider:  provider,
		Code:      code,
		State:     state,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	if errors.Is(err, ErrUnverifiedEmail) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "oauth email is not verified",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to complete " + provider + " login",
		})
		return
	}

	h.clearOAuthStateCookie(c, provider)
	h.setSessionCookie(c, response.Session.Token)
	c.JSON(http.StatusOK, response)
}

func (h *Handler) setSessionCookie(c *gin.Context, token string) {
	c.SetCookie(middleware.SessionCookieName, token, sessionCookieMaxAge, "/", "", !h.development, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", !h.development, true)
}

func (h *Handler) setOAuthStateCookie(c *gin.Context, provider, state string) {
	c.SetCookie(oauthStateCookieName(provider), state, oauthStateMaxAge, "/", "", !h.development, true)
}

func (h *Handler) clearOAuthStateCookie(c *gin.Context, provider string) {
	c.SetCookie(oauthStateCookieName(provider), "", -1, "/", "", !h.development, true)
}

func (h *Handler) validOAuthState(c *gin.Context, provider, state string) bool {
	storedState, err := c.Cookie(oauthStateCookieName(provider))
	return err == nil && storedState == state
}

func oauthStateCookieName(provider string) string {
	return "leamout_oauth_state_" + provider
}
