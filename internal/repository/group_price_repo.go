package repository

import (
	"context"
	"errors"

	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type GroupPriceRepository struct {
	db *gorm.DB
}

func NewGroupPriceRepository(db *gorm.DB) *GroupPriceRepository {
	return &GroupPriceRepository{db: db}
}

func (r *GroupPriceRepository) Get(ctx context.Context, groupID int64) (*model.GroupPrice, error) {
	var gp model.GroupPrice
	err := r.db.WithContext(ctx).First(&gp, "group_id = ?", groupID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &gp, nil
}

func (r *GroupPriceRepository) List(ctx context.Context) ([]*model.GroupPrice, error) {
	var items []*model.GroupPrice
	err := r.db.WithContext(ctx).Order("group_id ASC").Find(&items).Error
	return items, err
}

func (r *GroupPriceRepository) Upsert(ctx context.Context, gp *model.GroupPrice) error {
	return r.db.WithContext(ctx).Save(gp).Error
}

func (r *GroupPriceRepository) Delete(ctx context.Context, groupID int64) error {
	return r.db.WithContext(ctx).Delete(&model.GroupPrice{}, "group_id = ?", groupID).Error
}
