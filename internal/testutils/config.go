package testutils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TestEnvironment represents different testing environments
type TestEnvironment string

const (
	TestEnvUnit        TestEnvironment = "unit"
	TestEnvIntegration TestEnvironment = "integration"
	TestEnvE2E         TestEnvironment = "e2e"
	TestEnvPerformance TestEnvironment = "performance"
	TestEnvLoad        TestEnvironment = "load"
)

// TestConfiguration holds all test configuration
type TestConfiguration struct {
	Environment TestEnvironment
	Database    *DatabaseTestConfig
	Server      *ServerTestConfig
	WebSocket   *WebSocketTestConfig
	Performance *PerformanceTestConfig
	Load        *LoadTestConfig
	Security    *SecurityTestConfig
	Integration *IntegrationTestConfig
}

// DatabaseTestConfig holds database test configuration
type DatabaseTestConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MaxIdleTime     time.Duration
	MaxLifetime     time.Duration
	CleanupTimeout  time.Duration
	MigrationPath   string
	SeedDataPath    string
	TestDataPath    string
	BackupPath      string
	TruncateOnSetup bool
	RecreateOnSetup bool
}

// ServerTestConfig holds server test configuration
type ServerTestConfig struct {
	Host              string
	Port              int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	EnableProfiling   bool
	EnableMetrics     bool
	EnableHealthCheck bool
	StaticFilesPath   string
	TemplatesPath     string
	LogLevel          string
	LogFormat         string
	CORSEnabled       bool
	CORSOrigins       []string
	RateLimitEnabled  bool
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// WebSocketTestConfig holds WebSocket test configuration
type WebSocketTestConfig struct {
	Host                 string
	Port                 int
	Path                 string
	ReadBufferSize       int
	WriteBufferSize      int
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	PingInterval         time.Duration
	PongTimeout          time.Duration
	MaxMessageSize       int64
	MaxConnections       int
	EnableCompression    bool
	EnableOriginCheck    bool
	AllowedOrigins       []string
	HeartbeatInterval    time.Duration
	ReconnectInterval    time.Duration
	MaxReconnectAttempts int
}

// PerformanceTestConfig holds performance test configuration
type PerformanceTestConfig struct {
	MaxResponseTime        time.Duration
	MaxMemoryUsage         int64
	MaxCPUUsage            float64
	MaxDatabaseConnections int
	MaxConcurrentRequests  int
	MinThroughput          int
	MaxErrorRate           float64
	WarmupDuration         time.Duration
	TestDuration           time.Duration
	CooldownDuration       time.Duration
	SampleInterval         time.Duration
	EnableProfiling        bool
	EnableMetrics          bool
	MetricsPort            int
	ProfilingPort          int
	ReportPath             string
	ReportFormat           string
}

// LoadTestConfig holds load test configuration
type LoadTestConfig struct {
	ConcurrentUsers          int
	RequestsPerUser          int
	RampUpDuration           time.Duration
	TestDuration             time.Duration
	RampDownDuration         time.Duration
	RequestInterval          time.Duration
	MaxResponseTime          time.Duration
	AcceptableErrorRate      float64
	ScenarioFiles            []string
	DataFiles                []string
	ReportPath               string
	ReportFormat             string
	EnableRealTimeMetrics    bool
	EnableDetailedLogs       bool
	EnableResourceMonitoring bool
}

// SecurityTestConfig holds security test configuration
type SecurityTestConfig struct {
	EnableSQLInjectionTests     bool
	EnableXSSTests              bool
	EnableCSRFTests             bool
	EnableAuthTests             bool
	EnablePermissionTests       bool
	EnableInputValidationTests  bool
	EnableOutputEncodingTests   bool
	EnableRateLimitTests        bool
	EnableBruteForceTests       bool
	EnableSessionTests          bool
	TestUsersFile               string
	TestPasswordsFile           string
	TestPayloadsFile            string
	ReportPath                  string
	ReportFormat                string
	EnableVulnerabilityScanning bool
	ScannerConfig               map[string]string
}

// IntegrationTestConfig holds integration test configuration
type IntegrationTestConfig struct {
	EnableAPITests         bool
	EnableWebSocketTests   bool
	EnableDatabaseTests    bool
	EnableExternalServices bool
	EnableE2ETests         bool
	APITestsPath           string
	WebSocketTestsPath     string
	DatabaseTestsPath      string
	ExternalServicesConfig map[string]string
	E2ETestsPath           string
	TestDataPath           string
	MockServicesEnabled    bool
	MockServicesConfig     map[string]string
	ParallelExecution      bool
	MaxParallelTests       int
	TestTimeout            time.Duration
	RetryAttempts          int
	RetryDelay             time.Duration
}

// NewTestConfiguration creates a new test configuration
func NewTestConfiguration(env TestEnvironment) *TestConfiguration {
	return &TestConfiguration{
		Environment: env,
		Database:    newDatabaseTestConfig(),
		Server:      newServerTestConfig(),
		WebSocket:   newWebSocketTestConfig(),
		Performance: newPerformanceTestConfig(),
		Load:        newLoadTestConfig(),
		Security:    newSecurityTestConfig(),
		Integration: newIntegrationTestConfig(),
	}
}

// newDatabaseTestConfig creates database test configuration
func newDatabaseTestConfig() *DatabaseTestConfig {
	return &DatabaseTestConfig{
		Host:            getEnvString("TEST_DB_HOST", "localhost"),
		Port:            getEnvInt("TEST_DB_PORT", 5432),
		Username:        getEnvString("TEST_DB_USER", "kaskmanager"),
		Password:        getEnvString("TEST_DB_PASSWORD", "password"),
		Database:        getEnvString("TEST_DB_NAME", "kaskmanager_test"),
		SSLMode:         getEnvString("TEST_DB_SSLMODE", "disable"),
		MaxConnections:  getEnvInt("TEST_DB_MAX_CONNECTIONS", 25),
		MaxIdleTime:     getEnvDuration("TEST_DB_MAX_IDLE_TIME", 5*time.Minute),
		MaxLifetime:     getEnvDuration("TEST_DB_MAX_LIFETIME", 30*time.Minute),
		CleanupTimeout:  getEnvDuration("TEST_DB_CLEANUP_TIMEOUT", 30*time.Second),
		MigrationPath:   getEnvString("TEST_DB_MIGRATION_PATH", "./migrations"),
		SeedDataPath:    getEnvString("TEST_DB_SEED_DATA_PATH", "./testdata/seeds"),
		TestDataPath:    getEnvString("TEST_DB_TEST_DATA_PATH", "./testdata/tests"),
		BackupPath:      getEnvString("TEST_DB_BACKUP_PATH", "./testdata/backups"),
		TruncateOnSetup: getEnvBool("TEST_DB_TRUNCATE_ON_SETUP", true),
		RecreateOnSetup: getEnvBool("TEST_DB_RECREATE_ON_SETUP", false),
	}
}

// newServerTestConfig creates server test configuration
func newServerTestConfig() *ServerTestConfig {
	return &ServerTestConfig{
		Host:              getEnvString("TEST_SERVER_HOST", "localhost"),
		Port:              getEnvInt("TEST_SERVER_PORT", 8080),
		ReadTimeout:       getEnvDuration("TEST_SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      getEnvDuration("TEST_SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       getEnvDuration("TEST_SERVER_IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout:   getEnvDuration("TEST_SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		EnableProfiling:   getEnvBool("TEST_SERVER_ENABLE_PROFILING", false),
		EnableMetrics:     getEnvBool("TEST_SERVER_ENABLE_METRICS", false),
		EnableHealthCheck: getEnvBool("TEST_SERVER_ENABLE_HEALTH_CHECK", true),
		StaticFilesPath:   getEnvString("TEST_SERVER_STATIC_FILES_PATH", "./web/static"),
		TemplatesPath:     getEnvString("TEST_SERVER_TEMPLATES_PATH", "./web/templates"),
		LogLevel:          getEnvString("TEST_SERVER_LOG_LEVEL", "warn"),
		LogFormat:         getEnvString("TEST_SERVER_LOG_FORMAT", "json"),
		CORSEnabled:       getEnvBool("TEST_SERVER_CORS_ENABLED", true),
		CORSOrigins:       getEnvStringSlice("TEST_SERVER_CORS_ORIGINS", []string{"*"}),
		RateLimitEnabled:  getEnvBool("TEST_SERVER_RATE_LIMIT_ENABLED", false),
		RateLimitRequests: getEnvInt("TEST_SERVER_RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvDuration("TEST_SERVER_RATE_LIMIT_WINDOW", time.Minute),
	}
}

// newWebSocketTestConfig creates WebSocket test configuration
func newWebSocketTestConfig() *WebSocketTestConfig {
	return &WebSocketTestConfig{
		Host:                 getEnvString("TEST_WS_HOST", "localhost"),
		Port:                 getEnvInt("TEST_WS_PORT", 8080),
		Path:                 getEnvString("TEST_WS_PATH", "/ws"),
		ReadBufferSize:       getEnvInt("TEST_WS_READ_BUFFER_SIZE", 1024),
		WriteBufferSize:      getEnvInt("TEST_WS_WRITE_BUFFER_SIZE", 1024),
		ReadTimeout:          getEnvDuration("TEST_WS_READ_TIMEOUT", 60*time.Second),
		WriteTimeout:         getEnvDuration("TEST_WS_WRITE_TIMEOUT", 10*time.Second),
		PingInterval:         getEnvDuration("TEST_WS_PING_INTERVAL", 30*time.Second),
		PongTimeout:          getEnvDuration("TEST_WS_PONG_TIMEOUT", 60*time.Second),
		MaxMessageSize:       getEnvInt64("TEST_WS_MAX_MESSAGE_SIZE", 512*1024),
		MaxConnections:       getEnvInt("TEST_WS_MAX_CONNECTIONS", 100),
		EnableCompression:    getEnvBool("TEST_WS_ENABLE_COMPRESSION", true),
		EnableOriginCheck:    getEnvBool("TEST_WS_ENABLE_ORIGIN_CHECK", false),
		AllowedOrigins:       getEnvStringSlice("TEST_WS_ALLOWED_ORIGINS", []string{"*"}),
		HeartbeatInterval:    getEnvDuration("TEST_WS_HEARTBEAT_INTERVAL", 30*time.Second),
		ReconnectInterval:    getEnvDuration("TEST_WS_RECONNECT_INTERVAL", 5*time.Second),
		MaxReconnectAttempts: getEnvInt("TEST_WS_MAX_RECONNECT_ATTEMPTS", 5),
	}
}

// newPerformanceTestConfig creates performance test configuration
func newPerformanceTestConfig() *PerformanceTestConfig {
	return &PerformanceTestConfig{
		MaxResponseTime:        getEnvDuration("TEST_PERF_MAX_RESPONSE_TIME", 100*time.Millisecond),
		MaxMemoryUsage:         getEnvInt64("TEST_PERF_MAX_MEMORY_USAGE", 512*1024*1024),
		MaxCPUUsage:            getEnvFloat64("TEST_PERF_MAX_CPU_USAGE", 80.0),
		MaxDatabaseConnections: getEnvInt("TEST_PERF_MAX_DB_CONNECTIONS", 50),
		MaxConcurrentRequests:  getEnvInt("TEST_PERF_MAX_CONCURRENT_REQUESTS", 100),
		MinThroughput:          getEnvInt("TEST_PERF_MIN_THROUGHPUT", 1000),
		MaxErrorRate:           getEnvFloat64("TEST_PERF_MAX_ERROR_RATE", 1.0),
		WarmupDuration:         getEnvDuration("TEST_PERF_WARMUP_DURATION", 30*time.Second),
		TestDuration:           getEnvDuration("TEST_PERF_TEST_DURATION", 5*time.Minute),
		CooldownDuration:       getEnvDuration("TEST_PERF_COOLDOWN_DURATION", 30*time.Second),
		SampleInterval:         getEnvDuration("TEST_PERF_SAMPLE_INTERVAL", time.Second),
		EnableProfiling:        getEnvBool("TEST_PERF_ENABLE_PROFILING", true),
		EnableMetrics:          getEnvBool("TEST_PERF_ENABLE_METRICS", true),
		MetricsPort:            getEnvInt("TEST_PERF_METRICS_PORT", 9090),
		ProfilingPort:          getEnvInt("TEST_PERF_PROFILING_PORT", 6060),
		ReportPath:             getEnvString("TEST_PERF_REPORT_PATH", "./reports/performance"),
		ReportFormat:           getEnvString("TEST_PERF_REPORT_FORMAT", "html"),
	}
}

// newLoadTestConfig creates load test configuration
func newLoadTestConfig() *LoadTestConfig {
	return &LoadTestConfig{
		ConcurrentUsers:          getEnvInt("TEST_LOAD_CONCURRENT_USERS", 10),
		RequestsPerUser:          getEnvInt("TEST_LOAD_REQUESTS_PER_USER", 100),
		RampUpDuration:           getEnvDuration("TEST_LOAD_RAMP_UP_DURATION", 2*time.Minute),
		TestDuration:             getEnvDuration("TEST_LOAD_TEST_DURATION", 10*time.Minute),
		RampDownDuration:         getEnvDuration("TEST_LOAD_RAMP_DOWN_DURATION", 1*time.Minute),
		RequestInterval:          getEnvDuration("TEST_LOAD_REQUEST_INTERVAL", 100*time.Millisecond),
		MaxResponseTime:          getEnvDuration("TEST_LOAD_MAX_RESPONSE_TIME", 5*time.Second),
		AcceptableErrorRate:      getEnvFloat64("TEST_LOAD_ACCEPTABLE_ERROR_RATE", 5.0),
		ScenarioFiles:            getEnvStringSlice("TEST_LOAD_SCENARIO_FILES", []string{"./testdata/scenarios"}),
		DataFiles:                getEnvStringSlice("TEST_LOAD_DATA_FILES", []string{"./testdata/load"}),
		ReportPath:               getEnvString("TEST_LOAD_REPORT_PATH", "./reports/load"),
		ReportFormat:             getEnvString("TEST_LOAD_REPORT_FORMAT", "html"),
		EnableRealTimeMetrics:    getEnvBool("TEST_LOAD_ENABLE_REAL_TIME_METRICS", true),
		EnableDetailedLogs:       getEnvBool("TEST_LOAD_ENABLE_DETAILED_LOGS", false),
		EnableResourceMonitoring: getEnvBool("TEST_LOAD_ENABLE_RESOURCE_MONITORING", true),
	}
}

// newSecurityTestConfig creates security test configuration
func newSecurityTestConfig() *SecurityTestConfig {
	return &SecurityTestConfig{
		EnableSQLInjectionTests:     getEnvBool("TEST_SEC_ENABLE_SQL_INJECTION", true),
		EnableXSSTests:              getEnvBool("TEST_SEC_ENABLE_XSS", true),
		EnableCSRFTests:             getEnvBool("TEST_SEC_ENABLE_CSRF", true),
		EnableAuthTests:             getEnvBool("TEST_SEC_ENABLE_AUTH", true),
		EnablePermissionTests:       getEnvBool("TEST_SEC_ENABLE_PERMISSION", true),
		EnableInputValidationTests:  getEnvBool("TEST_SEC_ENABLE_INPUT_VALIDATION", true),
		EnableOutputEncodingTests:   getEnvBool("TEST_SEC_ENABLE_OUTPUT_ENCODING", true),
		EnableRateLimitTests:        getEnvBool("TEST_SEC_ENABLE_RATE_LIMIT", true),
		EnableBruteForceTests:       getEnvBool("TEST_SEC_ENABLE_BRUTE_FORCE", true),
		EnableSessionTests:          getEnvBool("TEST_SEC_ENABLE_SESSION", true),
		TestUsersFile:               getEnvString("TEST_SEC_TEST_USERS_FILE", "./testdata/security/users.json"),
		TestPasswordsFile:           getEnvString("TEST_SEC_TEST_PASSWORDS_FILE", "./testdata/security/passwords.txt"),
		TestPayloadsFile:            getEnvString("TEST_SEC_TEST_PAYLOADS_FILE", "./testdata/security/payloads.json"),
		ReportPath:                  getEnvString("TEST_SEC_REPORT_PATH", "./reports/security"),
		ReportFormat:                getEnvString("TEST_SEC_REPORT_FORMAT", "html"),
		EnableVulnerabilityScanning: getEnvBool("TEST_SEC_ENABLE_VULN_SCANNING", true),
		ScannerConfig: map[string]string{
			"timeout": "30s",
			"depth":   "3",
		},
	}
}

// newIntegrationTestConfig creates integration test configuration
func newIntegrationTestConfig() *IntegrationTestConfig {
	return &IntegrationTestConfig{
		EnableAPITests:         getEnvBool("TEST_INT_ENABLE_API", true),
		EnableWebSocketTests:   getEnvBool("TEST_INT_ENABLE_WEBSOCKET", true),
		EnableDatabaseTests:    getEnvBool("TEST_INT_ENABLE_DATABASE", true),
		EnableExternalServices: getEnvBool("TEST_INT_ENABLE_EXTERNAL_SERVICES", false),
		EnableE2ETests:         getEnvBool("TEST_INT_ENABLE_E2E", true),
		APITestsPath:           getEnvString("TEST_INT_API_TESTS_PATH", "./tests/integration/api"),
		WebSocketTestsPath:     getEnvString("TEST_INT_WS_TESTS_PATH", "./tests/integration/websocket"),
		DatabaseTestsPath:      getEnvString("TEST_INT_DB_TESTS_PATH", "./tests/integration/database"),
		ExternalServicesConfig: map[string]string{
			"timeout": "30s",
			"retries": "3",
		},
		E2ETestsPath:        getEnvString("TEST_INT_E2E_TESTS_PATH", "./tests/e2e"),
		TestDataPath:        getEnvString("TEST_INT_TEST_DATA_PATH", "./testdata/integration"),
		MockServicesEnabled: getEnvBool("TEST_INT_MOCK_SERVICES_ENABLED", true),
		MockServicesConfig: map[string]string{
			"port": "8081",
		},
		ParallelExecution: getEnvBool("TEST_INT_PARALLEL_EXECUTION", true),
		MaxParallelTests:  getEnvInt("TEST_INT_MAX_PARALLEL_TESTS", 4),
		TestTimeout:       getEnvDuration("TEST_INT_TEST_TIMEOUT", 5*time.Minute),
		RetryAttempts:     getEnvInt("TEST_INT_RETRY_ATTEMPTS", 3),
		RetryDelay:        getEnvDuration("TEST_INT_RETRY_DELAY", 5*time.Second),
	}
}

// GetDatabaseURL returns the database URL for testing
func (c *TestConfiguration) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetServerURL returns the server URL for testing
func (c *TestConfiguration) GetServerURL() string {
	return fmt.Sprintf("http://%s:%d", c.Server.Host, c.Server.Port)
}

// GetWebSocketURL returns the WebSocket URL for testing
func (c *TestConfiguration) GetWebSocketURL() string {
	return fmt.Sprintf("ws://%s:%d%s", c.WebSocket.Host, c.WebSocket.Port, c.WebSocket.Path)
}

// IsEnvironment checks if current environment matches
func (c *TestConfiguration) IsEnvironment(env TestEnvironment) bool {
	return c.Environment == env
}

// ShouldRunPerformanceTests checks if performance tests should run
func (c *TestConfiguration) ShouldRunPerformanceTests() bool {
	return c.Environment == TestEnvPerformance || c.Environment == TestEnvLoad
}

// ShouldRunSecurityTests checks if security tests should run
func (c *TestConfiguration) ShouldRunSecurityTests() bool {
	return c.Environment == TestEnvIntegration || c.Environment == TestEnvE2E
}

// ShouldRunIntegrationTests checks if integration tests should run
func (c *TestConfiguration) ShouldRunIntegrationTests() bool {
	return c.Environment == TestEnvIntegration || c.Environment == TestEnvE2E
}

// Environment variable helper functions
func getEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvStringSlice(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return fallback
}

// LoadFromFile loads configuration from file
func (c *TestConfiguration) LoadFromFile(filename string) error {
	// Implementation would load from YAML/JSON file
	return nil
}

// SaveToFile saves configuration to file
func (c *TestConfiguration) SaveToFile(filename string) error {
	// Implementation would save to YAML/JSON file
	return nil
}

// Validate validates the configuration
func (c *TestConfiguration) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 {
		return fmt.Errorf("database port must be positive")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Server.Port <= 0 {
		return fmt.Errorf("server port must be positive")
	}
	if c.WebSocket.Port <= 0 {
		return fmt.Errorf("websocket port must be positive")
	}
	return nil
}

// Default configurations for different environments
func NewUnitTestConfig() *TestConfiguration {
	config := NewTestConfiguration(TestEnvUnit)
	config.Database.RecreateOnSetup = true
	config.Server.EnableProfiling = false
	config.Server.EnableMetrics = false
	return config
}

func NewIntegrationTestConfig() *TestConfiguration {
	config := NewTestConfiguration(TestEnvIntegration)
	config.Database.RecreateOnSetup = false
	config.Server.EnableProfiling = true
	config.Server.EnableMetrics = true
	return config
}

func NewPerformanceTestConfig() *TestConfiguration {
	config := NewTestConfiguration(TestEnvPerformance)
	config.Database.RecreateOnSetup = false
	config.Server.EnableProfiling = true
	config.Server.EnableMetrics = true
	config.Performance.EnableProfiling = true
	config.Performance.EnableMetrics = true
	return config
}

func NewLoadTestConfig() *TestConfiguration {
	config := NewTestConfiguration(TestEnvLoad)
	config.Database.RecreateOnSetup = false
	config.Server.EnableProfiling = true
	config.Server.EnableMetrics = true
	config.Load.EnableRealTimeMetrics = true
	config.Load.EnableResourceMonitoring = true
	return config
}
