package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type ConversionHandler struct {
	conversionService *service.ConversionService
}

func NewConversionHandler(conversionService *service.ConversionService) *ConversionHandler {
	return &ConversionHandler{
		conversionService: conversionService,
	}
}

type QueryRequest struct {
	UserID         int64 `json:"user_id" binding:"required"`
	SubscriptionID int64 `json:"subscription_id" binding:"required"`
}

type CreateRequestBody struct {
	UserEmail        string  `json:"user_email" binding:"required"`
	Sub2APIUserID    int64   `json:"sub2api_user_id" binding:"required"`
	SubscriptionID   int64   `json:"subscription_id" binding:"required"`
	GroupName        string  `json:"group_name" binding:"required"`
	OriginalAmount   float64 `json:"original_amount" binding:"required"`
	ConsumedAmount   float64 `json:"consumed_amount" binding:"required"`
	ConversionAmount float64 `json:"conversion_amount" binding:"required"`
}

func (h *ConversionHandler) QuerySubscription(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.conversionService.QuerySubscription(c.Request.Context(), req.UserID, req.SubscriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ConversionHandler) CreateRequest(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queryResult := &service.QuerySubscriptionResult{
		UserEmail:        req.UserEmail,
		Sub2APIUserID:    req.Sub2APIUserID,
		SubscriptionID:   req.SubscriptionID,
		GroupName:        req.GroupName,
		OriginalAmount:   req.OriginalAmount,
		ConsumedAmount:   req.ConsumedAmount,
		ConversionAmount: req.ConversionAmount,
		Status:           "active",
	}

	request, err := h.conversionService.CreateRequest(c.Request.Context(), userID, queryResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (h *ConversionHandler) GetRequest(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	role, _ := middleware.GetUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	isAdmin := role == "admin"
	request, err := h.conversionService.GetRequest(c.Request.Context(), uint(id), userID, isAdmin)
	if err != nil {
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *ConversionHandler) ListMyRequests(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	requests, err := h.conversionService.ListMyRequests(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}
