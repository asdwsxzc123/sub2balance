package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"
)

var (
	ErrAdminAccountForbidden = errors.New("该账号不允许操作")
	ErrAccountNotFound       = errors.New("账号不存在")
	ErrDailyLimitExceeded    = errors.New("今日密码重置次数已达上限")
)

const (
	passwordLength   = 16
	uppercaseChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseChars   = "abcdefghijklmnopqrstuvwxyz"
	digitChars       = "0123456789"
	specialChars     = "!@#$%^&*"
	allPasswordChars = uppercaseChars + lowercaseChars + digitChars + specialChars
)

type PasswordResetService struct {
	sub2apiClient   *Sub2APIClient
	auditService    *AuditService
	settingsService *SettingsService
	// resetLocks serializes ResetPassword per operator: the daily-limit check
	// counts audit rows written at the end of the same flow, so without this
	// lock concurrent requests could all pass the check before any row lands.
	resetLocks sync.Map // operatorID (uint) -> *sync.Mutex
}

func NewPasswordResetService(sub2apiClient *Sub2APIClient, auditService *AuditService, settingsService *SettingsService) *PasswordResetService {
	return &PasswordResetService{
		sub2apiClient:   sub2apiClient,
		auditService:    auditService,
		settingsService: settingsService,
	}
}

// QueryAccount looks up an upstream account by exact email match, rejecting
// admin-looking emails before hitting upstream and admin-role users afterwards.
// The role check runs against the authoritative user-detail record (list
// endpoints may omit the role field) and fails closed when role is missing.
// Every query outcome is audited (never including any password material).
func (s *PasswordResetService) QueryAccount(ctx context.Context, operatorID uint, email string) (*Sub2APIUser, error) {
	if strings.Contains(strings.ToLower(email), "admin") {
		s.logQuery(ctx, operatorID, email, "denied")
		return nil, ErrAdminAccountForbidden
	}

	match, err := s.sub2apiClient.SearchUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUpstreamUserNotFound) {
			s.logQuery(ctx, operatorID, email, "not_found")
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Do not trust the (possibly abbreviated) search result: fetch the full
	// record so the admin guard sees the authoritative role, and so downstream
	// full-replace updates send the complete field set.
	user, err := s.sub2apiClient.GetUser(ctx, match.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user detail: %w", err)
	}

	if user.Role == "" || user.Role == "admin" {
		s.logQuery(ctx, operatorID, email, "denied")
		return nil, ErrAdminAccountForbidden
	}

	s.logQuery(ctx, operatorID, email, "found")
	user.Subscriptions = nil
	return user, nil
}

func (s *PasswordResetService) logQuery(ctx context.Context, operatorID uint, email, result string) {
	_ = s.auditService.Log(ctx, operatorID, "password_reset_query", map[string]interface{}{
		"email":  email,
		"result": result,
	})
}

func (s *PasswordResetService) logResetDenied(ctx context.Context, operatorID uint, email, reason string) {
	_ = s.auditService.Log(ctx, operatorID, "reset_password_denied", map[string]interface{}{
		"email":  email,
		"reason": reason,
	})
}

// ResetPassword resets the upstream account password to a freshly generated one.
// The new password is returned to the caller and never written to audit logs.
// Each operator is capped at a configurable number of resets per local calendar
// day, counted from successful "reset_password" audit entries.
func (s *PasswordResetService) ResetPassword(ctx context.Context, operatorID uint, email string) (string, *Sub2APIUser, error) {
	lockAny, _ := s.resetLocks.LoadOrStore(operatorID, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	limit := s.settingsService.PasswordResetDailyLimit()
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := s.auditService.CountByUserActionSince(ctx, operatorID, "reset_password", startOfDay)
	if err != nil {
		return "", nil, fmt.Errorf("failed to count today's password resets: %w", err)
	}
	if count >= int64(limit) {
		s.logResetDenied(ctx, operatorID, email, "daily_limit_exceeded")
		return "", nil, fmt.Errorf("%w（%d 次）", ErrDailyLimitExceeded, limit)
	}

	user, err := s.QueryAccount(ctx, operatorID, email)
	if err != nil {
		switch {
		case errors.Is(err, ErrAdminAccountForbidden):
			s.logResetDenied(ctx, operatorID, email, "admin_forbidden")
		case errors.Is(err, ErrAccountNotFound):
			s.logResetDenied(ctx, operatorID, email, "not_found")
		}
		return "", nil, err
	}

	newPassword, err := generatePassword()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate password: %w", err)
	}

	if err := s.sub2apiClient.UpdateUserPassword(ctx, user, newPassword); err != nil {
		return "", nil, fmt.Errorf("failed to update password: %w", err)
	}

	// The audit row is also the daily-limit counter: a lost write means a reset
	// that never counts against the quota, so surface the failure loudly.
	if err := s.auditService.Log(ctx, operatorID, "reset_password", map[string]interface{}{
		"email":           user.Email,
		"sub2api_user_id": user.ID,
	}); err != nil {
		log.Printf("ERROR: reset_password audit write failed (operator=%d, email=%s): %v — reset succeeded but will not count toward the daily limit", operatorID, user.Email, err)
	}

	return newPassword, user, nil
}

// generatePassword builds a 16-char password containing at least one uppercase
// letter, one lowercase letter, one digit and one special character, using
// crypto/rand for both selection and shuffling.
func generatePassword() (string, error) {
	classes := []string{uppercaseChars, lowercaseChars, digitChars, specialChars}

	chars := make([]byte, 0, passwordLength)
	for _, class := range classes {
		ch, err := randomChar(class)
		if err != nil {
			return "", err
		}
		chars = append(chars, ch)
	}
	for len(chars) < passwordLength {
		ch, err := randomChar(allPasswordChars)
		if err != nil {
			return "", err
		}
		chars = append(chars, ch)
	}

	// Fisher-Yates shuffle driven by crypto/rand.
	for i := len(chars) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := int(n.Int64())
		chars[i], chars[j] = chars[j], chars[i]
	}

	return string(chars), nil
}

func randomChar(charset string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, err
	}
	return charset[n.Int64()], nil
}
