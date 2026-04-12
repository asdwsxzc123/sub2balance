package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Sub2APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

func NewSub2APIClient(baseURL, apiKey string, timeout time.Duration, maxRetries int) *Sub2APIClient {
	return &Sub2APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

type Sub2APIUser struct {
	ID      int64   `json:"id"`
	Email   string  `json:"email"`
	Balance float64 `json:"balance"`
}

type Sub2APISubscription struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	GroupID         int64   `json:"group_id"`
	GroupName       string  `json:"group_name"`
	Status          string  `json:"status"`
	DailyUsedUSD    float64 `json:"daily_used_usd"`
	WeeklyUsedUSD   float64 `json:"weekly_used_usd"`
	MonthlyUsedUSD  float64 `json:"monthly_used_usd"`
	DailyLimitUSD   float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
}

func (c *Sub2APIClient) GetUserByEmail(ctx context.Context, email string) (*Sub2APIUser, error) {
	// Note: sub2api doesn't have direct email lookup, need to implement search
	// For now, return error indicating manual lookup needed
	return nil, fmt.Errorf("email lookup not implemented - use user ID directly")
}

func (c *Sub2APIClient) GetUser(ctx context.Context, userID int64) (*Sub2APIUser, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d", c.baseURL, userID)

	var user Sub2APIUser
	err := c.doRequest(ctx, "GET", url, nil, &user)
	return &user, err
}

func (c *Sub2APIClient) GetSubscription(ctx context.Context, subscriptionID int64) (*Sub2APISubscription, error) {
	url := fmt.Sprintf("%s/api/v1/admin/subscriptions/%d", c.baseURL, subscriptionID)

	var sub Sub2APISubscription
	err := c.doRequest(ctx, "GET", url, nil, &sub)
	return &sub, err
}

func (c *Sub2APIClient) AddBalance(ctx context.Context, userID int64, amount float64, note string) error {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d/balance", c.baseURL, userID)

	body := map[string]interface{}{
		"balance":   amount,
		"operation": "add",
		"notes":     note,
	}

	return c.doRequest(ctx, "POST", url, body, nil)
}

func (c *Sub2APIClient) CancelSubscription(ctx context.Context, subscriptionID int64) error {
	url := fmt.Sprintf("%s/api/v1/admin/subscriptions/%d", c.baseURL, subscriptionID)
	return c.doRequest(ctx, "DELETE", url, nil, nil)
}

func (c *Sub2APIClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second * time.Duration(attempt))
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
			}
			return nil
		}

		lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))

		// Don't retry on client errors
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}
