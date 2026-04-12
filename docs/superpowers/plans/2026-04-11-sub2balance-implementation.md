# Sub2Balance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an internal tool for converting Claude monthly subscriptions to balance with staff submission and admin approval workflow.

**Architecture:** Go backend with Gin framework, SQLite database, embedded frontend using Alpine.js, JWT authentication, integration with sub2api via x-api-key.

**Tech Stack:** Go 1.21+, Gin, GORM, SQLite, Alpine.js, TailwindCSS, JWT

---

## File Structure

### Backend
```
cmd/
  server/
    main.go                          # Application entry point
internal/
  config/
    config.go                        # Configuration loading
  model/
    user.go                          # User model
    conversion_request.go            # Conversion request model
    audit_log.go                     # Audit log model
  repository/
    user_repo.go                     # User data access
    conversion_repo.go               # Conversion request data access
    audit_repo.go                    # Audit log data access
  service/
    auth_service.go                  # Authentication logic
    conversion_service.go            # Conversion business logic
    sub2api_client.go                # sub2api API wrapper
    audit_service.go                 # Audit logging
  handler/
    auth_handler.go                  # Auth endpoints
    conversion_handler.go            # Conversion endpoints
    admin_handler.go                 # Admin endpoints
    user_handler.go                  # User management endpoints
  middleware/
    auth.go                          # JWT authentication
    role.go                          # Role-based authorization
    rate_limit.go                    # Rate limiting
  web/
    dist/                            # Embedded frontend files
```

### Frontend
```
web/
  index.html                         # Staff home page
  login.html                         # Login page
  my-requests.html                   # Staff requests list
  admin-pending.html                 # Admin pending requests
  admin-requests.html                # Admin all requests
  admin-users.html                   # Admin user management
  admin-logs.html                    # Admin audit logs
  assets/
    app.js                           # Shared JS utilities
    styles.css                       # TailwindCSS styles
```

### Config & Docs
```
config.yaml.example                  # Example configuration
README.md                            # Project documentation
.gitignore                           # Git ignore rules
go.mod                               # Go dependencies
go.sum                               # Go dependency checksums
```

---

## Task 1: Project Initialization

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `.gitignore`
- Create: `README.md`
- Create: `config.yaml.example`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/mac/git/person/sub2balance
go mod init github.com/yourusername/sub2balance
```

Expected: `go.mod` created

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/gin-gonic/gin@latest
go get gorm.io/gorm@latest
go get gorm.io/driver/sqlite@latest
go get github.com/golang-jwt/jwt/v5@latest
go get golang.org/x/crypto/bcrypt@latest
go get gopkg.in/yaml.v3@latest
go get github.com/ulule/limiter/v3@latest
```

Expected: Dependencies added to `go.mod`

- [ ] **Step 3: Create .gitignore**

```gitignore
# Binaries
sub2balance
*.exe
*.dll
*.so
*.dylib

# Test binary
*.test

# Output
*.out

# Database
*.db
*.db-shm
*.db-wal
data/

# Config
config.yaml

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
```

- [ ] **Step 4: Create README.md**

```markdown
# Sub2Balance

Internal tool for converting Claude monthly subscriptions to balance with approval workflow.

## Features

- Staff can query subscriptions and submit conversion requests
- Admin can review and approve/reject requests
- Automatic balance addition and subscription cancellation via sub2api API
- Audit logging for all operations

## Quick Start

1. Copy config file: `cp config.yaml.example config.yaml`
2. Edit config.yaml with your sub2api credentials
3. Set environment variables:
   ```bash
   export JWT_SECRET="your-secret-here"
   export SUB2API_URL="https://your-sub2api.com"
   export SUB2API_API_KEY="admin-xxxxx"
   ```
4. Run: `go run cmd/server/main.go`
5. Access: `http://localhost:8080`
6. Default admin: `admin@sub2balance.local` / `admin123`

## Build

```bash
go build -o sub2balance cmd/server/main.go
```

## Tech Stack

- Go 1.21+ + Gin + GORM
- SQLite
- Alpine.js + TailwindCSS
- JWT authentication
```

- [ ] **Step 5: Create config.yaml.example**

```yaml
server:
  port: 8080
  mode: release  # debug or release

database:
  path: ./data/sub2balance.db

jwt:
  secret: ${JWT_SECRET}
  expire_hours: 24

sub2api:
  base_url: ${SUB2API_URL}
  api_key: ${SUB2API_API_KEY}
  timeout_seconds: 30
  max_retries: 3

security:
  bcrypt_cost: 12
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

- [ ] **Step 6: Commit**

```bash
git add .
git commit -m "chore: initialize project structure

- Add Go module with dependencies
- Add .gitignore, README, config example
- Set up project foundation

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Configuration Module

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: Write configuration struct**

```go
package config

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Sub2API  Sub2APIConfig  `yaml:"sub2api"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type Sub2APIConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxRetries     int    `yaml:"max_retries"`
}

type SecurityConfig struct {
	BcryptCost int               `yaml:"bcrypt_cost"`
	RateLimit  RateLimitConfig   `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled            bool `yaml:"enabled"`
	RequestsPerMinute  int  `yaml:"requests_per_minute"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT secret is required")
	}
	if cfg.Sub2API.BaseURL == "" {
		return nil, fmt.Errorf("sub2api base URL is required")
	}
	if cfg.Sub2API.APIKey == "" {
		return nil, fmt.Errorf("sub2api API key is required")
	}

	return &cfg, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/config/
git commit -m "feat: add configuration module

- Support YAML config with env var expansion
- Validate required fields
- Define all config structures

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Database Models

**Files:**
- Create: `internal/model/user.go`
- Create: `internal/model/conversion_request.go`
- Create: `internal/model/audit_log.go`

- [ ] **Step 1: Create user model**

```go
package model

import (
	"time"
	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         string         `gorm:"not null;check:role IN ('admin', 'staff')" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
```

- [ ] **Step 2: Create conversion request model**

```go
package model

import (
	"time"
	"gorm.io/gorm"
)

type ConversionRequest struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	UserEmail        string         `gorm:"not null;index" json:"user_email"`
	Sub2APIUserID    int64          `gorm:"not null" json:"sub2api_user_id"`
	SubscriptionID   int64          `gorm:"not null" json:"subscription_id"`
	GroupName        string         `gorm:"not null" json:"group_name"`
	OriginalAmount   float64        `gorm:"not null" json:"original_amount"`
	ConsumedAmount   float64        `gorm:"not null" json:"consumed_amount"`
	ConversionAmount float64        `gorm:"not null" json:"conversion_amount"`
	FinalAmount      *float64       `json:"final_amount"`
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
```

- [ ] **Step 3: Create audit log model**

```go
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
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/
git commit -m "feat: add database models

- User model with role-based access
- ConversionRequest model with workflow states
- AuditLog model for operation tracking

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Database Initialization

**Files:**
- Create: `cmd/server/main.go` (initial version)

- [ ] **Step 1: Create main.go with database setup**

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yourusername/sub2balance/internal/config"
	"github.com/yourusername/sub2balance/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	log.Println("Database initialized successfully")
	log.Printf("Server will run on port %d", cfg.Server.Port)
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	// Create data directory if not exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database
	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.User{},
		&model.ConversionRequest{},
		&model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create default admin if not exists
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), cfg.Security.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		admin := &model.User{
			Email:        "admin@sub2balance.local",
			PasswordHash: string(hash),
			Role:         "admin",
		}

		if err := db.Create(admin).Error; err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Println("Default admin created: admin@sub2balance.local / admin123")
	}

	return db, nil
}
```

- [ ] **Step 2: Test database initialization**

```bash
go run cmd/server/main.go
```

Expected output:
```
Default admin created: admin@sub2balance.local / admin123
Database initialized successfully
Server will run on port 8080
```

- [ ] **Step 3: Verify database file created**

```bash
ls -lh data/sub2balance.db
```

Expected: Database file exists

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add database initialization

- Auto-migrate all models
- Create default admin user
- Set up SQLite connection

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Repository Layer

**Files:**
- Create: `internal/repository/user_repo.go`
- Create: `internal/repository/conversion_repo.go`
- Create: `internal/repository/audit_repo.go`

- [ ] **Step 1: Create user repository**

```go
package repository

import (
	"context"
	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	err := r.db.WithContext(ctx).Find(&users).Error
	return users, err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
```

- [ ] **Step 2: Create conversion repository**

```go
package repository

import (
	"context"
	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type ConversionRepository struct {
	db *gorm.DB
}

func NewConversionRepository(db *gorm.DB) *ConversionRepository {
	return &ConversionRepository{db: db}
}

func (r *ConversionRepository) Create(ctx context.Context, req *model.ConversionRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *ConversionRepository) GetByID(ctx context.Context, id uint) (*model.ConversionRequest, error) {
	var req model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *ConversionRepository) ListBySubmitter(ctx context.Context, userID uint) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Where("submitted_by = ?", userID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) ListByStatus(ctx context.Context, status string) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) ListAll(ctx context.Context) ([]*model.ConversionRequest, error) {
	var requests []*model.ConversionRequest
	err := r.db.WithContext(ctx).
		Preload("SubmittedByUser").
		Preload("ReviewedByUser").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *ConversionRepository) Update(ctx context.Context, req *model.ConversionRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}
```

- [ ] **Step 3: Create audit repository**

```go
package repository

import (
	"context"
	"github.com/yourusername/sub2balance/internal/model"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AuditRepository) List(ctx context.Context, limit, offset int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *AuditRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AuditLog{}).Count(&count).Error
	return count, err
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/
git commit -m "feat: add repository layer

- User repository with CRUD operations
- Conversion repository with status filtering
- Audit repository with pagination

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Sub2API Client

**Files:**
- Create: `internal/service/sub2api_client.go`

- [ ] **Step 1: Write sub2api client**

```go
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/sub2api_client.go
git commit -m "feat: add sub2api client

- HTTP client with retry logic
- User and subscription queries
- Balance addition and subscription cancellation
- Error handling with exponential backoff

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Authentication Service

**Files:**
- Create: `internal/service/auth_service.go`

- [ ] **Step 1: Write auth service**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	userRepo      *repository.UserRepository
	jwtSecret     []byte
	jwtExpireHour int
	bcryptCost    int
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpireHour, bcryptCost int) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtSecret:     []byte(jwtSecret),
		jwtExpireHour: jwtExpireHour,
		bcryptCost:    bcryptCost,
	}
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(s.jwtExpireHour))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/auth_service.go
git commit -m "feat: add authentication service

- JWT token generation and validation
- Password hashing with bcrypt
- Login with email/password

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Audit Service

**Files:**
- Create: `internal/service/audit_service.go`

- [ ] **Step 1: Write audit service**

```go
package service

import (
	"context"
	"encoding/json"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

type AuditService struct {
	auditRepo *repository.AuditRepository
}

func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

func (s *AuditService) Log(ctx context.Context, userID uint, action string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	log := &model.AuditLog{
		UserID:  userID,
		Action:  action,
		Details: string(detailsJSON),
	}

	return s.auditRepo.Create(ctx, log)
}

func (s *AuditService) LogWithRequest(ctx context.Context, userID uint, requestID uint, action string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	log := &model.AuditLog{
		RequestID: &requestID,
		UserID:    userID,
		Action:    action,
		Details:   string(detailsJSON),
	}

	return s.auditRepo.Create(ctx, log)
}

func (s *AuditService) List(ctx context.Context, page, pageSize int) ([]*model.AuditLog, int64, error) {
	offset := (page - 1) * pageSize
	logs, err := s.auditRepo.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.auditRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/audit_service.go
git commit -m "feat: add audit service

- Log operations with JSON details
- Support request-linked logs
- Pagination support

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Conversion Service

**Files:**
- Create: `internal/service/conversion_service.go`

- [ ] **Step 1: Write conversion service**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

var (
	ErrRequestNotFound     = errors.New("conversion request not found")
	ErrInvalidStatus       = errors.New("invalid request status")
	ErrUnauthorized        = errors.New("unauthorized operation")
	ErrSubscriptionInvalid = errors.New("subscription is not valid for conversion")
)

type ConversionService struct {
	convRepo      *repository.ConversionRepository
	sub2apiClient *Sub2APIClient
	auditService  *AuditService
}

func NewConversionService(
	convRepo *repository.ConversionRepository,
	sub2apiClient *Sub2APIClient,
	auditService *AuditService,
) *ConversionService {
	return &ConversionService{
		convRepo:      convRepo,
		sub2apiClient: sub2apiClient,
		auditService:  auditService,
	}
}

type QuerySubscriptionResult struct {
	UserEmail        string  `json:"user_email"`
	Sub2APIUserID    int64   `json:"sub2api_user_id"`
	SubscriptionID   int64   `json:"subscription_id"`
	GroupName        string  `json:"group_name"`
	OriginalAmount   float64 `json:"original_amount"`
	ConsumedAmount   float64 `json:"consumed_amount"`
	ConversionAmount float64 `json:"conversion_amount"`
	Status           string  `json:"status"`
}

func (s *ConversionService) QuerySubscription(ctx context.Context, userID int64, subscriptionID int64) (*QuerySubscriptionResult, error) {
	// Get user info
	user, err := s.sub2apiClient.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get subscription info
	sub, err := s.sub2apiClient.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription is active
	if sub.Status != "active" {
		return nil, ErrSubscriptionInvalid
	}

	// Calculate consumed amount (use monthly as the primary metric)
	consumedAmount := sub.MonthlyUsedUSD

	// For this implementation, we'll use monthly limit as the "original amount"
	// In a real scenario, you'd need to track the actual paid amount
	originalAmount := sub.MonthlyLimitUSD

	// Calculate conversion amount
	conversionAmount := originalAmount - consumedAmount
	if conversionAmount < 0 {
		conversionAmount = 0
	}

	return &QuerySubscriptionResult{
		UserEmail:        user.Email,
		Sub2APIUserID:    user.ID,
		SubscriptionID:   sub.ID,
		GroupName:        sub.GroupName,
		OriginalAmount:   originalAmount,
		ConsumedAmount:   consumedAmount,
		ConversionAmount: conversionAmount,
		Status:           sub.Status,
	}, nil
}

func (s *ConversionService) CreateRequest(ctx context.Context, submitterID uint, query *QuerySubscriptionResult) (*model.ConversionRequest, error) {
	req := &model.ConversionRequest{
		UserEmail:        query.UserEmail,
		Sub2APIUserID:    query.Sub2APIUserID,
		SubscriptionID:   query.SubscriptionID,
		GroupName:        query.GroupName,
		OriginalAmount:   query.OriginalAmount,
		ConsumedAmount:   query.ConsumedAmount,
		ConversionAmount: query.ConversionAmount,
		Status:           "pending",
		SubmittedBy:      submitterID,
	}

	if err := s.convRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, submitterID, req.ID, "create_request", map[string]interface{}{
		"user_email":        req.UserEmail,
		"subscription_id":   req.SubscriptionID,
		"conversion_amount": req.ConversionAmount,
	})

	return req, nil
}

func (s *ConversionService) GetRequest(ctx context.Context, id uint, userID uint, isAdmin bool) (*model.ConversionRequest, error) {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrRequestNotFound
	}

	// Staff can only view their own requests
	if !isAdmin && req.SubmittedBy != userID {
		return nil, ErrUnauthorized
	}

	return req, nil
}

func (s *ConversionService) ListMyRequests(ctx context.Context, userID uint) ([]*model.ConversionRequest, error) {
	return s.convRepo.ListBySubmitter(ctx, userID)
}

func (s *ConversionService) ListAllRequests(ctx context.Context, status string) ([]*model.ConversionRequest, error) {
	if status != "" {
		return s.convRepo.ListByStatus(ctx, status)
	}
	return s.convRepo.ListAll(ctx)
}

func (s *ConversionService) ApproveRequest(ctx context.Context, id uint, reviewerID uint, finalAmount *float64, note string) error {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != "pending" {
		return ErrInvalidStatus
	}

	// Use final amount if provided, otherwise use conversion amount
	amountToAdd := req.ConversionAmount
	if finalAmount != nil {
		amountToAdd = *finalAmount
		req.FinalAmount = finalAmount
	}

	// Add balance to sub2api
	noteText := fmt.Sprintf("Converted from subscription #%d", req.SubscriptionID)
	if note != "" {
		noteText += " - " + note
	}

	if err := s.sub2apiClient.AddBalance(ctx, req.Sub2APIUserID, amountToAdd, noteText); err != nil {
		return fmt.Errorf("failed to add balance: %w", err)
	}

	// Cancel subscription
	if err := s.sub2apiClient.CancelSubscription(ctx, req.SubscriptionID); err != nil {
		// Balance was added but cancellation failed - log this critical error
		_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "cancellation_failed", map[string]interface{}{
			"error":           err.Error(),
			"subscription_id": req.SubscriptionID,
			"balance_added":   amountToAdd,
		})
		return fmt.Errorf("balance added but subscription cancellation failed: %w", err)
	}

	// Update request status
	now := time.Now()
	req.Status = "approved"
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := s.convRepo.Update(ctx, req); err != nil {
		return err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "approve_request", map[string]interface{}{
		"final_amount":    amountToAdd,
		"subscription_id": req.SubscriptionID,
		"note":            note,
	})

	return nil
}

func (s *ConversionService) RejectRequest(ctx context.Context, id uint, reviewerID uint, note string) error {
	req, err := s.convRepo.GetByID(ctx, id)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != "pending" {
		return ErrInvalidStatus
	}

	now := time.Now()
	req.Status = "rejected"
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now

	if err := s.convRepo.Update(ctx, req); err != nil {
		return err
	}

	// Log audit
	_ = s.auditService.LogWithRequest(ctx, reviewerID, req.ID, "reject_request", map[string]interface{}{
		"note": note,
	})

	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/conversion_service.go
git commit -m "feat: add conversion service

- Query subscription from sub2api
- Create conversion requests
- Approve/reject with sub2api integration
- Audit logging for all operations

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Middleware

**Files:**
- Create: `internal/middleware/auth.go`
- Create: `internal/middleware/role.go`
- Create: `internal/middleware/rate_limit.go`

- [ ] **Step 1: Create auth middleware**

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/service"
)

func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
```

- [ ] **Step 2: Create role middleware**

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRole(c)
		if !ok || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 3: Create rate limit middleware**

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old requests
	if times, exists := rl.requests[key]; exists {
		var valid []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		rl.requests[key] = valid
	}

	// Check limit
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// Add new request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	limiter := newRateLimiter(requestsPerMinute)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/middleware/
git commit -m "feat: add middleware

- JWT authentication middleware
- Role-based authorization
- Rate limiting by IP

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: HTTP Handlers

**Files:**
- Create: `internal/handler/auth_handler.go`
- Create: `internal/handler/conversion_handler.go`
- Create: `internal/handler/admin_handler.go`
- Create: `internal/handler/user_handler.go`

- [ ] **Step 1: Create auth handler**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string            `json:"token"`
	User  map[string]interface{} `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT is stateless, logout is handled client-side
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	email, _ := c.Get("user_email")
	role, _ := middleware.GetUserRole(c)

	c.JSON(http.StatusOK, gin.H{
		"id":    userID,
		"email": email,
		"role":  role,
	})
}
```

- [ ] **Step 2: Create conversion handler**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type ConversionHandler struct {
	convService *service.ConversionService
}

func NewConversionHandler(convService *service.ConversionService) *ConversionHandler {
	return &ConversionHandler{
		convService: convService,
	}
}

type QueryRequest struct {
	UserID         int64 `json:"user_id" binding:"required"`
	SubscriptionID int64 `json:"subscription_id" binding:"required"`
}

func (h *ConversionHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.convService.QuerySubscription(c.Request.Context(), req.UserID, req.SubscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type CreateRequest struct {
	UserEmail        string  `json:"user_email" binding:"required"`
	Sub2APIUserID    int64   `json:"sub2api_user_id" binding:"required"`
	SubscriptionID   int64   `json:"subscription_id" binding:"required"`
	GroupName        string  `json:"group_name" binding:"required"`
	OriginalAmount   float64 `json:"original_amount" binding:"required"`
	ConsumedAmount   float64 `json:"consumed_amount" binding:"required"`
	ConversionAmount float64 `json:"conversion_amount" binding:"required"`
}

func (h *ConversionHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)

	query := &service.QuerySubscriptionResult{
		UserEmail:        req.UserEmail,
		Sub2APIUserID:    req.Sub2APIUserID,
		SubscriptionID:   req.SubscriptionID,
		GroupName:        req.GroupName,
		OriginalAmount:   req.OriginalAmount,
		ConsumedAmount:   req.ConsumedAmount,
		ConversionAmount: req.ConversionAmount,
	}

	result, err := h.convService.CreateRequest(c.Request.Context(), userID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *ConversionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := middleware.GetUserID(c)
	role, _ := middleware.GetUserRole(c)
	isAdmin := role == "admin"

	result, err := h.convService.GetRequest(c.Request.Context(), uint(id), userID, isAdmin)
	if err != nil {
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ConversionHandler) ListMy(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	results, err := h.convService.ListMyRequests(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
```

- [ ] **Step 3: Create admin handler**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/service"
)

type AdminHandler struct {
	convService  *service.ConversionService
	auditService *service.AuditService
}

func NewAdminHandler(convService *service.ConversionService, auditService *service.AuditService) *AdminHandler {
	return &AdminHandler{
		convService:  convService,
		auditService: auditService,
	}
}

func (h *AdminHandler) ListRequests(c *gin.Context) {
	status := c.Query("status")

	results, err := h.convService.ListAllRequests(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

type ApproveRequest struct {
	FinalAmount *float64 `json:"final_amount"`
	ReviewNote  string   `json:"review_note"`
}

func (h *AdminHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)

	if err := h.convService.ApproveRequest(c.Request.Context(), uint(id), userID, req.FinalAmount, req.ReviewNote); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request approved successfully"})
}

type RejectRequest struct {
	ReviewNote string `json:"review_note" binding:"required"`
}

func (h *AdminHandler) Reject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)

	if err := h.convService.RejectRequest(c.Request.Context(), uint(id), userID, req.ReviewNote); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected successfully"})
}

func (h *AdminHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	logs, total, err := h.auditService.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
```

- [ ] **Step 4: Create user handler**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
	"github.com/yourusername/sub2balance/internal/service"
)

type UserHandler struct {
	userRepo    *repository.UserRepository
	authService *service.AuthService
}

func NewUserHandler(userRepo *repository.UserRepository, authService *service.AuthService) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		authService: authService,
	}
}

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.userRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin staff"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

type UpdateUserRequest struct {
	Email    string  `json:"email" binding:"omitempty,email"`
	Password *string `json:"password" binding:"omitempty,min=6"`
	Role     string  `json:"role" binding:"omitempty,oneof=admin staff"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != nil {
		hash, err := h.authService.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.PasswordHash = hash
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/handler/
git commit -m "feat: add HTTP handlers

- Auth handler for login/logout
- Conversion handler for staff operations
- Admin handler for review operations
- User handler for user management

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Main Server Setup

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update main.go with complete server setup**

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/sub2balance/internal/config"
	"github.com/yourusername/sub2balance/internal/handler"
	"github.com/yourusername/sub2balance/internal/middleware"
	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
	"github.com/yourusername/sub2balance/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	convRepo := repository.NewConversionRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.Security.BcryptCost)
	auditService := service.NewAuditService(auditRepo)
	sub2apiClient := service.NewSub2APIClient(
		cfg.Sub2API.BaseURL,
		cfg.Sub2API.APIKey,
		time.Duration(cfg.Sub2API.TimeoutSeconds)*time.Second,
		cfg.Sub2API.MaxRetries,
	)
	convService := service.NewConversionService(convRepo, sub2apiClient, auditService)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	convHandler := handler.NewConversionHandler(convService)
	adminHandler := handler.NewAdminHandler(convService, auditService)
	userHandler := handler.NewUserHandler(userRepo, authService)

	// Setup router
	r := gin.Default()

	// Rate limiting
	if cfg.Security.RateLimit.Enabled {
		r.Use(middleware.RateLimitMiddleware(cfg.Security.RateLimit.RequestsPerMinute))
	}

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
		}
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		// Auth
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.Me)

		// Conversions (staff)
		conversions := protected.Group("/conversions")
		{
			conversions.POST("/query", convHandler.Query)
			conversions.POST("", convHandler.Create)
			conversions.GET("", convHandler.ListMy)
			conversions.GET("/:id", convHandler.Get)
		}

		// Admin routes
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			// Conversion management
			admin.GET("/conversions", adminHandler.ListRequests)
			admin.GET("/conversions/:id", convHandler.Get)
			admin.PUT("/conversions/:id/approve", adminHandler.Approve)
			admin.PUT("/conversions/:id/reject", adminHandler.Reject)

			// User management
			admin.GET("/users", userHandler.List)
			admin.POST("/users", userHandler.Create)
			admin.PUT("/users/:id", userHandler.Update)
			admin.DELETE("/users/:id", userHandler.Delete)

			// Audit logs
			admin.GET("/audit-logs", adminHandler.ListLogs)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	// Create data directory if not exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database
	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.User{},
		&model.ConversionRequest{},
		&model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create default admin if not exists
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), cfg.Security.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		admin := &model.User{
			Email:        "admin@sub2balance.local",
			PasswordHash: string(hash),
			Role:         "admin",
		}

		if err := db.Create(admin).Error; err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Println("Default admin created: admin@sub2balance.local / admin123")
	}

	return db, nil
}
```

- [ ] **Step 2: Test server startup**

```bash
# Create config file first
cp config.yaml.example config.yaml

# Set environment variables
export JWT_SECRET="test-secret-key-change-in-production"
export SUB2API_URL="http://localhost:8080"
export SUB2API_API_KEY="admin-test-key"

# Run server
go run cmd/server/main.go
```

Expected output:
```
Default admin created: admin@sub2balance.local / admin123
Server starting on :8080
```

- [ ] **Step 3: Test health endpoint**

```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

- [ ] **Step 4: Test login endpoint**

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@sub2balance.local","password":"admin123"}'
```

Expected: JSON with token and user info

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: complete server setup with all routes

- Initialize all services and handlers
- Set up API routes with middleware
- Add CORS and rate limiting
- Health check endpoint

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Frontend - Base HTML and Assets

**Files:**
- Create: `web/assets/app.js`
- Create: `web/assets/styles.css`
- Create: `web/login.html`

- [ ] **Step 1: Create app.js with utilities**

```javascript
// API base URL
const API_BASE = '';

// Get token from localStorage
function getToken() {
    return localStorage.getItem('token');
}

// Set token to localStorage
function setToken(token) {
    localStorage.setItem('token', token);
}

// Remove token from localStorage
function removeToken() {
    localStorage.removeItem('token');
}

// Get user from localStorage
function getUser() {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
}

// Set user to localStorage
function setUser(user) {
    localStorage.setItem('user', JSON.stringify(user));
}

// Remove user from localStorage
function removeUser() {
    localStorage.removeItem('user');
}

// Check if user is admin
function isAdmin() {
    const user = getUser();
    return user && user.role === 'admin';
}

// API request helper
async function apiRequest(endpoint, options = {}) {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
    };

    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers,
    });

    if (response.status === 401) {
        removeToken();
        removeUser();
        window.location.href = '/login.html';
        throw new Error('Unauthorized');
    }

    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.error || 'Request failed');
    }

    return data;
}

// Logout
async function logout() {
    try {
        await apiRequest('/api/auth/logout', { method: 'POST' });
    } catch (e) {
        // Ignore errors
    }
    removeToken();
    removeUser();
    window.location.href = '/login.html';
}

// Check authentication
function checkAuth() {
    const token = getToken();
    if (!token) {
        window.location.href = '/login.html';
        return false;
    }
    return true;
}

// Format date
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
}

// Format amount
function formatAmount(amount) {
    return `$${amount.toFixed(2)}`;
}

// Show error message
function showError(message) {
    alert(`Error: ${message}`);
}

// Show success message
function showSuccess(message) {
    alert(message);
}
```

- [ ] **Step 2: Create styles.css**

```css
@import url('https://cdn.jsdelivr.net/npm/tailwindcss@3.3.0/dist/tailwind.min.css');

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.card {
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    padding: 20px;
    margin-bottom: 20px;
}

.btn {
    padding: 8px 16px;
    border-radius: 4px;
    border: none;
    cursor: pointer;
    font-size: 14px;
    transition: all 0.2s;
}

.btn-primary {
    background: #3b82f6;
    color: white;
}

.btn-primary:hover {
    background: #2563eb;
}

.btn-success {
    background: #10b981;
    color: white;
}

.btn-success:hover {
    background: #059669;
}

.btn-danger {
    background: #ef4444;
    color: white;
}

.btn-danger:hover {
    background: #dc2626;
}

.form-group {
    margin-bottom: 16px;
}

.form-label {
    display: block;
    margin-bottom: 4px;
    font-weight: 500;
}

.form-input {
    width: 100%;
    padding: 8px 12px;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    font-size: 14px;
}

.form-input:focus {
    outline: none;
    border-color: #3b82f6;
}

.table {
    width: 100%;
    border-collapse: collapse;
}

.table th,
.table td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid #e5e7eb;
}

.table th {
    background: #f9fafb;
    font-weight: 600;
}

.badge {
    display: inline-block;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
}

.badge-pending {
    background: #fef3c7;
    color: #92400e;
}

.badge-approved {
    background: #d1fae5;
    color: #065f46;
}

.badge-rejected {
    background: #fee2e2;
    color: #991b1b;
}

.nav {
    background: #1f2937;
    color: white;
    padding: 16px 0;
    margin-bottom: 20px;
}

.nav-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.nav-links a {
    color: white;
    text-decoration: none;
    margin-left: 20px;
}

.nav-links a:hover {
    text-decoration: underline;
}
```

- [ ] **Step 3: Create login.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - Sub2Balance</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <div class="min-h-screen flex items-center justify-center">
        <div class="card max-w-md w-full">
            <h1 class="text-2xl font-bold mb-6 text-center">Sub2Balance</h1>
            <form id="loginForm" x-data="loginForm()" @submit.prevent="submit">
                <div class="form-group">
                    <label class="form-label">Email</label>
                    <input type="email" class="form-input" x-model="email" required>
                </div>
                <div class="form-group">
                    <label class="form-label">Password</label>
                    <input type="password" class="form-input" x-model="password" required>
                </div>
                <div x-show="error" class="text-red-600 text-sm mb-4" x-text="error"></div>
                <button type="submit" class="btn btn-primary w-full" :disabled="loading">
                    <span x-show="!loading">Login</span>
                    <span x-show="loading">Logging in...</span>
                </button>
            </form>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        function loginForm() {
            return {
                email: '',
                password: '',
                error: '',
                loading: false,

                async submit() {
                    this.error = '';
                    this.loading = true;

                    try {
                        const data = await apiRequest('/api/auth/login', {
                            method: 'POST',
                            body: JSON.stringify({
                                email: this.email,
                                password: this.password,
                            }),
                        });

                        setToken(data.token);
                        setUser(data.user);

                        // Redirect based on role
                        if (data.user.role === 'admin') {
                            window.location.href = '/admin-pending.html';
                        } else {
                            window.location.href = '/index.html';
                        }
                    } catch (e) {
                        this.error = e.message;
                    } finally {
                        this.loading = false;
                    }
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 4: Commit**

```bash
git add web/
git commit -m "feat: add frontend base and login page

- Shared JS utilities for API calls
- TailwindCSS styles
- Login page with Alpine.js

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Frontend - Staff Pages

**Files:**
- Create: `web/index.html`
- Create: `web/my-requests.html`

- [ ] **Step 1: Create index.html (staff home page)**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Create Request - Sub2Balance</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance</div>
            <div class="nav-links">
                <a href="/index.html">Create Request</a>
                <a href="/my-requests.html">My Requests</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="createRequest()">
        <div class="card">
            <h2 class="text-xl font-bold mb-4">Query Subscription</h2>
            <form @submit.prevent="querySubscription">
                <div class="form-group">
                    <label class="form-label">Sub2API User ID</label>
                    <input type="number" class="form-input" x-model="userId" required>
                </div>
                <div class="form-group">
                    <label class="form-label">Subscription ID</label>
                    <input type="number" class="form-input" x-model="subscriptionId" required>
                </div>
                <button type="submit" class="btn btn-primary" :disabled="loading">
                    <span x-show="!loading">Query</span>
                    <span x-show="loading">Querying...</span>
                </button>
            </form>
        </div>

        <div x-show="result" class="card">
            <h2 class="text-xl font-bold mb-4">Subscription Details</h2>
            <div class="space-y-2">
                <div><strong>User Email:</strong> <span x-text="result?.user_email"></span></div>
                <div><strong>Group Name:</strong> <span x-text="result?.group_name"></span></div>
                <div><strong>Original Amount:</strong> <span x-text="formatAmount(result?.original_amount)"></span></div>
                <div><strong>Consumed Amount:</strong> <span x-text="formatAmount(result?.consumed_amount)"></span></div>
                <div><strong>Conversion Amount:</strong> <span class="text-green-600 font-bold" x-text="formatAmount(result?.conversion_amount)"></span></div>
            </div>
            <div class="mt-4">
                <button @click="submitRequest" class="btn btn-success" :disabled="submitting">
                    <span x-show="!submitting">Submit Request</span>
                    <span x-show="submitting">Submitting...</span>
                </button>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth()) {
            throw new Error('Not authenticated');
        }

        function createRequest() {
            return {
                userId: '',
                subscriptionId: '',
                result: null,
                loading: false,
                submitting: false,

                async querySubscription() {
                    this.loading = true;
                    this.result = null;

                    try {
                        this.result = await apiRequest('/api/conversions/query', {
                            method: 'POST',
                            body: JSON.stringify({
                                user_id: parseInt(this.userId),
                                subscription_id: parseInt(this.subscriptionId),
                            }),
                        });
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                async submitRequest() {
                    this.submitting = true;

                    try {
                        await apiRequest('/api/conversions', {
                            method: 'POST',
                            body: JSON.stringify(this.result),
                        });

                        showSuccess('Request submitted successfully');
                        window.location.href = '/my-requests.html';
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.submitting = false;
                    }
                },

                formatAmount(amount) {
                    return amount ? `$${amount.toFixed(2)}` : '$0.00';
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 2: Create my-requests.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>My Requests - Sub2Balance</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance</div>
            <div class="nav-links">
                <a href="/index.html">Create Request</a>
                <a href="/my-requests.html">My Requests</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="myRequests()" x-init="loadRequests()">
        <div class="card">
            <h2 class="text-xl font-bold mb-4">My Requests</h2>
            
            <div x-show="loading" class="text-center py-8">Loading...</div>
            
            <div x-show="!loading && requests.length === 0" class="text-center py-8 text-gray-500">
                No requests found
            </div>

            <div x-show="!loading && requests.length > 0">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>User Email</th>
                            <th>Group</th>
                            <th>Amount</th>
                            <th>Status</th>
                            <th>Created</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <template x-for="req in requests" :key="req.id">
                            <tr>
                                <td x-text="req.id"></td>
                                <td x-text="req.user_email"></td>
                                <td x-text="req.group_name"></td>
                                <td x-text="formatAmount(req.final_amount || req.conversion_amount)"></td>
                                <td>
                                    <span class="badge" :class="getBadgeClass(req.status)" x-text="req.status"></span>
                                </td>
                                <td x-text="formatDate(req.created_at)"></td>
                                <td>
                                    <button @click="viewDetails(req.id)" class="btn btn-primary btn-sm">View</button>
                                </td>
                            </tr>
                        </template>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Details Modal -->
        <div x-show="selectedRequest" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center" @click.self="selectedRequest = null">
            <div class="card max-w-2xl w-full max-h-screen overflow-y-auto">
                <h3 class="text-lg font-bold mb-4">Request Details</h3>
                <template x-if="selectedRequest">
                    <div class="space-y-2">
                        <div><strong>ID:</strong> <span x-text="selectedRequest.id"></span></div>
                        <div><strong>User Email:</strong> <span x-text="selectedRequest.user_email"></span></div>
                        <div><strong>Group:</strong> <span x-text="selectedRequest.group_name"></span></div>
                        <div><strong>Original Amount:</strong> <span x-text="formatAmount(selectedRequest.original_amount)"></span></div>
                        <div><strong>Consumed Amount:</strong> <span x-text="formatAmount(selectedRequest.consumed_amount)"></span></div>
                        <div><strong>Conversion Amount:</strong> <span x-text="formatAmount(selectedRequest.conversion_amount)"></span></div>
                        <div x-show="selectedRequest.final_amount"><strong>Final Amount:</strong> <span x-text="formatAmount(selectedRequest.final_amount)"></span></div>
                        <div><strong>Status:</strong> <span class="badge" :class="getBadgeClass(selectedRequest.status)" x-text="selectedRequest.status"></span></div>
                        <div x-show="selectedRequest.review_note"><strong>Review Note:</strong> <span x-text="selectedRequest.review_note"></span></div>
                        <div><strong>Created:</strong> <span x-text="formatDate(selectedRequest.created_at)"></span></div>
                        <div x-show="selectedRequest.reviewed_at"><strong>Reviewed:</strong> <span x-text="formatDate(selectedRequest.reviewed_at)"></span></div>
                    </div>
                </template>
                <div class="mt-4">
                    <button @click="selectedRequest = null" class="btn btn-primary">Close</button>
                </div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth()) {
            throw new Error('Not authenticated');
        }

        function myRequests() {
            return {
                requests: [],
                selectedRequest: null,
                loading: false,

                async loadRequests() {
                    this.loading = true;
                    try {
                        this.requests = await apiRequest('/api/conversions');
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                async viewDetails(id) {
                    try {
                        this.selectedRequest = await apiRequest(`/api/conversions/${id}`);
                    } catch (e) {
                        showError(e.message);
                    }
                },

                getBadgeClass(status) {
                    return `badge-${status}`;
                },

                formatAmount(amount) {
                    return amount ? `$${amount.toFixed(2)}` : '$0.00';
                },

                formatDate(dateString) {
                    return new Date(dateString).toLocaleString();
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 3: Commit**

```bash
git add web/
git commit -m "feat: add staff pages

- Home page for querying and creating requests
- My requests page with list and details

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: Frontend - Admin Pages

**Files:**
- Create: `web/admin-pending.html`
- Create: `web/admin-requests.html`
- Create: `web/admin-users.html`
- Create: `web/admin-logs.html`

- [ ] **Step 1: Create admin-pending.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pending Requests - Sub2Balance Admin</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance Admin</div>
            <div class="nav-links">
                <a href="/admin-pending.html">Pending</a>
                <a href="/admin-requests.html">All Requests</a>
                <a href="/admin-users.html">Users</a>
                <a href="/admin-logs.html">Logs</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="pendingRequests()" x-init="loadRequests()">
        <div class="card">
            <h2 class="text-xl font-bold mb-4">Pending Requests</h2>
            
            <div x-show="loading" class="text-center py-8">Loading...</div>
            
            <div x-show="!loading && requests.length === 0" class="text-center py-8 text-gray-500">
                No pending requests
            </div>

            <div x-show="!loading && requests.length > 0">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>User Email</th>
                            <th>Group</th>
                            <th>Amount</th>
                            <th>Submitted By</th>
                            <th>Created</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <template x-for="req in requests" :key="req.id">
                            <tr>
                                <td x-text="req.id"></td>
                                <td x-text="req.user_email"></td>
                                <td x-text="req.group_name"></td>
                                <td x-text="formatAmount(req.conversion_amount)"></td>
                                <td x-text="req.submitted_by_user?.email"></td>
                                <td x-text="formatDate(req.created_at)"></td>
                                <td>
                                    <button @click="review(req)" class="btn btn-primary btn-sm">Review</button>
                                </td>
                            </tr>
                        </template>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Review Modal -->
        <div x-show="reviewingRequest" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center" @click.self="reviewingRequest = null">
            <div class="card max-w-2xl w-full max-h-screen overflow-y-auto">
                <h3 class="text-lg font-bold mb-4">Review Request</h3>
                <template x-if="reviewingRequest">
                    <div>
                        <div class="space-y-2 mb-4">
                            <div><strong>User Email:</strong> <span x-text="reviewingRequest.user_email"></span></div>
                            <div><strong>Group:</strong> <span x-text="reviewingRequest.group_name"></span></div>
                            <div><strong>Original Amount:</strong> <span x-text="formatAmount(reviewingRequest.original_amount)"></span></div>
                            <div><strong>Consumed Amount:</strong> <span x-text="formatAmount(reviewingRequest.consumed_amount)"></span></div>
                            <div><strong>Conversion Amount:</strong> <span class="text-green-600 font-bold" x-text="formatAmount(reviewingRequest.conversion_amount)"></span></div>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Final Amount (optional, leave empty to use conversion amount)</label>
                            <input type="number" step="0.01" class="form-input" x-model="finalAmount">
                        </div>

                        <div class="form-group">
                            <label class="form-label">Review Note</label>
                            <textarea class="form-input" rows="3" x-model="reviewNote"></textarea>
                        </div>

                        <div class="flex gap-2">
                            <button @click="approve" class="btn btn-success" :disabled="processing">
                                <span x-show="!processing">Approve</span>
                                <span x-show="processing">Processing...</span>
                            </button>
                            <button @click="reject" class="btn btn-danger" :disabled="processing">
                                <span x-show="!processing">Reject</span>
                                <span x-show="processing">Processing...</span>
                            </button>
                            <button @click="reviewingRequest = null" class="btn">Cancel</button>
                        </div>
                    </div>
                </template>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth() || !isAdmin()) {
            window.location.href = '/index.html';
        }

        function pendingRequests() {
            return {
                requests: [],
                reviewingRequest: null,
                finalAmount: '',
                reviewNote: '',
                loading: false,
                processing: false,

                async loadRequests() {
                    this.loading = true;
                    try {
                        this.requests = await apiRequest('/api/admin/conversions?status=pending');
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                review(req) {
                    this.reviewingRequest = req;
                    this.finalAmount = '';
                    this.reviewNote = '';
                },

                async approve() {
                    if (!confirm('Are you sure you want to approve this request?')) {
                        return;
                    }

                    this.processing = true;
                    try {
                        const body = {
                            review_note: this.reviewNote,
                        };
                        if (this.finalAmount) {
                            body.final_amount = parseFloat(this.finalAmount);
                        }

                        await apiRequest(`/api/admin/conversions/${this.reviewingRequest.id}/approve`, {
                            method: 'PUT',
                            body: JSON.stringify(body),
                        });

                        showSuccess('Request approved successfully');
                        this.reviewingRequest = null;
                        await this.loadRequests();
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.processing = false;
                    }
                },

                async reject() {
                    if (!this.reviewNote) {
                        showError('Review note is required for rejection');
                        return;
                    }

                    if (!confirm('Are you sure you want to reject this request?')) {
                        return;
                    }

                    this.processing = true;
                    try {
                        await apiRequest(`/api/admin/conversions/${this.reviewingRequest.id}/reject`, {
                            method: 'PUT',
                            body: JSON.stringify({
                                review_note: this.reviewNote,
                            }),
                        });

                        showSuccess('Request rejected');
                        this.reviewingRequest = null;
                        await this.loadRequests();
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.processing = false;
                    }
                },

                formatAmount(amount) {
                    return amount ? `$${amount.toFixed(2)}` : '$0.00';
                },

                formatDate(dateString) {
                    return new Date(dateString).toLocaleString();
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 2: Create admin-requests.html (simplified version)**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>All Requests - Sub2Balance Admin</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance Admin</div>
            <div class="nav-links">
                <a href="/admin-pending.html">Pending</a>
                <a href="/admin-requests.html">All Requests</a>
                <a href="/admin-users.html">Users</a>
                <a href="/admin-logs.html">Logs</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="allRequests()" x-init="loadRequests()">
        <div class="card">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-bold">All Requests</h2>
                <select class="form-input w-48" x-model="statusFilter" @change="loadRequests()">
                    <option value="">All Status</option>
                    <option value="pending">Pending</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                </select>
            </div>
            
            <div x-show="loading" class="text-center py-8">Loading...</div>
            
            <div x-show="!loading && requests.length === 0" class="text-center py-8 text-gray-500">
                No requests found
            </div>

            <div x-show="!loading && requests.length > 0">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>User Email</th>
                            <th>Group</th>
                            <th>Amount</th>
                            <th>Status</th>
                            <th>Submitted By</th>
                            <th>Created</th>
                        </tr>
                    </thead>
                    <tbody>
                        <template x-for="req in requests" :key="req.id">
                            <tr>
                                <td x-text="req.id"></td>
                                <td x-text="req.user_email"></td>
                                <td x-text="req.group_name"></td>
                                <td x-text="formatAmount(req.final_amount || req.conversion_amount)"></td>
                                <td>
                                    <span class="badge" :class="getBadgeClass(req.status)" x-text="req.status"></span>
                                </td>
                                <td x-text="req.submitted_by_user?.email"></td>
                                <td x-text="formatDate(req.created_at)"></td>
                            </tr>
                        </template>
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth() || !isAdmin()) {
            window.location.href = '/index.html';
        }

        function allRequests() {
            return {
                requests: [],
                statusFilter: '',
                loading: false,

                async loadRequests() {
                    this.loading = true;
                    try {
                        const url = this.statusFilter 
                            ? `/api/admin/conversions?status=${this.statusFilter}`
                            : '/api/admin/conversions';
                        this.requests = await apiRequest(url);
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                getBadgeClass(status) {
                    return `badge-${status}`;
                },

                formatAmount(amount) {
                    return amount ? `$${amount.toFixed(2)}` : '$0.00';
                },

                formatDate(dateString) {
                    return new Date(dateString).toLocaleString();
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 3: Create admin-users.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Management - Sub2Balance Admin</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance Admin</div>
            <div class="nav-links">
                <a href="/admin-pending.html">Pending</a>
                <a href="/admin-requests.html">All Requests</a>
                <a href="/admin-users.html">Users</a>
                <a href="/admin-logs.html">Logs</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="userManagement()" x-init="loadUsers()">
        <div class="card">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-bold">Users</h2>
                <button @click="showCreateForm = true" class="btn btn-primary">Create User</button>
            </div>
            
            <div x-show="loading" class="text-center py-8">Loading...</div>
            
            <div x-show="!loading">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Email</th>
                            <th>Role</th>
                            <th>Created</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <template x-for="user in users" :key="user.id">
                            <tr>
                                <td x-text="user.id"></td>
                                <td x-text="user.email"></td>
                                <td x-text="user.role"></td>
                                <td x-text="formatDate(user.created_at)"></td>
                                <td>
                                    <button @click="deleteUser(user.id)" class="btn btn-danger btn-sm">Delete</button>
                                </td>
                            </tr>
                        </template>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Create User Modal -->
        <div x-show="showCreateForm" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
            <div class="card max-w-md w-full">
                <h3 class="text-lg font-bold mb-4">Create User</h3>
                <form @submit.prevent="createUser">
                    <div class="form-group">
                        <label class="form-label">Email</label>
                        <input type="email" class="form-input" x-model="newUser.email" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Password</label>
                        <input type="password" class="form-input" x-model="newUser.password" required minlength="6">
                    </div>
                    <div class="form-group">
                        <label class="form-label">Role</label>
                        <select class="form-input" x-model="newUser.role" required>
                            <option value="staff">Staff</option>
                            <option value="admin">Admin</option>
                        </select>
                    </div>
                    <div class="flex gap-2">
                        <button type="submit" class="btn btn-primary" :disabled="creating">
                            <span x-show="!creating">Create</span>
                            <span x-show="creating">Creating...</span>
                        </button>
                        <button type="button" @click="showCreateForm = false" class="btn">Cancel</button>
                    </div>
                </form>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth() || !isAdmin()) {
            window.location.href = '/index.html';
        }

        function userManagement() {
            return {
                users: [],
                showCreateForm: false,
                newUser: { email: '', password: '', role: 'staff' },
                loading: false,
                creating: false,

                async loadUsers() {
                    this.loading = true;
                    try {
                        this.users = await apiRequest('/api/admin/users');
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                async createUser() {
                    this.creating = true;
                    try {
                        await apiRequest('/api/admin/users', {
                            method: 'POST',
                            body: JSON.stringify(this.newUser),
                        });
                        showSuccess('User created successfully');
                        this.showCreateForm = false;
                        this.newUser = { email: '', password: '', role: 'staff' };
                        await this.loadUsers();
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.creating = false;
                    }
                },

                async deleteUser(id) {
                    if (!confirm('Are you sure you want to delete this user?')) {
                        return;
                    }

                    try {
                        await apiRequest(`/api/admin/users/${id}`, { method: 'DELETE' });
                        showSuccess('User deleted successfully');
                        await this.loadUsers();
                    } catch (e) {
                        showError(e.message);
                    }
                },

                formatDate(dateString) {
                    return new Date(dateString).toLocaleString();
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 4: Create admin-logs.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Audit Logs - Sub2Balance Admin</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/app.js"></script>
</head>
<body class="bg-gray-100">
    <nav class="nav">
        <div class="nav-container">
            <div class="text-xl font-bold">Sub2Balance Admin</div>
            <div class="nav-links">
                <a href="/admin-pending.html">Pending</a>
                <a href="/admin-requests.html">All Requests</a>
                <a href="/admin-users.html">Users</a>
                <a href="/admin-logs.html">Logs</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        </div>
    </nav>

    <div class="container" x-data="auditLogs()" x-init="loadLogs()">
        <div class="card">
            <h2 class="text-xl font-bold mb-4">Audit Logs</h2>
            
            <div x-show="loading" class="text-center py-8">Loading...</div>
            
            <div x-show="!loading">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>User</th>
                            <th>Action</th>
                            <th>Request ID</th>
                            <th>Time</th>
                        </tr>
                    </thead>
                    <tbody>
                        <template x-for="log in logs" :key="log.id">
                            <tr>
                                <td x-text="log.id"></td>
                                <td x-text="log.user?.email"></td>
                                <td x-text="log.action"></td>
                                <td x-text="log.request_id || '-'"></td>
                                <td x-text="formatDate(log.created_at)"></td>
                            </tr>
                        </template>
                    </tbody>
                </table>

                <div class="flex justify-between items-center mt-4">
                    <button @click="prevPage" :disabled="page === 1" class="btn btn-primary">Previous</button>
                    <span>Page <span x-text="page"></span></span>
                    <button @click="nextPage" :disabled="logs.length < pageSize" class="btn btn-primary">Next</button>
                </div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        if (!checkAuth() || !isAdmin()) {
            window.location.href = '/index.html';
        }

        function auditLogs() {
            return {
                logs: [],
                page: 1,
                pageSize: 50,
                loading: false,

                async loadLogs() {
                    this.loading = true;
                    try {
                        const data = await apiRequest(`/api/admin/audit-logs?page=${this.page}&page_size=${this.pageSize}`);
                        this.logs = data.logs;
                    } catch (e) {
                        showError(e.message);
                    } finally {
                        this.loading = false;
                    }
                },

                async prevPage() {
                    if (this.page > 1) {
                        this.page--;
                        await this.loadLogs();
                    }
                },

                async nextPage() {
                    this.page++;
                    await this.loadLogs();
                },

                formatDate(dateString) {
                    return new Date(dateString).toLocaleString();
                }
            };
        }
    </script>
</body>
</html>
```

- [ ] **Step 5: Commit**

```bash
git add web/
git commit -m "feat: add admin pages

- Pending requests review page
- All requests list with filtering
- User management page
- Audit logs page with pagination

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Embed Frontend in Binary

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add embed directive and static file serving**

Add at the top of `cmd/server/main.go`:

```go
//go:embed web/*
var webFS embed.FS
```

Add import:
```go
"embed"
"io/fs"
```

Add before `// Start server` in main():

```go
	// Serve static files
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to get web root: %v", err)
	}
	r.StaticFS("/", http.FS(webRoot))
```

- [ ] **Step 2: Build with embed tag**

```bash
go build -tags embed -o sub2balance cmd/server/main.go
```

Expected: Binary created with embedded frontend

- [ ] **Step 3: Test embedded frontend**

```bash
./sub2balance
```

Open browser to `http://localhost:8080/login.html`

Expected: Login page loads

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: embed frontend in binary

- Use go:embed to include web files
- Serve static files from embedded FS
- Single binary deployment

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: Documentation and Deployment

**Files:**
- Modify: `README.md`
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.env.example`

- [ ] **Step 1: Update README.md with complete documentation**

```markdown
# Sub2Balance

Internal tool for converting Claude monthly subscriptions to balance with approval workflow.

## Features

- **Staff Functions**
  - Query subscription info from sub2api by user ID and subscription ID
  - Submit conversion requests
  - View own submission history

- **Admin Functions**
  - Review pending requests
  - Approve/reject with optional amount modification
  - Manage users (create staff/admin accounts)
  - View audit logs

- **System Features**
  - Automatic balance addition via sub2api API
  - Automatic subscription cancellation
  - Complete audit trail
  - JWT authentication
  - Rate limiting

## Tech Stack

- Go 1.21+ + Gin + GORM
- SQLite (single-file database)
- Alpine.js + TailwindCSS (embedded frontend)
- JWT authentication

## Quick Start

### 1. Configuration

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` or set environment variables:

```bash
export JWT_SECRET="your-secret-key-here"
export SUB2API_URL="https://your-sub2api.com"
export SUB2API_API_KEY="admin-xxxxxxxx"
```

### 2. Run

```bash
# From source
go run cmd/server/main.go

# Or build and run
go build -o sub2balance cmd/server/main.go
./sub2balance
```

### 3. Access

- URL: `http://localhost:8080`
- Default admin: `admin@sub2balance.local` / `admin123`
- **Important**: Change the default password after first login

## Build

```bash
# Build binary with embedded frontend
go build -o sub2balance cmd/server/main.go

# The binary includes all frontend files
# Just copy the binary and config.yaml to deploy
```

## Docker Deployment

```bash
# Build image
docker build -t sub2balance .

# Run with docker-compose
docker-compose up -d
```

## Configuration

### config.yaml

```yaml
server:
  port: 8080
  mode: release  # debug or release

database:
  path: ./data/sub2balance.db

jwt:
  secret: ${JWT_SECRET}
  expire_hours: 24

sub2api:
  base_url: ${SUB2API_URL}
  api_key: ${SUB2API_API_KEY}
  timeout_seconds: 30
  max_retries: 3

security:
  bcrypt_cost: 12
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

### Environment Variables

- `JWT_SECRET` (required): Secret key for JWT signing
- `SUB2API_URL` (required): Base URL of your sub2api instance
- `SUB2API_API_KEY` (required): Admin API key from sub2api (format: `admin-<64hex>`)

## Usage

### Staff Workflow

1. Login with staff account
2. Go to "Create Request"
3. Enter sub2api user ID and subscription ID
4. Click "Query" to fetch subscription details
5. Review the conversion amount
6. Click "Submit Request"
7. Check "My Requests" for status updates

### Admin Workflow

1. Login with admin account
2. Go to "Pending" to see pending requests
3. Click "Review" on a request
4. Optionally modify the final amount
5. Add a review note
6. Click "Approve" or "Reject"

### Conversion Formula

```
Conversion Amount = Original Amount - Consumed Amount
```

- Original Amount: Monthly subscription limit (in USD)
- Consumed Amount: Already used amount (in USD)
- Final Amount: Admin can override this during approval

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user

### Conversions (Staff)
- `POST /api/conversions/query` - Query subscription
- `POST /api/conversions` - Create request
- `GET /api/conversions` - List own requests
- `GET /api/conversions/:id` - Get request details

### Admin
- `GET /api/admin/conversions` - List all requests
- `PUT /api/admin/conversions/:id/approve` - Approve
- `PUT /api/admin/conversions/:id/reject` - Reject
- `GET /api/admin/users` - List users
- `POST /api/admin/users` - Create user
- `DELETE /api/admin/users/:id` - Delete user
- `GET /api/admin/audit-logs` - List audit logs

## Security

- JWT authentication with 24-hour expiry
- Password hashing with bcrypt
- Role-based access control (admin/staff)
- Rate limiting (60 requests/minute per IP)
- Audit logging for all operations
- CORS protection

## Backup

SQLite database is a single file at `./data/sub2balance.db`

```bash
# Backup
cp data/sub2balance.db data/sub2balance.db.backup

# Restore
cp data/sub2balance.db.backup data/sub2balance.db
```

## Troubleshooting

### "Failed to load config"
- Ensure `config.yaml` exists
- Check environment variables are set

### "Failed to initialize database"
- Ensure `data/` directory is writable
- Check disk space

### "Invalid or expired token"
- Token expired (24 hours)
- Login again

### Sub2API API errors
- Verify `SUB2API_URL` is correct
- Verify `SUB2API_API_KEY` is valid
- Check sub2api is accessible

## License

MIT
```

- [ ] **Step 2: Create Dockerfile**

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o sub2balance cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/sub2balance .
COPY config.yaml.example config.yaml

RUN mkdir -p data

EXPOSE 8080

CMD ["./sub2balance"]
```

- [ ] **Step 3: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  sub2balance:
    build: .
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - SUB2API_URL=${SUB2API_URL}
      - SUB2API_API_KEY=${SUB2API_API_KEY}
    volumes:
      - ./data:/root/data
    restart: unless-stopped
```

- [ ] **Step 4: Create .env.example**

```bash
# JWT Secret (generate with: openssl rand -hex 32)
JWT_SECRET=your-secret-key-here

# Sub2API Configuration
SUB2API_URL=https://your-sub2api.com
SUB2API_API_KEY=admin-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

- [ ] **Step 5: Commit**

```bash
git add README.md Dockerfile docker-compose.yml .env.example
git commit -m "docs: add complete documentation and deployment files

- Comprehensive README with usage guide
- Dockerfile for containerized deployment
- docker-compose.yml for easy setup
- Environment variable examples

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Final Testing

**Files:**
- None (testing only)

- [ ] **Step 1: Test complete workflow**

```bash
# Start server
export JWT_SECRET="test-secret"
export SUB2API_URL="http://localhost:8080"
export SUB2API_API_KEY="admin-test"
go run cmd/server/main.go
```

- [ ] **Step 2: Test staff workflow**

1. Open `http://localhost:8080/login.html`
2. Login as admin (admin@sub2balance.local / admin123)
3. Go to Users, create a staff account
4. Logout and login as staff
5. Try to query a subscription (will fail without real sub2api)
6. Verify staff cannot access admin pages

- [ ] **Step 3: Test admin workflow**

1. Login as admin
2. Go to Pending requests
3. Verify empty state shows correctly
4. Go to Users
5. Create a test staff user
6. Delete the test user
7. Go to Audit Logs
8. Verify logs show user creation/deletion

- [ ] **Step 4: Test API directly**

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@sub2balance.local","password":"admin123"}' \
  | jq -r '.token')

# Get current user
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/auth/me

# List users (admin only)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/users
```

Expected: All API calls succeed

- [ ] **Step 5: Test rate limiting**

```bash
# Send 70 requests rapidly (should hit rate limit at 60)
for i in {1..70}; do
  curl -s http://localhost:8080/health > /dev/null
  echo "Request $i"
done
```

Expected: Some requests return 429 Too Many Requests

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "test: verify complete system functionality

- Staff and admin workflows tested
- API endpoints verified
- Rate limiting confirmed
- Ready for deployment

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Implementation Complete

All tasks completed. The system is ready for deployment.

### Next Steps

1. Deploy to production server
2. Configure real sub2api credentials
3. Change default admin password
4. Create staff accounts
5. Test with real subscription data

### Deployment Checklist

- [ ] Set strong JWT_SECRET
- [ ] Configure correct SUB2API_URL
- [ ] Set valid SUB2API_API_KEY
- [ ] Change default admin password
- [ ] Set up database backups
- [ ] Configure firewall rules
- [ ] Enable HTTPS (reverse proxy)
- [ ] Monitor audit logs

