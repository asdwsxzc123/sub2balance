package model

import "time"

type GroupPrice struct {
	GroupID   int64     `gorm:"primaryKey;autoIncrement:false" json:"group_id"`
	GroupName string    `gorm:"not null" json:"group_name"`
	Price     float64   `gorm:"not null" json:"price"`
	Currency  string    `gorm:"not null;default:CNY" json:"currency"`
	Note      string    `json:"note"`
	UpdatedBy uint      `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GroupPrice) TableName() string {
	return "group_prices"
}
