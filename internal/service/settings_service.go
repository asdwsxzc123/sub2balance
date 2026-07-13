package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

var ErrSub2APINotConfigured = errors.New("sub2api is not configured yet, please set base_url and api_key in admin settings")

// DefaultPasswordResetDailyLimit is used when the setting is absent or invalid.
const DefaultPasswordResetDailyLimit = 5

type SettingsService struct {
	repo *repository.SettingRepository

	mu    sync.RWMutex
	cache map[string]string
}

func NewSettingsService(repo *repository.SettingRepository) *SettingsService {
	return &SettingsService{
		repo:  repo,
		cache: map[string]string{},
	}
}

// Load hydrates the cache from DB. Call once at startup.
func (s *SettingsService) Load(ctx context.Context) error {
	items, err := s.repo.All(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range items {
		s.cache[it.Key] = it.Value
	}
	return nil
}

func (s *SettingsService) get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[key]
}

func (s *SettingsService) set(ctx context.Context, key, value string, updatedBy uint) error {
	if err := s.repo.Upsert(ctx, &model.SystemSetting{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
		UpdatedBy: updatedBy,
	}); err != nil {
		return err
	}
	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()
	return nil
}

// Sub2API returns (baseURL, apiKey). Both may be empty if not configured.
func (s *SettingsService) Sub2API() (baseURL, apiKey string) {
	return s.get(model.SettingKeySub2APIBaseURL), s.get(model.SettingKeySub2APIAPIKey)
}

// Sub2APIConfigured reports whether both fields are set.
func (s *SettingsService) Sub2APIConfigured() bool {
	u, k := s.Sub2API()
	return u != "" && k != ""
}

type Sub2APIUpdate struct {
	BaseURL string
	APIKey  string // empty means "keep existing"
}

// PasswordResetDailyLimit returns the per-operator daily cap on password resets.
// Falls back to DefaultPasswordResetDailyLimit when unset or unparsable.
func (s *SettingsService) PasswordResetDailyLimit() int {
	raw := s.get(model.SettingKeyPasswordResetDailyLimit)
	if raw == "" {
		return DefaultPasswordResetDailyLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return DefaultPasswordResetDailyLimit
	}
	return n
}

func (s *SettingsService) UpdatePasswordResetDailyLimit(ctx context.Context, n int, updatedBy uint) error {
	return s.set(ctx, model.SettingKeyPasswordResetDailyLimit, strconv.Itoa(n), updatedBy)
}

func (s *SettingsService) UpdateSub2API(ctx context.Context, in Sub2APIUpdate, updatedBy uint) error {
	if in.BaseURL != "" {
		if err := s.set(ctx, model.SettingKeySub2APIBaseURL, in.BaseURL, updatedBy); err != nil {
			return err
		}
	}
	if in.APIKey != "" {
		if err := s.set(ctx, model.SettingKeySub2APIAPIKey, in.APIKey, updatedBy); err != nil {
			return err
		}
	}
	return nil
}
