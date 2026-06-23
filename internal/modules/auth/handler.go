package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Google(c *gin.Context) {
	redirectURL, err := h.service.Login(c.Request.Context(), "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start google login",
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) GoogleCallback(c *gin.Context) {
	h.oauthCallback(c, "google")
}

func (h *Handler) GitHub(c *gin.Context) {
	redirectURL, err := h.service.Login(c.Request.Context(), "github")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start github login",
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *Handler) GitHubCallback(c *gin.Context) {
	h.oauthCallback(c, "github")
}

func (h *Handler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
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

	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
		"code":     code,
		"state":    state,
	})
}
