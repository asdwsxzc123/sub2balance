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
	Admin    AdminConfig    `yaml:"admin"`
}

type AdminConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
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

// Sub2APIConfig only carries HTTP client tuning. base_url and api_key are now
// stored in the system_settings table and managed via the admin UI.
type Sub2APIConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxRetries     int `yaml:"max_retries"`
}

type SecurityConfig struct {
	BcryptCost int             `yaml:"bcrypt_cost"`
	RateLimit  RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT secret is required")
	}
	if cfg.Admin.Email == "" {
		return nil, fmt.Errorf("admin email is required")
	}
	if cfg.Admin.Password == "" {
		return nil, fmt.Errorf("admin password is required")
	}

	if cfg.Sub2API.TimeoutSeconds <= 0 {
		cfg.Sub2API.TimeoutSeconds = 30
	}
	if cfg.Sub2API.MaxRetries < 0 {
		cfg.Sub2API.MaxRetries = 0
	}

	return &cfg, nil
}
