package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Scanner  ScannerConfig  `yaml:"scanner"`
	Vibes    VibesConfig    `yaml:"vibes"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	Development  bool          `yaml:"development"`
	CORS         CORSConfig    `yaml:"cors"`
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool          `yaml:"enabled"`
	RequestsPerMinute int     `yaml:"requests_per_minute"`
	BurstSize   int           `yaml:"burst_size"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// ScannerConfig contains file scanning configuration
type ScannerConfig struct {
	MaxFileSize      int64    `yaml:"max_file_size"`
	MaxFiles         int      `yaml:"max_files"`
	IgnorePatterns   []string `yaml:"ignore_patterns"`
	IncludePatterns  []string `yaml:"include_patterns"`
	MaxDepth         int      `yaml:"max_depth"`
	FollowSymlinks   bool     `yaml:"follow_symlinks"`
	ScanTimeout      time.Duration `yaml:"scan_timeout"`
	ConcurrentWorkers int     `yaml:"concurrent_workers"`
}

// VibesConfig contains vibe checker configuration
type VibesConfig struct {
	EnabledCheckers  []string                   `yaml:"enabled_checkers"`
	CheckerConfigs   map[string]CheckerConfig   `yaml:"checker_configs"`
	GlobalSettings   map[string]interface{}     `yaml:"global_settings"`
	OutputFormat     string                     `yaml:"output_format"`
	MinConfidence    float64                    `yaml:"min_confidence"`
	MaxIssues        int                        `yaml:"max_issues"`
}

// CheckerConfig contains configuration for individual checkers
type CheckerConfig struct {
	Enabled   bool                   `yaml:"enabled"`
	Settings  map[string]interface{} `yaml:"settings"`
	Severity  string                 `yaml:"severity"`
	Priority  int                    `yaml:"priority"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Type           string        `yaml:"type"`
	ConnectionString string      `yaml:"connection_string"`
	MaxConnections int           `yaml:"max_connections"`
	MaxIdleTime    time.Duration `yaml:"max_idle_time"`
	MaxLifetime    time.Duration `yaml:"max_lifetime"`
	EnableMetrics  bool          `yaml:"enable_metrics"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	Structured bool   `yaml:"structured"`
	FileRotation struct {
		Enabled    bool   `yaml:"enabled"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"file_rotation"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EnableHTTPS     bool   `yaml:"enable_https"`
	CertFile        string `yaml:"cert_file"`
	KeyFile         string `yaml:"key_file"`
	EnableHSTS      bool   `yaml:"enable_hsts"`
	EnableCSRF      bool   `yaml:"enable_csrf"`
	JWTSecret       string `yaml:"jwt_secret"`
	APIKeyHeader    string `yaml:"api_key_header"`
	TrustedProxies  []string `yaml:"trusted_proxies"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			Development:  false,
			CORS: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
				ExposedHeaders: []string{"X-Total-Count"},
				AllowCredentials: false,
				MaxAge:         3600,
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
				BurstSize:         10,
				CleanupInterval:   time.Minute,
			},
		},
		Scanner: ScannerConfig{
			MaxFileSize:       50 * 1024 * 1024, // 50MB
			MaxFiles:          10000,
			IgnorePatterns:    []string{".git", "node_modules", ".DS_Store", "*.log", "*.tmp"},
			IncludePatterns:   []string{"*.go", "*.js", "*.ts", "*.py", "*.java", "*.rs", "*.cpp", "*.c", "*.h"},
			MaxDepth:          10,
			FollowSymlinks:    false,
			ScanTimeout:       5 * time.Minute,
			ConcurrentWorkers: 4,
		},
		Vibes: VibesConfig{
			EnabledCheckers: []string{"code", "security", "performance", "file", "git", "dependency", "documentation"},
			CheckerConfigs: map[string]CheckerConfig{
				"code": {
					Enabled:  true,
					Settings: map[string]interface{}{
						"enable_complexity": true,
						"enable_duplication": true,
						"max_function_lines": 50,
						"max_complexity": 10,
					},
					Severity: "warning",
					Priority: 1,
				},
				"security": {
					Enabled:  true,
					Settings: map[string]interface{}{
						"enable_sql_injection": true,
						"enable_xss": true,
						"enable_hardcoded_secrets": true,
					},
					Severity: "error",
					Priority: 1,
				},
				"performance": {
					Enabled:  true,
					Settings: map[string]interface{}{
						"enable_memory_leaks": true,
						"enable_slow_operations": true,
						"enable_loop_optimization": true,
					},
					Severity: "warning",
					Priority: 2,
				},
			},
			GlobalSettings: map[string]interface{}{
				"max_line_length": 120,
				"indent_size": 4,
				"use_tabs": false,
			},
			OutputFormat:  "json",
			MinConfidence: 0.5,
			MaxIssues:     1000,
		},
		Database: DatabaseConfig{
			Type:           "sqlite",
			ConnectionString: "file:kodevibe.db?cache=shared&mode=rwc",
			MaxConnections: 10,
			MaxIdleTime:    30 * time.Minute,
			MaxLifetime:    time.Hour,
			EnableMetrics:  true,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			Structured: false,
		},
		Security: SecurityConfig{
			EnableHTTPS:    false,
			EnableHSTS:     false,
			EnableCSRF:     false,
			APIKeyHeader:   "X-API-Key",
			TrustedProxies: []string{},
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}
	
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive")
	}
	
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive")
	}

	// Validate scanner config
	if c.Scanner.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	
	if c.Scanner.MaxFiles <= 0 {
		return fmt.Errorf("max files must be positive")
	}
	
	if c.Scanner.MaxDepth < 0 {
		return fmt.Errorf("max depth cannot be negative")
	}
	
	if c.Scanner.ConcurrentWorkers <= 0 {
		return fmt.Errorf("concurrent workers must be positive")
	}

	// Validate vibes config
	if c.Vibes.MinConfidence < 0 || c.Vibes.MinConfidence > 1 {
		return fmt.Errorf("min confidence must be between 0 and 1")
	}
	
	if c.Vibes.MaxIssues <= 0 {
		return fmt.Errorf("max issues must be positive")
	}
	
	validFormats := map[string]bool{
		"json":   true,
		"yaml":   true,
		"xml":    true,
		"csv":    true,
		"table":  true,
	}
	
	if !validFormats[c.Vibes.OutputFormat] {
		return fmt.Errorf("invalid output format: %s", c.Vibes.OutputFormat)
	}

	// Validate database config
	if c.Database.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	// Validate logging config
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

// Save saves the configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}