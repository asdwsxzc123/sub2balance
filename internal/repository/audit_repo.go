package repository

import (
	"context"
	"time"

	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AuditRepository) List(ctx context.Context, limit, offset int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *AuditRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AuditLog{}).Count(&count).Error
	return count, err
}

func (r *AuditRepository) CountByUserActionSince(ctx context.Context, userID uint, action string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("user_id = ? AND action = ? AND created_at >= ?", userID, action, since).
		Count(&count).Error
	return count, err
}
