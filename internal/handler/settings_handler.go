package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type SettingsHandler struct {
	settings *service.SettingsService
	client   *service.Sub2APIClient
	audit    *service.AuditService
}

func NewSettingsHandler(settings *service.SettingsService, client *service.Sub2APIClient, audit *service.AuditService) *SettingsHandler {
	return &SettingsHandler{settings: settings, client: client, audit: audit}
}

type sub2apiSettingsResponse struct {
	BaseURL      string `json:"base_url"`
	APIKeyMasked string `json:"api_key_masked"`
	Configured   bool   `json:"configured"`
}

type updateSub2APIRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

func maskKey(k string) string {
	if k == "" {
		return ""
	}
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

// GET /api/admin/settings/sub2api
func (h *SettingsHandler) GetSub2API(c *gin.Context) {
	baseURL, apiKey := h.settings.Sub2API()
	c.JSON(http.StatusOK, sub2apiSettingsResponse{
		BaseURL:      baseURL,
		APIKeyMasked: maskKey(apiKey),
		Configured:   baseURL != "" && apiKey != "",
	})
}

// PUT /api/admin/settings/sub2api
func (h *SettingsHandler) UpdateSub2API(c *gin.Context) {
	var req updateSub2APIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url is required"})
		return
	}

	actorID, _ := middleware.GetUserID(c)
	if err := h.settings.UpdateSub2API(c.Request.Context(), service.Sub2APIUpdate{
		BaseURL: req.BaseURL,
		APIKey:  req.APIKey,
	}, actorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.audit.Log(c.Request.Context(), actorID, "update_sub2api_settings", map[string]any{
		"base_url":        req.BaseURL,
		"api_key_rotated": req.APIKey != "",
	})

	h.GetSub2API(c)
}

// POST /api/admin/settings/sub2api/test
// Tests connectivity with provided credentials (or current stored ones if body empty).
func (h *SettingsHandler) TestSub2API(c *gin.Context) {
	var req updateSub2APIRequest
	_ = c.ShouldBindJSON(&req)

	baseURL := req.BaseURL
	apiKey := req.APIKey
	if baseURL == "" || apiKey == "" {
		curURL, curKey := h.settings.Sub2API()
		if baseURL == "" {
			baseURL = curURL
		}
		if apiKey == "" {
			apiKey = curKey
		}
	}
	if baseURL == "" || apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url and api_key are required"})
		return
	}

	if err := h.client.TestConnection(c.Request.Context(), baseURL, apiKey); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
