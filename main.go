package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
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

//go:embed all:frontend/dist
var webFS embed.FS

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// Handle --version before anything else (must not depend on config.yaml):
	// the self-upgrade flow runs "{new binary} --version" as a sanity check.
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		os.Exit(0)
	}

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
	groupPriceRepo := repository.NewGroupPriceRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.Security.BcryptCost)
	auditService := service.NewAuditService(auditRepo)
	settingsService := service.NewSettingsService(settingRepo)
	if err := settingsService.Load(context.Background()); err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}
	sub2apiClient := service.NewSub2APIClient(
		settingsService,
		time.Duration(cfg.Sub2API.TimeoutSeconds)*time.Second,
		cfg.Sub2API.MaxRetries,
	)
	if !settingsService.Sub2APIConfigured() {
		log.Println("WARNING: sub2api is not configured yet — log in as admin and set it at /admin/settings before using conversion features")
	}
	convService := service.NewConversionService(convRepo, groupPriceRepo, sub2apiClient, auditService)
	groupPriceService := service.NewGroupPriceService(groupPriceRepo, sub2apiClient, auditService)
	passwordResetService := service.NewPasswordResetService(sub2apiClient, auditService, settingsService)
	upgradeService := service.NewUpgradeService(version, auditService)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	convHandler := handler.NewConversionHandler(convService)
	adminHandler := handler.NewAdminHandler(convService, auditService)
	userHandler := handler.NewUserHandler(userRepo, authService)
	groupPriceHandler := handler.NewGroupPriceHandler(groupPriceService)
	settingsHandler := handler.NewSettingsHandler(settingsService, sub2apiClient, auditService)
	passwordResetHandler := handler.NewPasswordResetHandler(passwordResetService)
	upgradeHandler := handler.NewUpgradeHandler(upgradeService)

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
		protected.POST("/auth/change-password", authHandler.ChangePassword)

		// Conversions (staff)
		conversions := protected.Group("/conversions")
		{
			conversions.POST("/query", convHandler.QuerySubscription)
			conversions.POST("/query-by-email", convHandler.QueryByEmail)
			conversions.POST("", convHandler.CreateRequest)
			conversions.GET("", convHandler.ListMyRequests)
			conversions.GET("/:id", convHandler.GetRequest)
		}

		// Subscription groups (staff + admin)
		protected.GET("/groups", convHandler.ListGroups)

		// Password reset for upstream accounts (staff + admin)
		protected.POST("/password-reset/query", passwordResetHandler.Query)
		protected.POST("/password-reset", passwordResetHandler.Reset)

		// Admin routes
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			// Conversion management
			admin.GET("/conversions", adminHandler.ListAllRequests)
			admin.GET("/conversions/:id", convHandler.GetRequest)
			admin.PUT("/conversions/:id/approve", adminHandler.ApproveRequest)
			admin.PUT("/conversions/:id/reject", adminHandler.RejectRequest)

			// User management
			admin.GET("/users", userHandler.ListUsers)
			admin.POST("/users", userHandler.CreateUser)
			admin.PUT("/users/:id", userHandler.UpdateUser)
			admin.DELETE("/users/:id", userHandler.DeleteUser)

			// Audit logs
			admin.GET("/audit-logs", adminHandler.ListAuditLogs)

			// Group-to-price mapping
			admin.GET("/group-prices", groupPriceHandler.List)
			admin.PUT("/group-prices/:group_id", groupPriceHandler.Upsert)
			admin.DELETE("/group-prices/:group_id", groupPriceHandler.Delete)

			// System settings (Sub2API upstream credentials)
			admin.GET("/settings/sub2api", settingsHandler.GetSub2API)
			admin.PUT("/settings/sub2api", settingsHandler.UpdateSub2API)
			admin.POST("/settings/sub2api/test", settingsHandler.TestSub2API)

			// System settings (password reset daily limit)
			admin.GET("/settings/password-reset", settingsHandler.GetPasswordReset)
			admin.PUT("/settings/password-reset", settingsHandler.UpdatePasswordReset)

			// System version & self-upgrade
			admin.GET("/system/version", upgradeHandler.GetVersion)
			admin.GET("/system/latest", upgradeHandler.GetLatest)
			admin.POST("/system/upgrade", upgradeHandler.Upgrade)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve SPA (React build) via NoRoute — falls back to index.html for client-side routes.
	webRoot, err := fs.Sub(webFS, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to get web root: %v", err)
	}
	fileServer := http.FileServer(http.FS(webRoot))
	indexHTML, err := fs.ReadFile(webRoot, "index.html")
	if err != nil {
		log.Fatalf("Failed to read index.html: %v", err)
	}
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p != "/" && p != "/index.html" {
			if _, statErr := fs.Stat(webRoot, p[1:]); statErr != nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
				return
			}
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
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

	// SQLite CHECK constraints are baked into the table DDL and are not rebuilt by
	// AutoMigrate. If the deployed table still restricts request_type to
	// ('balance','switch'), rename it aside so AutoMigrate recreates it, then copy
	// the rows back.
	if err := migrateConversionRequestTypeConstraint(db); err != nil {
		return nil, fmt.Errorf("failed to migrate conversion_requests constraint: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.User{},
		&model.ConversionRequest{},
		&model.AuditLog{},
		&model.GroupPrice{},
		&model.SystemSetting{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	if err := restoreConversionRequestRows(db); err != nil {
		return nil, fmt.Errorf("failed to restore conversion_requests rows: %w", err)
	}

	// Create default admin if not exists
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.Password), cfg.Security.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		admin := &model.User{
			Email:        cfg.Admin.Email,
			PasswordHash: string(hash),
			Role:         "admin",
		}

		if err := db.Create(admin).Error; err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Printf("Default admin created: %s", cfg.Admin.Email)
	}

	return db, nil
}

// migrateConversionRequestTypeConstraint renames conversion_requests aside when
// its CHECK constraint predates the 'bind'/'unbind' request types, so AutoMigrate
// can recreate the table with the new constraint.
func migrateConversionRequestTypeConstraint(db *gorm.DB) error {
	// Heal deployments stuck from a previous half-finished migration: the renamed
	// table may still hold indexes under their original global names, which blocks
	// AutoMigrate's CREATE INDEX on the new table.
	if err := dropSQLiteTableIndexes(db, "conversion_requests_old"); err != nil {
		return err
	}

	var tableSQL string
	err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='conversion_requests'").Scan(&tableSQL).Error
	if err != nil {
		return fmt.Errorf("failed to read table definition: %w", err)
	}
	if tableSQL == "" || strings.Contains(tableSQL, "bind") {
		return nil
	}

	if err := db.Exec("ALTER TABLE conversion_requests RENAME TO conversion_requests_old").Error; err != nil {
		return fmt.Errorf("failed to rename old table: %w", err)
	}
	// Renaming a table keeps its indexes attached under their original names.
	// Index names are database-global in SQLite, so drop them now or AutoMigrate's
	// CREATE INDEX on the rebuilt table fails with "index ... already exists".
	if err := dropSQLiteTableIndexes(db, "conversion_requests_old"); err != nil {
		return err
	}
	log.Println("conversion_requests: outdated request_type constraint detected, table will be rebuilt")
	return nil
}

// dropSQLiteTableIndexes drops all explicitly created indexes attached to the
// given table. Auto-created indexes (sqlite_autoindex_*) back UNIQUE/PK
// constraints and cannot (and need not) be dropped.
func dropSQLiteTableIndexes(db *gorm.DB, table string) error {
	var names []string
	err := db.Raw(
		"SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=? AND name NOT LIKE 'sqlite_autoindex%'",
		table,
	).Scan(&names).Error
	if err != nil {
		return fmt.Errorf("failed to list indexes of %s: %w", table, err)
	}
	for _, name := range names {
		if err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %q", name)).Error; err != nil {
			return fmt.Errorf("failed to drop index %s: %w", name, err)
		}
	}
	return nil
}

// restoreConversionRequestRows copies rows from conversion_requests_old (created by
// migrateConversionRequestTypeConstraint) into the freshly migrated table using the
// intersection of both tables' columns, then drops the old table.
func restoreConversionRequestRows(db *gorm.DB) error {
	var oldTable string
	err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name='conversion_requests_old'").Scan(&oldTable).Error
	if err != nil {
		return fmt.Errorf("failed to check for old table: %w", err)
	}
	if oldTable == "" {
		return nil
	}

	newCols, err := sqliteTableColumns(db, "conversion_requests")
	if err != nil {
		return err
	}
	oldCols, err := sqliteTableColumns(db, "conversion_requests_old")
	if err != nil {
		return err
	}

	newColSet := make(map[string]bool, len(newCols))
	for _, col := range newCols {
		newColSet[col] = true
	}
	common := make([]string, 0, len(oldCols))
	for _, col := range oldCols {
		if newColSet[col] {
			common = append(common, col)
		}
	}
	if len(common) == 0 {
		return fmt.Errorf("no common columns between conversion_requests and conversion_requests_old")
	}

	colList := strings.Join(common, ", ")
	copySQL := fmt.Sprintf(
		"INSERT INTO conversion_requests (%s) SELECT %s FROM conversion_requests_old",
		colList, colList,
	)
	// Copy and drop atomically so a failure mid-way never loses the old rows.
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(copySQL).Error; err != nil {
			return fmt.Errorf("failed to copy rows from old table: %w", err)
		}
		if err := tx.Exec("DROP TABLE conversion_requests_old").Error; err != nil {
			return fmt.Errorf("failed to drop old table: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	log.Println("conversion_requests: table rebuilt with updated request_type constraint, rows restored")
	return nil
}

func sqliteTableColumns(db *gorm.DB, table string) ([]string, error) {
	rows, err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", table)).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to read columns of %s: %w", table, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue interface{}
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column info of %s: %w", table, err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate columns of %s: %w", table, err)
	}
	return cols, nil
}
