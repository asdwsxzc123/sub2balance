package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type AdminHandler struct {
	conversionService *service.ConversionService
	auditService      *service.AuditService
}

func NewAdminHandler(conversionService *service.ConversionService, auditService *service.AuditService) *AdminHandler {
	return &AdminHandler{
		conversionService: conversionService,
		auditService:      auditService,
	}
}

type ApproveRequest struct {
	FinalAmount *float64 `json:"final_amount"`
	Note        string   `json:"note"`
}

type RejectRequest struct {
	Note string `json:"note" binding:"required"`
}

func (h *AdminHandler) ListAllRequests(c *gin.Context) {
	status := c.Query("status")

	requests, err := h.conversionService.ListAllRequests(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *AdminHandler) ApproveRequest(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.conversionService.ApproveRequest(c.Request.Context(), uint(id), userID, req.FinalAmount, req.Note); err != nil {
		if err == service.ErrRequestNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		if err == service.ErrInvalidStatus {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request cannot be approved in current status"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request approved successfully"})
}

func (h *AdminHandler) RejectRequest(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.conversionService.RejectRequest(c.Request.Context(), uint(id), userID, req.Note); err != nil {
		if err == service.ErrRequestNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		if err == service.ErrInvalidStatus {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request cannot be rejected in current status"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected successfully"})
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	logs, err := h.auditService.ListLogs(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}
