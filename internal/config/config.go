package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Environment string         `mapstructure:"environment"`
	LogLevel    string         `mapstructure:"log_level"`
	LogFormat   string         `mapstructure:"log_format"`
	Server      ServerConfig   `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig    `mapstructure:"redis"`
	Auth        AuthConfig     `mapstructure:"auth"`
	RnD         RnDConfig      `mapstructure:"rnd"`
	Monitoring  MonitorConfig  `mapstructure:"monitoring"`
	WebSocket   WSConfig       `mapstructure:"websocket"`
	Security    SecurityConfig `mapstructure:"security"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
	TLSEnabled   bool   `mapstructure:"tls_enabled"`
	TLSCertFile  string `mapstructure:"tls_cert_file"`
	TLSKeyFile   string `mapstructure:"tls_key_file"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	MigrationsPath  string        `mapstructure:"migrations_path"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	PoolSize     int           `mapstructure:"pool_size"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`
	JWTExpiration      time.Duration `mapstructure:"jwt_expiration"`
	RefreshExpiration  time.Duration `mapstructure:"refresh_expiration"`
	PasswordMinLength  int           `mapstructure:"password_min_length"`
	MaxLoginAttempts   int           `mapstructure:"max_login_attempts"`
	LoginAttemptWindow time.Duration `mapstructure:"login_attempt_window"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout"`
	RequireEmailVerify bool          `mapstructure:"require_email_verify"`
}

// RnDConfig holds R&D module configuration
type RnDConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	WorkerCount          int           `mapstructure:"worker_count"`
	QueueSize            int           `mapstructure:"queue_size"`
	ProcessingTimeout    time.Duration `mapstructure:"processing_timeout"`
	LearningInterval     time.Duration `mapstructure:"learning_interval"`
	PatternAnalysisDepth int           `mapstructure:"pattern_analysis_depth"`
	ProjectGenerationMax int           `mapstructure:"project_generation_max"`
	CoordinationMode     string        `mapstructure:"coordination_mode"`
	AgentMaxCount        int           `mapstructure:"agent_max_count"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	MetricsPath        string        `mapstructure:"metrics_path"`
	HealthCheckPath    string        `mapstructure:"health_check_path"`
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	RetentionPeriod    time.Duration `mapstructure:"retention_period"`
	AlertsEnabled      bool          `mapstructure:"alerts_enabled"`
	AlertWebhookURL    string        `mapstructure:"alert_webhook_url"`
}

// WSConfig holds WebSocket configuration
type WSConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	Path             string        `mapstructure:"path"`
	ReadBufferSize   int           `mapstructure:"read_buffer_size"`
	WriteBufferSize  int           `mapstructure:"write_buffer_size"`
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout"`
	PingPeriod       time.Duration `mapstructure:"ping_period"`
	PongWait         time.Duration `mapstructure:"pong_wait"`
	WriteWait        time.Duration `mapstructure:"write_wait"`
	MaxMessageSize   int64         `mapstructure:"max_message_size"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// CORS settings
	CORS CORSConfig `mapstructure:"cors"`

	// Rate limiting
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`

	// Security headers
	Headers SecurityHeadersConfig `mapstructure:"headers"`

	// Input validation
	Validation ValidationConfig `mapstructure:"validation"`

	// API security
	API APISecurityConfig `mapstructure:"api"`

	// File upload security
	FileUpload FileUploadConfig `mapstructure:"file_upload"`

	// Security monitoring
	Monitoring SecurityMonitoringConfig `mapstructure:"monitoring"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	MaxAge           int      `mapstructure:"max_age"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	GlobalRPS        int           `mapstructure:"global_rps"`
	GlobalBurst      int           `mapstructure:"global_burst"`
	PerIPRPS         int           `mapstructure:"per_ip_rps"`
	PerIPBurst       int           `mapstructure:"per_ip_burst"`
	PerUserRPS       int           `mapstructure:"per_user_rps"`
	PerUserBurst     int           `mapstructure:"per_user_burst"`
	WindowSize       time.Duration `mapstructure:"window_size"`
	CleanupInterval  time.Duration `mapstructure:"cleanup_interval"`
	BlockDuration    time.Duration `mapstructure:"block_duration"`
	WhitelistedIPs   []string      `mapstructure:"whitelisted_ips"`
	WhitelistedUsers []string      `mapstructure:"whitelisted_users"`
}

// SecurityHeadersConfig holds security headers configuration
type SecurityHeadersConfig struct {
	Enabled               bool              `mapstructure:"enabled"`
	ContentSecurityPolicy string            `mapstructure:"content_security_policy"`
	HSTSMaxAge            int               `mapstructure:"hsts_max_age"`
	HSTSIncludeSubdomains bool              `mapstructure:"hsts_include_subdomains"`
	HSTSPreload           bool              `mapstructure:"hsts_preload"`
	XFrameOptions         string            `mapstructure:"x_frame_options"`
	XContentTypeOptions   string            `mapstructure:"x_content_type_options"`
	XXSSProtection        string            `mapstructure:"x_xss_protection"`
	ReferrerPolicy        string            `mapstructure:"referrer_policy"`
	PermissionsPolicy     string            `mapstructure:"permissions_policy"`
	CustomHeaders         map[string]string `mapstructure:"custom_headers"`
}

// ValidationConfig holds input validation configuration
type ValidationConfig struct {
	Enabled                bool     `mapstructure:"enabled"`
	EnableHTMLSanitization bool     `mapstructure:"enable_html_sanitization"`
	StrictMode             bool     `mapstructure:"strict_mode"`
	MaxFieldLength         int      `mapstructure:"max_field_length"`
	AllowedFileTypes       []string `mapstructure:"allowed_file_types"`
	BlockedPatterns        []string `mapstructure:"blocked_patterns"`
}

// APISecurityConfig holds API security configuration
type APISecurityConfig struct {
	RequireAPIKey    bool          `mapstructure:"require_api_key"`
	APIKeyHeader     string        `mapstructure:"api_key_header"`
	RequestTimeout   time.Duration `mapstructure:"request_timeout"`
	MaxRequestSize   int64         `mapstructure:"max_request_size"`
	EnableRequestID  bool          `mapstructure:"enable_request_id"`
	LogSensitiveData bool          `mapstructure:"log_sensitive_data"`
}

// FileUploadConfig holds file upload security configuration
type FileUploadConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	MaxFileSize       int64    `mapstructure:"max_file_size"`
	AllowedMimeTypes  []string `mapstructure:"allowed_mime_types"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
	ScanForMalware    bool     `mapstructure:"scan_for_malware"`
	QuarantinePath    string   `mapstructure:"quarantine_path"`
}

// SecurityMonitoringConfig holds security monitoring configuration
type SecurityMonitoringConfig struct {
	Enabled                   bool          `mapstructure:"enabled"`
	LogSecurityEvents         bool          `mapstructure:"log_security_events"`
	AlertOnSuspiciousActivity bool          `mapstructure:"alert_on_suspicious_activity"`
	MaxFailedAttempts         int           `mapstructure:"max_failed_attempts"`
	LockoutDuration           time.Duration `mapstructure:"lockout_duration"`
	NotificationWebhook       string        `mapstructure:"notification_webhook"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Environment
	viper.SetDefault("environment", "development")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_format", "json")

	// Server
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 120)
	viper.SetDefault("server.tls_enabled", false)

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "kaskmanager")
	viper.SetDefault("database.database", "kaskmanager_rd")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.migrations_path", "migrations")

	// Redis
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.pool_size", 10)

	// Auth
	viper.SetDefault("auth.jwt_secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt_expiration", "24h")
	viper.SetDefault("auth.refresh_expiration", "168h") // 7 days
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.login_attempt_window", "15m")
	viper.SetDefault("auth.session_timeout", "24h")
	viper.SetDefault("auth.require_email_verify", false)

	// R&D
	viper.SetDefault("rnd.enabled", true)
	viper.SetDefault("rnd.worker_count", 4)
	viper.SetDefault("rnd.queue_size", 100)
	viper.SetDefault("rnd.processing_timeout", "30m")
	viper.SetDefault("rnd.learning_interval", "1h")
	viper.SetDefault("rnd.pattern_analysis_depth", 5)
	viper.SetDefault("rnd.project_generation_max", 10)
	viper.SetDefault("rnd.coordination_mode", "centralized")
	viper.SetDefault("rnd.agent_max_count", 10)

	// Monitoring
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.metrics_path", "/metrics")
	viper.SetDefault("monitoring.health_check_path", "/health")
	viper.SetDefault("monitoring.collection_interval", "30s")
	viper.SetDefault("monitoring.retention_period", "24h")
	viper.SetDefault("monitoring.alerts_enabled", false)

	// WebSocket
	viper.SetDefault("websocket.enabled", true)
	viper.SetDefault("websocket.path", "/ws")
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
	viper.SetDefault("websocket.handshake_timeout", "10s")
	viper.SetDefault("websocket.ping_period", "54s")
	viper.SetDefault("websocket.pong_wait", "60s")
	viper.SetDefault("websocket.write_wait", "10s")
	viper.SetDefault("websocket.max_message_size", 512)

	// Security
	viper.SetDefault("security.cors.allowed_origins", []string{})
	viper.SetDefault("security.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors.allowed_headers", []string{"Content-Type", "Authorization", "X-Requested-With"})
	viper.SetDefault("security.cors.max_age", 86400)
	viper.SetDefault("security.cors.allow_credentials", true)

	// Rate limiting
	viper.SetDefault("security.rate_limit.enabled", true)
	viper.SetDefault("security.rate_limit.global_rps", 1000)
	viper.SetDefault("security.rate_limit.global_burst", 2000)
	viper.SetDefault("security.rate_limit.per_ip_rps", 100)
	viper.SetDefault("security.rate_limit.per_ip_burst", 200)
	viper.SetDefault("security.rate_limit.per_user_rps", 500)
	viper.SetDefault("security.rate_limit.per_user_burst", 1000)
	viper.SetDefault("security.rate_limit.window_size", "1m")
	viper.SetDefault("security.rate_limit.cleanup_interval", "5m")
	viper.SetDefault("security.rate_limit.block_duration", "15m")
	viper.SetDefault("security.rate_limit.whitelisted_ips", []string{})
	viper.SetDefault("security.rate_limit.whitelisted_users", []string{})

	// Security headers
	viper.SetDefault("security.headers.enabled", true)
	viper.SetDefault("security.headers.hsts_max_age", 31536000)
	viper.SetDefault("security.headers.hsts_include_subdomains", true)
	viper.SetDefault("security.headers.hsts_preload", false)
	viper.SetDefault("security.headers.x_frame_options", "DENY")
	viper.SetDefault("security.headers.x_content_type_options", "nosniff")
	viper.SetDefault("security.headers.x_xss_protection", "1; mode=block")
	viper.SetDefault("security.headers.referrer_policy", "strict-origin-when-cross-origin")

	// Input validation
	viper.SetDefault("security.validation.enabled", true)
	viper.SetDefault("security.validation.enable_html_sanitization", true)
	viper.SetDefault("security.validation.strict_mode", false)
	viper.SetDefault("security.validation.max_field_length", 5000)
	viper.SetDefault("security.validation.allowed_file_types", []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx", ".txt", ".csv"})

	// API security
	viper.SetDefault("security.api.require_api_key", false)
	viper.SetDefault("security.api.api_key_header", "X-API-Key")
	viper.SetDefault("security.api.request_timeout", "30s")
	viper.SetDefault("security.api.max_request_size", 10485760) // 10MB
	viper.SetDefault("security.api.enable_request_id", true)
	viper.SetDefault("security.api.log_sensitive_data", false)

	// File upload
	viper.SetDefault("security.file_upload.enabled", true)
	viper.SetDefault("security.file_upload.max_file_size", 10485760) // 10MB
	viper.SetDefault("security.file_upload.allowed_mime_types", []string{"image/jpeg", "image/png", "image/gif", "application/pdf", "text/plain"})
	viper.SetDefault("security.file_upload.allowed_extensions", []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt"})
	viper.SetDefault("security.file_upload.scan_for_malware", false)

	// Security monitoring
	viper.SetDefault("security.monitoring.enabled", true)
	viper.SetDefault("security.monitoring.log_security_events", true)
	viper.SetDefault("security.monitoring.alert_on_suspicious_activity", true)
	viper.SetDefault("security.monitoring.max_failed_attempts", 5)
	viper.SetDefault("security.monitoring.lockout_duration", "15m")
}

// validate validates the configuration
func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Auth.JWTSecret == "" || config.Auth.JWTSecret == "your-secret-key-change-in-production" {
		if config.Environment == "production" {
			return fmt.Errorf("JWT secret must be set in production")
		}
	}

	if config.Auth.PasswordMinLength < 4 {
		return fmt.Errorf("password minimum length must be at least 4")
	}

	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}
