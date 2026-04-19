package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type GroupPriceHandler struct {
	svc *service.GroupPriceService
}

func NewGroupPriceHandler(svc *service.GroupPriceService) *GroupPriceHandler {
	return &GroupPriceHandler{svc: svc}
}

type UpsertGroupPriceRequest struct {
	GroupName string  `json:"group_name" binding:"required"`
	Price     float64 `json:"price" binding:"gte=0"`
	Currency  string  `json:"currency"`
	Note      string  `json:"note"`
}

func (h *GroupPriceHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *GroupPriceHandler) Upsert(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	var req UpsertGroupPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actorID, _ := middleware.GetUserID(c)
	gp, err := h.svc.Upsert(c.Request.Context(), &service.UpsertGroupPriceInput{
		GroupID:   groupID,
		GroupName: req.GroupName,
		Price:     req.Price,
		Currency:  req.Currency,
		Note:      req.Note,
		UpdatedBy: actorID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gp)
}

func (h *GroupPriceHandler) Delete(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	actorID, _ := middleware.GetUserID(c)
	if err := h.svc.Delete(c.Request.Context(), groupID, actorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
