package model

import "time"

const (
	SettingKeySub2APIBaseURL          = "sub2api.base_url"
	SettingKeySub2APIAPIKey           = "sub2api.api_key"
	SettingKeyPasswordResetDailyLimit = "password_reset_daily_limit"
)

type SystemSetting struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy uint      `json:"updated_by"`
}

func (SystemSetting) TableName() string { return "system_settings" }
