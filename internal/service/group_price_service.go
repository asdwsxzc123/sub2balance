package service

import (
	"context"
	"fmt"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

type GroupPriceService struct {
	repo          *repository.GroupPriceRepository
	sub2apiClient *Sub2APIClient
	auditService  *AuditService
}

func NewGroupPriceService(
	repo *repository.GroupPriceRepository,
	sub2apiClient *Sub2APIClient,
	auditService *AuditService,
) *GroupPriceService {
	return &GroupPriceService{
		repo:          repo,
		sub2apiClient: sub2apiClient,
		auditService:  auditService,
	}
}

type GroupPriceRow struct {
	GroupID       int64    `json:"group_id"`
	GroupName     string   `json:"group_name"`
	Platform      string   `json:"platform"`
	DailyLimitUSD *float64 `json:"daily_limit_usd"`
	ParsedPrice   *float64 `json:"parsed_price,omitempty"`
	Price         *float64 `json:"price,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	Note          string   `json:"note,omitempty"`
}

// List returns every active subscription group from sub2api merged with any saved price override.
func (s *GroupPriceService) List(ctx context.Context) ([]GroupPriceRow, error) {
	groups, err := s.sub2apiClient.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	saved, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	savedByID := make(map[int64]*model.GroupPrice, len(saved))
	for _, gp := range saved {
		savedByID[gp.GroupID] = gp
	}

	rows := make([]GroupPriceRow, 0, len(groups))
	for _, g := range groups {
		if g.SubscriptionType != "subscription" || g.Status != "active" {
			continue
		}
		row := GroupPriceRow{
			GroupID:       g.ID,
			GroupName:     g.Name,
			Platform:      g.Platform,
			DailyLimitUSD: g.DailyLimitUSD,
		}
		if parsed, ok := parsePurchaseAmount(g.Name); ok {
			p := parsed
			row.ParsedPrice = &p
		}
		if gp, ok := savedByID[g.ID]; ok {
			p := gp.Price
			row.Price = &p
			row.Currency = gp.Currency
			row.Note = gp.Note
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type UpsertGroupPriceInput struct {
	GroupID   int64
	GroupName string
	Price     float64
	Currency  string
	Note      string
	UpdatedBy uint
}

func (s *GroupPriceService) Upsert(ctx context.Context, in *UpsertGroupPriceInput) (*model.GroupPrice, error) {
	if in.GroupID <= 0 {
		return nil, fmt.Errorf("group_id is required")
	}
	if in.Price < 0 {
		return nil, fmt.Errorf("price cannot be negative")
	}
	currency := in.Currency
	if currency == "" {
		currency = "CNY"
	}

	gp := &model.GroupPrice{
		GroupID:   in.GroupID,
		GroupName: in.GroupName,
		Price:     in.Price,
		Currency:  currency,
		Note:      in.Note,
		UpdatedBy: in.UpdatedBy,
	}
	if err := s.repo.Upsert(ctx, gp); err != nil {
		return nil, err
	}

	_ = s.auditService.Log(ctx, in.UpdatedBy, "upsert_group_price", map[string]interface{}{
		"group_id":   gp.GroupID,
		"group_name": gp.GroupName,
		"price":      gp.Price,
		"currency":   gp.Currency,
	})

	return gp, nil
}

func (s *GroupPriceService) Delete(ctx context.Context, groupID int64, actorID uint) error {
	if err := s.repo.Delete(ctx, groupID); err != nil {
		return err
	}
	_ = s.auditService.Log(ctx, actorID, "delete_group_price", map[string]interface{}{
		"group_id": groupID,
	})
	return nil
}
