package model

import (
	"gorm.io/gorm"
	"time"
)

type ConversionRequest struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	RequestType      string         `gorm:"not null;default:balance;index;check:request_type IN ('balance','switch','bind','unbind')" json:"request_type"`
	UserEmail        string         `gorm:"not null;index" json:"user_email"`
	Sub2APIUserID    int64          `gorm:"not null" json:"sub2api_user_id"`
	SubscriptionID   int64          `gorm:"not null" json:"subscription_id"`
	GroupName        string         `gorm:"not null" json:"group_name"`
	OriginalAmount   float64        `gorm:"not null" json:"original_amount"`
	ConsumedAmount   float64        `gorm:"not null" json:"consumed_amount"`
	ConversionAmount float64        `gorm:"not null" json:"conversion_amount"`
	FinalAmount      *float64       `json:"final_amount"`
	TargetGroupID    *int64         `json:"target_group_id"`
	TargetGroupName  *string        `json:"target_group_name"`
	ValidityDays     *int           `json:"validity_days"`
	Status           string         `gorm:"not null;index;check:status IN ('pending', 'approved', 'rejected')" json:"status"`
	SubmittedBy      uint           `gorm:"not null;index" json:"submitted_by"`
	SubmittedByUser  *User          `gorm:"foreignKey:SubmittedBy" json:"submitted_by_user,omitempty"`
	ReviewedBy       *uint          `gorm:"index" json:"reviewed_by"`
	ReviewedByUser   *User          `gorm:"foreignKey:ReviewedBy" json:"reviewed_by_user,omitempty"`
	ReviewNote       string         `json:"review_note"`
	ReviewedAt       *time.Time     `json:"reviewed_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ConversionRequest) TableName() string {
	return "conversion_requests"
}
