package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type UpgradeHandler struct {
	upgradeService *service.UpgradeService
}

func NewUpgradeHandler(upgradeService *service.UpgradeService) *UpgradeHandler {
	return &UpgradeHandler{
		upgradeService: upgradeService,
	}
}

type upgradeRequest struct {
	Version string `json:"version"`
}

// respondUpgradeError maps known business errors to specific statuses; anything
// else is logged server-side and answered with a fixed message so internal
// details never leak to clients.
func respondUpgradeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAlreadyUpToDate),
		errors.Is(err, service.ErrReleaseNotFound),
		errors.Is(err, service.ErrAssetNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUpgradeInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrContainerDeployment):
		c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
	default:
		log.Printf("system upgrade error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "升级失败，请查看服务器日志"})
	}
}

// GET /api/admin/system/version
func (h *UpgradeHandler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, h.upgradeService.CurrentInfo())
}

// GET /api/admin/system/latest
func (h *UpgradeHandler) GetLatest(c *gin.Context) {
	info, err := h.upgradeService.CheckLatest(c.Request.Context())
	if err != nil {
		log.Printf("check latest version error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法获取最新版本信息，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, info)
}

// POST /api/admin/system/upgrade
// Body is optional; an empty or missing version means "upgrade to latest".
func (h *UpgradeHandler) Upgrade(c *gin.Context) {
	var req upgradeRequest
	_ = c.ShouldBindJSON(&req)

	operatorID, _ := middleware.GetUserID(c)

	from, to, err := h.upgradeService.Upgrade(c.Request.Context(), operatorID, req.Version)
	if err != nil {
		respondUpgradeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "restarting",
		"from":   from,
		"to":     to,
	})
	// Restart only after the response above is flushed to the client.
	h.upgradeService.Restart()
}
