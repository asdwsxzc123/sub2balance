package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
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

//go:embed web
var webFS embed.FS

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
			conversions.POST("/query", convHandler.QuerySubscription)
			conversions.POST("", convHandler.CreateRequest)
			conversions.GET("", convHandler.ListMyRequests)
			conversions.GET("/:id", convHandler.GetRequest)
		}

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
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve static files
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to get web root: %v", err)
	}
	r.StaticFS("/", http.FS(webRoot))

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
