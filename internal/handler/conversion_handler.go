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

type QueryByEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type CreateRequestBody struct {
	RequestType      string  `json:"request_type"`
	UserEmail        string  `json:"user_email" binding:"required,email"`
	Sub2APIUserID    int64   `json:"sub2api_user_id" binding:"required"`
	SubscriptionID   int64   `json:"subscription_id" binding:"required"`
	GroupName        string  `json:"group_name"`
	OriginalAmount   float64 `json:"original_amount" binding:"gte=0"`
	ConsumedAmount   float64 `json:"consumed_amount" binding:"gte=0"`
	ConversionAmount float64 `json:"conversion_amount" binding:"gte=0"`
	TargetGroupID    *int64  `json:"target_group_id"`
	TargetGroupName  *string `json:"target_group_name"`
	ValidityDays     *int    `json:"validity_days"`
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

func (h *ConversionHandler) QueryByEmail(c *gin.Context) {
	var req QueryByEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.conversionService.QueryByEmail(c.Request.Context(), req.Email)
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

	requestType := req.RequestType
	if requestType == "" {
		requestType = "balance"
	}

	switch requestType {
	case "switch":
		if req.TargetGroupID == nil || *req.TargetGroupID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_group_id is required for switch"})
			return
		}
		if req.TargetGroupName == nil || *req.TargetGroupName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_group_name is required for switch"})
			return
		}
		if req.ValidityDays == nil || *req.ValidityDays <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "validity_days must be greater than 0"})
			return
		}
	case "balance":
		if req.ConversionAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversion_amount must be greater than 0"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_type"})
		return
	}

	input := &service.CreateRequestInput{
		Query: &service.QuerySubscriptionResult{
			UserEmail:        req.UserEmail,
			Sub2APIUserID:    req.Sub2APIUserID,
			SubscriptionID:   req.SubscriptionID,
			GroupName:        req.GroupName,
			OriginalAmount:   req.OriginalAmount,
			ConsumedAmount:   req.ConsumedAmount,
			ConversionAmount: req.ConversionAmount,
			Status:           "active",
		},
		RequestType:     requestType,
		TargetGroupID:   req.TargetGroupID,
		TargetGroupName: req.TargetGroupName,
		ValidityDays:    req.ValidityDays,
	}

	request, err := h.conversionService.CreateRequest(c.Request.Context(), userID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (h *ConversionHandler) ListGroups(c *gin.Context) {
	platform := c.Query("platform")

	groups, err := h.conversionService.ListAvailableGroups(c.Request.Context(), platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
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
