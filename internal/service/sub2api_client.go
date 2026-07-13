package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUpstreamUserNotFound indicates no exact email match in the upstream user search.
var ErrUpstreamUserNotFound = errors.New("upstream user not found")

type Sub2APIClient struct {
	settings   *SettingsService
	httpClient *http.Client
	maxRetries int
}

func NewSub2APIClient(settings *SettingsService, timeout time.Duration, maxRetries int) *Sub2APIClient {
	return &Sub2APIClient{
		settings: settings,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

func (c *Sub2APIClient) credentials() (baseURL, apiKey string, err error) {
	baseURL, apiKey = c.settings.Sub2API()
	if baseURL == "" || apiKey == "" {
		return "", "", ErrSub2APINotConfigured
	}
	return strings.TrimRight(baseURL, "/"), apiKey, nil
}

type Sub2APIUser struct {
	ID            int64                 `json:"id"`
	Email         string                `json:"email"`
	Username      string                `json:"username"`
	Role          string                `json:"role"`
	Notes         string                `json:"notes"`
	Concurrency   int                   `json:"concurrency"`
	RPMLimit      int                   `json:"rpm_limit"`
	Status        string                `json:"status"`
	CreatedAt     string                `json:"created_at"`
	Balance       float64               `json:"balance"`
	Subscriptions []Sub2APISubscription `json:"subscriptions,omitempty"`
}

type Sub2APIGroup struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Platform         string   `json:"platform"`
	SubscriptionType string   `json:"subscription_type"`
	Status           string   `json:"status"`
	IsExclusive      bool     `json:"is_exclusive"`
	SortOrder        int      `json:"sort_order"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd"`
}

type Sub2APISubscription struct {
	ID              int64         `json:"id"`
	UserID          int64         `json:"user_id"`
	GroupID         int64         `json:"group_id"`
	Status          string        `json:"status"`
	DailyUsageUSD   float64       `json:"daily_usage_usd"`
	WeeklyUsageUSD  float64       `json:"weekly_usage_usd"`
	MonthlyUsageUSD float64       `json:"monthly_usage_usd"`
	StartsAt        string        `json:"starts_at"`
	ExpiresAt       string        `json:"expires_at"`
	User            *Sub2APIUser  `json:"user"`
	Group           *Sub2APIGroup `json:"group"`
}

type sub2apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type sub2apiList[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int   `json:"pages"`
}

func (c *Sub2APIClient) SearchUserByEmail(ctx context.Context, email string) (*Sub2APIUser, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf(
		"%s/api/v1/admin/users?page=1&page_size=20&search=%s&include_subscriptions=true&sort_by=created_at&sort_order=desc",
		baseURL, url.QueryEscape(email),
	)

	var list sub2apiList[Sub2APIUser]
	if err := c.doRequest(ctx, "GET", endpoint, nil, &list); err != nil {
		return nil, err
	}

	for i := range list.Items {
		if list.Items[i].Email == email {
			return &list.Items[i], nil
		}
	}
	return nil, fmt.Errorf("%w: no user found for email %q", ErrUpstreamUserNotFound, email)
}

func (c *Sub2APIClient) GetUser(ctx context.Context, userID int64) (*Sub2APIUser, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/users/%d", baseURL, userID)

	var user Sub2APIUser
	err = c.doRequest(ctx, "GET", endpoint, nil, &user)
	return &user, err
}

func (c *Sub2APIClient) GetSubscription(ctx context.Context, subscriptionID int64) (*Sub2APISubscription, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/subscriptions/%d", baseURL, subscriptionID)

	var sub Sub2APISubscription
	err = c.doRequest(ctx, "GET", endpoint, nil, &sub)
	return &sub, err
}

func (c *Sub2APIClient) ListSubscriptionsByUser(ctx context.Context, userID int64) ([]Sub2APISubscription, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/subscriptions?user_id=%d&page=1&page_size=100", baseURL, userID)

	var list sub2apiList[Sub2APISubscription]
	if err := c.doRequest(ctx, "GET", endpoint, nil, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Sub2APIClient) AddBalance(ctx context.Context, userID int64, amount float64, note string) error {
	baseURL, _, err := c.credentials()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/users/%d/balance", baseURL, userID)

	body := map[string]interface{}{
		"balance":   amount,
		"operation": "add",
		"notes":     note,
	}

	return c.doRequest(ctx, "POST", endpoint, body, nil)
}

// UpdateUserPassword resets a user's password via the upstream full-update endpoint.
// The upstream PUT replaces the whole user record, so all existing fields must be
// sent back unchanged to avoid wiping the user's configuration.
func (c *Sub2APIClient) UpdateUserPassword(ctx context.Context, user *Sub2APIUser, newPassword string) error {
	baseURL, _, err := c.credentials()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/users/%d", baseURL, user.ID)

	body := map[string]interface{}{
		"email":       user.Email,
		"username":    user.Username,
		"notes":       user.Notes,
		"role":        user.Role,
		"concurrency": user.Concurrency,
		"rpm_limit":   user.RPMLimit,
		"password":    newPassword,
	}

	return c.doRequest(ctx, "PUT", endpoint, body, nil)
}

func (c *Sub2APIClient) CancelSubscription(ctx context.Context, subscriptionID int64) error {
	baseURL, _, err := c.credentials()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/subscriptions/%d", baseURL, subscriptionID)
	return c.doRequest(ctx, "DELETE", endpoint, nil, nil)
}

func (c *Sub2APIClient) ListGroups(ctx context.Context) ([]Sub2APIGroup, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf(
		"%s/api/v1/admin/groups?page=1&page_size=100&sort_by=sort_order&sort_order=asc",
		baseURL,
	)

	var list sub2apiList[Sub2APIGroup]
	if err := c.doRequest(ctx, "GET", endpoint, nil, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Sub2APIClient) AssignSubscription(ctx context.Context, userID, groupID int64, validityDays int) (*Sub2APISubscription, error) {
	baseURL, _, err := c.credentials()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/subscriptions/assign", baseURL)

	body := map[string]interface{}{
		"user_id":       userID,
		"group_id":      groupID,
		"validity_days": validityDays,
	}

	var sub Sub2APISubscription
	if err := c.doRequest(ctx, "POST", endpoint, body, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// TestConnection sends a lightweight probe with the provided credentials.
// Used by admin settings page to validate before/after save.
func (c *Sub2APIClient) TestConnection(ctx context.Context, baseURL, apiKey string) error {
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := fmt.Sprintf("%s/api/v1/admin/groups?page=1&page_size=1", baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("upstream status %d: %s", resp.StatusCode, string(body))
}

func (c *Sub2APIClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	_, apiKey, err := c.credentials()
	if err != nil {
		return err
	}
	var bodyData []byte
	if body != nil {
		bodyData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second * time.Duration(attempt))
		}

		// Build a fresh body reader per attempt: a shared reader is exhausted
		// after the first attempt, making every retry send an empty body.
		var reqBody io.Reader
		if bodyData != nil {
			reqBody = bytes.NewReader(bodyData)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if len(respBody) == 0 {
				return nil
			}
			var envelope sub2apiEnvelope
			if err := json.Unmarshal(respBody, &envelope); err != nil {
				return fmt.Errorf("failed to parse response envelope: %w", err)
			}
			if envelope.Code != 0 {
				return fmt.Errorf("upstream API error (code %d): %s", envelope.Code, envelope.Message)
			}
			if result == nil || len(envelope.Data) == 0 {
				return nil
			}
			if err := json.Unmarshal(envelope.Data, result); err != nil {
				return fmt.Errorf("failed to parse response data: %w", err)
			}
			return nil
		}

		lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}
