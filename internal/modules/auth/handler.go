package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const sessionCookieMaxAge = 30 * 24 * 60 * 60

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
	token, _ := c.Cookie(SessionCookieName)
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
	redirectURL, err := h.service.Login(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start " + provider + " login",
		})
		return
	}

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

	response, token, err := h.service.CompleteOAuthLogin(c.Request.Context(), OAuthLoginRequest{
		Provider:  provider,
		Code:      code,
		State:     state,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	if errors.Is(err, ErrInvalidOAuthState) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid oauth state",
		})
		return
	}
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

	h.setSessionCookie(c, token)
	c.JSON(http.StatusOK, response)
}

func (h *Handler) setSessionCookie(c *gin.Context, token string) {
	c.SetCookie(SessionCookieName, token, sessionCookieMaxAge, "/", "", !h.development, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetCookie(SessionCookieName, "", -1, "/", "", !h.development, true)
}
