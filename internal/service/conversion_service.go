package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

var (
	ErrRequestNotFound     = errors.New("conversion request not found")
	ErrInvalidStatus       = errors.New("invalid request status")
	ErrUnauthorized        = errors.New("unauthorized operation")
	ErrSubscriptionInvalid = errors.New("subscription is not valid for conversion")
)

type ConversionService struct {
	convRepo      *repository.ConversionRepository
	sub2apiClient *Sub2APIClient
	auditService  *AuditService
}

func NewConversionService(
	convRepo *repository.ConversionRepository,
	sub2apiClient *Sub2APIClient,
	auditService *AuditService,
) *ConversionService {
	return &ConversionService{
		convRepo:      convRepo,
		sub2apiClient: sub2apiClient,
		auditService:  auditService,
	}
}

type QuerySubscriptionResult struct {
	UserEmail        string  `json:"user_email"`
	Sub2APIUserID    int64   `json:"sub2api_user_id"`
	SubscriptionID   int64   `json:"subscription_id"`
	GroupName        string  `json:"group_name"`
	OriginalAmount   float64 `json:"original_amount"`
	ConsumedAmount   float64 `json:"consumed_amount"`
	ConversionAmount float64 `json:"conversion_amount"`
	Status           string  `json:"status"`
}

func (s *ConversionService) QuerySubscription(ctx context.Context, userID int64, subscriptionID int64) (*QuerySubscriptionResult, error) {
	// Get user info
	user, err := s.sub2apiClient.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get subscription info
	sub, err := s.sub2apiClient.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription is active
	if sub.Status != "active" {
		return nil, ErrSubscriptionInvalid
	}

	// Calculate consumed amount (use monthly as the primary metric)
	consumedAmount := sub.MonthlyUsedUSD

	// For this implementation, we'll use monthly limit as the "original amount"
	// In a real scenario, you'd need to track the actual paid amount
	originalAmount := sub.MonthlyLimitUSD

	// Calculate conversion amount
	conversionAmount := originalAmount - consumedAmount
	if conversionAmount < 0 {
		conversionAmount = 0
	}

	return &QuerySubscriptionResult{
		UserEmail:        user.Email,
		Sub2APIUserID:    user.ID,
		SubscriptionID:   sub.ID,
		GroupName:        sub.GroupName,
		OriginalAmount:   originalAmount,
		ConsumedAmount:   consumedAmount,
		ConversionAmount: conversionAmount,
		Status:           sub.Status,
	}, nil
}

func (s *ConversionService) CreateRequest(ctx context.Context, submitterID uint, query *QuerySubscriptionResult) (*model.ConversionRequest, error) {
	req := &model.ConversionRequest{
		UserEmail:        query.UserEmail,
		Sub2APIUserID:    query.Sub2APIUserID,
		SubscriptionID:   query.SubscriptionID,
		GroupName:        query.GroupName,
		OriginalAmount:   query.OriginalAmount,
		ConsumedAmount:   query.ConsumedAmount,
		ConversionAmount: query.ConversionAmount,
		Status:           "pending",
		SubmittedBy:      submitterID,
	}

	if err := s.convRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, submitterID, req.ID, "create_request", map[string]interface{}{
		"user_email":        req.UserEmail,
		"subscription_id":   req.SubscriptionID,
		"conversion_amount": req.ConversionAmount,
	})

	return req, nil
}

func (s *ConversionService) GetRequest(ctx context.Context, id uint, userID uint, isAdmin bool) (*model.ConversionRequest, error) {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrRequestNotFound
	}

	// Staff can only view their own requests
	if !isAdmin && req.SubmittedBy != userID {
		return nil, ErrUnauthorized
	}

	return req, nil
}

func (s *ConversionService) ListMyRequests(ctx context.Context, userID uint) ([]*model.ConversionRequest, error) {
	return s.convRepo.ListBySubmitter(ctx, userID)
}

func (s *ConversionService) ListAllRequests(ctx context.Context, status string) ([]*model.ConversionRequest, error) {
	if status != "" {
		return s.convRepo.ListByStatus(ctx, status)
	}
	return s.convRepo.ListAll(ctx)
}

func (s *ConversionService) ApproveRequest(ctx context.Context, id uint, reviewerID uint, finalAmount *float64, note string) error {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != "pending" {
		return ErrInvalidStatus
	}

	// Use final amount if provided, otherwise use conversion amount
	amountToAdd := req.ConversionAmount
	if finalAmount != nil {
		amountToAdd = *finalAmount
		req.FinalAmount = finalAmount
	}

	// Add balance to sub2api
	noteText := fmt.Sprintf("Converted from subscription #%d", req.SubscriptionID)
	if note != "" {
		noteText += " - " + note
	}

	if err := s.sub2apiClient.AddBalance(ctx, req.Sub2APIUserID, amountToAdd, noteText); err != nil {
		return fmt.Errorf("failed to add balance: %w", err)
	}

	// Cancel subscription
	if err := s.sub2apiClient.CancelSubscription(ctx, req.SubscriptionID); err != nil {
		// Balance was added but cancellation failed - log this critical error
		_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "cancellation_failed", map[string]interface{}{
			"error":           err.Error(),
			"subscription_id": req.SubscriptionID,
			"balance_added":   amountToAdd,
		})
		return fmt.Errorf("balance added but subscription cancellation failed: %w", err)
	}

	// Update request status
	now := time.Now()
	req.Status = "approved"
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := s.convRepo.Update(ctx, req); err != nil {
		return err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "approve_request", map[string]interface{}{
		"final_amount":    amountToAdd,
		"subscription_id": req.SubscriptionID,
		"note":            note,
	})

	return nil
}

func (s *ConversionService) RejectRequest(ctx context.Context, id uint, reviewerID uint, note string) error {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != "pending" {
		return ErrInvalidStatus
	}

	now := time.Now()
	req.Status = "rejected"
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := s.convRepo.Update(ctx, req); err != nil {
		return err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "reject_request", map[string]interface{}{
		"note": note,
	})

	return nil
}
