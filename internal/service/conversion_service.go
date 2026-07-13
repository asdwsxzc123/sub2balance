package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

// Extracts the price after a pipe separator in a group name. Handles full-width (｜),
// half-width (|), and the 丨 ideograph. Matches e.g. "codex 50刀｜289 套餐" → 289,
// "codex | 300刀" → 300. Returns 0 when nothing matches.
var purchaseAmountPattern = regexp.MustCompile(`[|｜丨]\s*([0-9]+(?:\.[0-9]+)?)`)

func parsePurchaseAmount(groupName string) (float64, bool) {
	m := purchaseAmountPattern.FindStringSubmatch(groupName)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

var (
	ErrRequestNotFound     = errors.New("conversion request not found")
	ErrInvalidStatus       = errors.New("invalid request status")
	ErrUnauthorized        = errors.New("unauthorized operation")
	ErrSubscriptionInvalid = errors.New("subscription is not valid for conversion")
)

type ConversionService struct {
	convRepo       *repository.ConversionRepository
	groupPriceRepo *repository.GroupPriceRepository
	sub2apiClient  *Sub2APIClient
	auditService   *AuditService
}

func NewConversionService(
	convRepo *repository.ConversionRepository,
	groupPriceRepo *repository.GroupPriceRepository,
	sub2apiClient *Sub2APIClient,
	auditService *AuditService,
) *ConversionService {
	return &ConversionService{
		convRepo:       convRepo,
		groupPriceRepo: groupPriceRepo,
		sub2apiClient:  sub2apiClient,
		auditService:   auditService,
	}
}

type QuerySubscriptionResult struct {
	UserEmail        string  `json:"user_email"`
	Sub2APIUserID    int64   `json:"sub2api_user_id"`
	SubscriptionID   int64   `json:"subscription_id"`
	GroupID          int64   `json:"group_id"`
	GroupName        string  `json:"group_name"`
	Platform         string  `json:"platform,omitempty"`
	OriginalAmount   float64 `json:"original_amount"`
	OriginalSource   string  `json:"original_source,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	ConsumedAmount   float64 `json:"consumed_amount"`
	ConversionAmount float64 `json:"conversion_amount"`
	Status           string  `json:"status"`
	ExpiresAt        string  `json:"expires_at,omitempty"`
}

type CreateRequestInput struct {
	Query           *QuerySubscriptionResult
	RequestType     string
	TargetGroupID   *int64
	TargetGroupName *string
	ValidityDays    *int
}

type AvailableGroup struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Platform      string   `json:"platform"`
	DailyLimitUSD *float64 `json:"daily_limit_usd"`
}

type QueryByEmailResult struct {
	User          *Sub2APIUser               `json:"user"`
	Subscriptions []*QuerySubscriptionResult `json:"subscriptions"`
}

func (s *ConversionService) buildQueryResult(ctx context.Context, user *Sub2APIUser, sub *Sub2APISubscription) *QuerySubscriptionResult {
	email := ""
	if user != nil {
		email = user.Email
	} else if sub.User != nil {
		email = sub.User.Email
	}

	groupID := sub.GroupID
	groupName := ""
	platform := ""
	var usdLimit float64
	if sub.Group != nil {
		groupName = sub.Group.Name
		platform = sub.Group.Platform
		switch {
		case sub.Group.MonthlyLimitUSD != nil:
			usdLimit = *sub.Group.MonthlyLimitUSD
		case sub.Group.WeeklyLimitUSD != nil:
			usdLimit = *sub.Group.WeeklyLimitUSD
		case sub.Group.DailyLimitUSD != nil:
			usdLimit = *sub.Group.DailyLimitUSD
		}
	}

	// Pricing priority: DB mapping > parsed from group name > upstream USD limit.
	var original float64
	var source string
	currency := "CNY"

	if s.groupPriceRepo != nil && groupID > 0 {
		if gp, err := s.groupPriceRepo.Get(ctx, groupID); err == nil && gp != nil {
			original = gp.Price
			source = "mapping"
			if gp.Currency != "" {
				currency = gp.Currency
			}
		}
	}
	if source == "" {
		if parsed, ok := parsePurchaseAmount(groupName); ok {
			original = parsed
			source = "parsed"
		}
	}
	if source == "" {
		original = usdLimit
		source = "limit"
		currency = "USD"
	}

	var consumed float64
	switch {
	case sub.MonthlyUsageUSD > 0:
		consumed = sub.MonthlyUsageUSD
	case sub.WeeklyUsageUSD > 0:
		consumed = sub.WeeklyUsageUSD
	default:
		consumed = sub.DailyUsageUSD
	}

	// 1:1 抵扣：转换金额 = 实付金额 - 已消耗美元额度，无论原始币种。
	conversion := original - consumed
	if conversion < 0 {
		conversion = 0
	}

	userID := sub.UserID
	if user != nil {
		userID = user.ID
	}

	return &QuerySubscriptionResult{
		UserEmail:        email,
		Sub2APIUserID:    userID,
		SubscriptionID:   sub.ID,
		GroupID:          groupID,
		GroupName:        groupName,
		Platform:         platform,
		OriginalAmount:   original,
		OriginalSource:   source,
		Currency:         currency,
		ConsumedAmount:   consumed,
		ConversionAmount: conversion,
		Status:           sub.Status,
		ExpiresAt:        sub.ExpiresAt,
	}
}

func (s *ConversionService) QuerySubscription(ctx context.Context, userID int64, subscriptionID int64) (*QuerySubscriptionResult, error) {
	user, err := s.sub2apiClient.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	sub, err := s.sub2apiClient.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.Status != "active" {
		return nil, ErrSubscriptionInvalid
	}

	return s.buildQueryResult(ctx, user, sub), nil
}

func (s *ConversionService) QueryByEmail(ctx context.Context, email string) (*QueryByEmailResult, error) {
	user, err := s.sub2apiClient.SearchUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// The user-search list endpoint returns embedded subscriptions without usage data
	// (daily/weekly/monthly_usage_usd = 0). Fetch each subscription individually to
	// get accurate usage so conversion_amount is not incorrectly inflated.
	type indexed struct {
		i   int
		sub *Sub2APISubscription
	}
	ch := make(chan indexed, len(user.Subscriptions))
	var wg sync.WaitGroup
	for i := range user.Subscriptions {
		wg.Add(1)
		go func(i int, embedded Sub2APISubscription) {
			defer wg.Done()
			full, err := s.sub2apiClient.GetSubscription(ctx, embedded.ID)
			if err != nil {
				// Fall back to embedded data if the individual fetch fails.
				ch <- indexed{i, &embedded}
				return
			}
			ch <- indexed{i, full}
		}(i, user.Subscriptions[i])
	}
	wg.Wait()
	close(ch)

	subs := make([]*Sub2APISubscription, len(user.Subscriptions))
	for item := range ch {
		subs[item.i] = item.sub
	}

	results := make([]*QuerySubscriptionResult, 0, len(subs))
	for _, sub := range subs {
		results = append(results, s.buildQueryResult(ctx, user, sub))
	}

	// Strip embedded subscriptions from the user payload to keep the response focused.
	userCopy := *user
	userCopy.Subscriptions = nil

	return &QueryByEmailResult{
		User:          &userCopy,
		Subscriptions: results,
	}, nil
}

func (s *ConversionService) CreateRequest(ctx context.Context, submitterID uint, input *CreateRequestInput) (*model.ConversionRequest, error) {
	requestType := input.RequestType
	if requestType == "" {
		requestType = "balance"
	}

	query := input.Query
	req := &model.ConversionRequest{
		RequestType:      requestType,
		UserEmail:        query.UserEmail,
		Sub2APIUserID:    query.Sub2APIUserID,
		SubscriptionID:   query.SubscriptionID,
		GroupName:        query.GroupName,
		OriginalAmount:   query.OriginalAmount,
		ConsumedAmount:   query.ConsumedAmount,
		ConversionAmount: query.ConversionAmount,
		TargetGroupID:    input.TargetGroupID,
		TargetGroupName:  input.TargetGroupName,
		ValidityDays:     input.ValidityDays,
		Status:           "pending",
		SubmittedBy:      submitterID,
	}

	if err := s.convRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	details := map[string]interface{}{
		"request_type":    req.RequestType,
		"user_email":      req.UserEmail,
		"subscription_id": req.SubscriptionID,
	}
	switch req.RequestType {
	case "switch", "bind":
		details["target_group_id"] = req.TargetGroupID
		details["target_group_name"] = req.TargetGroupName
		details["validity_days"] = req.ValidityDays
	case "unbind":
		details["group_name"] = req.GroupName
	default:
		details["conversion_amount"] = req.ConversionAmount
	}
	_ = s.auditService.LogWithRequest(ctx, submitterID, req.ID, "create_request", details)

	return req, nil
}

func (s *ConversionService) ListAvailableGroups(ctx context.Context, platform string) ([]AvailableGroup, error) {
	groups, err := s.sub2apiClient.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]AvailableGroup, 0, len(groups))
	for _, g := range groups {
		if g.SubscriptionType != "subscription" {
			continue
		}
		if g.Status != "active" {
			continue
		}
		if platform != "" && g.Platform != platform {
			continue
		}
		result = append(result, AvailableGroup{
			ID:            g.ID,
			Name:          g.Name,
			Platform:      g.Platform,
			DailyLimitUSD: g.DailyLimitUSD,
		})
	}
	return result, nil
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

	var auditDetails map[string]interface{}

	switch req.RequestType {
	case "switch":
		if req.TargetGroupID == nil || req.ValidityDays == nil {
			return fmt.Errorf("switch request missing target_group_id or validity_days")
		}

		newSub, err := s.sub2apiClient.AssignSubscription(ctx, req.Sub2APIUserID, *req.TargetGroupID, *req.ValidityDays)
		if err != nil {
			return fmt.Errorf("failed to assign new subscription: %w", err)
		}

		if err := s.sub2apiClient.CancelSubscription(ctx, req.SubscriptionID); err != nil {
			_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "cancellation_failed", map[string]interface{}{
				"error":               err.Error(),
				"source_subscription": req.SubscriptionID,
				"new_subscription":    newSub.ID,
				"target_group_id":     *req.TargetGroupID,
			})
			return fmt.Errorf("new subscription assigned but source cancellation failed: %w", err)
		}

		auditDetails = map[string]interface{}{
			"request_type":        "switch",
			"source_subscription": req.SubscriptionID,
			"target_group_id":     *req.TargetGroupID,
			"target_group_name":   req.TargetGroupName,
			"validity_days":       *req.ValidityDays,
			"new_subscription":    newSub.ID,
			"note":                note,
		}

	case "unbind":
		// Verify ownership before cancelling: a stale or tampered request must
		// never cancel a subscription that belongs to someone else.
		sub, err := s.sub2apiClient.GetSubscription(ctx, req.SubscriptionID)
		if err != nil {
			return fmt.Errorf("failed to get subscription: %w", err)
		}
		if sub.UserID != req.Sub2APIUserID {
			return fmt.Errorf("订阅不属于该用户，已阻止解绑")
		}

		if err := s.sub2apiClient.CancelSubscription(ctx, req.SubscriptionID); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}

		auditDetails = map[string]interface{}{
			"request_type":    "unbind",
			"subscription_id": req.SubscriptionID,
			"group_name":      req.GroupName,
			"note":            note,
		}

	case "bind":
		if req.TargetGroupID == nil || req.ValidityDays == nil {
			return fmt.Errorf("bind request missing target_group_id or validity_days")
		}

		newSub, err := s.sub2apiClient.AssignSubscription(ctx, req.Sub2APIUserID, *req.TargetGroupID, *req.ValidityDays)
		if err != nil {
			return fmt.Errorf("failed to assign subscription: %w", err)
		}

		auditDetails = map[string]interface{}{
			"request_type":      "bind",
			"target_group_id":   *req.TargetGroupID,
			"target_group_name": req.TargetGroupName,
			"validity_days":     *req.ValidityDays,
			"new_subscription":  newSub.ID,
			"note":              note,
		}

	case "balance", "":
		amountToAdd := req.ConversionAmount
		if finalAmount != nil {
			amountToAdd = *finalAmount
			req.FinalAmount = finalAmount
		}

		noteText := fmt.Sprintf("Converted from subscription #%d", req.SubscriptionID)
		if note != "" {
			noteText += " - " + note
		}

		if err := s.sub2apiClient.AddBalance(ctx, req.Sub2APIUserID, amountToAdd, noteText); err != nil {
			return fmt.Errorf("failed to add balance: %w", err)
		}

		if err := s.sub2apiClient.CancelSubscription(ctx, req.SubscriptionID); err != nil {
			_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "cancellation_failed", map[string]interface{}{
				"error":           err.Error(),
				"subscription_id": req.SubscriptionID,
				"balance_added":   amountToAdd,
			})
			return fmt.Errorf("balance added but subscription cancellation failed: %w", err)
		}

		auditDetails = map[string]interface{}{
			"request_type":    "balance",
			"final_amount":    amountToAdd,
			"subscription_id": req.SubscriptionID,
			"note":            note,
		}

	default:
		return fmt.Errorf("unknown request_type: %s", req.RequestType)
	}

	now := time.Now()
	req.Status = "approved"
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := s.convRepo.Update(ctx, req); err != nil {
		return err
	}

	_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "approve_request", auditDetails)

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
