package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type PasswordResetHandler struct {
	passwordResetService *service.PasswordResetService
}

func NewPasswordResetHandler(passwordResetService *service.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{
		passwordResetService: passwordResetService,
	}
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type passwordResetUserView struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// respondPasswordResetError maps known business errors to specific statuses and
// messages; anything else is logged server-side and answered with a fixed
// message so upstream response bodies never leak to clients.
func respondPasswordResetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAdminAccountForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrAccountNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDailyLimitExceeded):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
	default:
		log.Printf("password reset error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请稍后重试"})
	}
}

func (h *PasswordResetHandler) Query(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID, _ := middleware.GetUserID(c)

	user, err := h.passwordResetService.QueryAccount(c.Request.Context(), operatorID, req.Email)
	if err != nil {
		respondPasswordResetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": passwordResetUserView{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		},
	})
}

func (h *PasswordResetHandler) Reset(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID, _ := middleware.GetUserID(c)

	newPassword, user, err := h.passwordResetService.ResetPassword(c.Request.Context(), operatorID, req.Email)
	if err != nil {
		respondPasswordResetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":        user.Email,
		"user_id":      user.ID,
		"new_password": newPassword,
	})
}
