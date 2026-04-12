package repository

import (
	"context"
	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type ConversionRepository struct {
	db *gorm.DB
}

func NewConversionRepository(db *gorm.DB) *ConversionRepository {
	return &ConversionRepository{db: db}
}

func (r *ConversionRepository) Create(ctx context.Context, req *model.ConversionRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *ConversionRepository) GetByID(ctx context.Context, id uint) (*model.ConversionRequest, error) {
	var req model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *ConversionRepository) ListBySubmitter(ctx context.Context, userID uint) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Where("submitted_by = ?", userID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) ListByStatus(ctx context.Context, status string) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) ListAll(ctx context.Context) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) Update(ctx context.Context, req *model.ConversionRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}
