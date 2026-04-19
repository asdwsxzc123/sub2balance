package repository

import (
	"context"
	"errors"

	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) Get(ctx context.Context, key string) (*model.SystemSetting, error) {
	var s model.SystemSetting
	err := r.db.WithContext(ctx).First(&s, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SettingRepository) All(ctx context.Context) ([]model.SystemSetting, error) {
	var items []model.SystemSetting
	err := r.db.WithContext(ctx).Find(&items).Error
	return items, err
}

func (r *SettingRepository) Upsert(ctx context.Context, s *model.SystemSetting) error {
	return r.db.WithContext(ctx).Save(s).Error
}
