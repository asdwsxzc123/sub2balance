package model

import (
	"time"
)

type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	RequestID *uint     `gorm:"index" json:"request_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action    string    `gorm:"not null" json:"action"`
	Details   string    `gorm:"type:text" json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
