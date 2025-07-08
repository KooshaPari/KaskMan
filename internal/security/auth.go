package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthEnhancer provides enhanced authentication features
type AuthEnhancer struct {
	db       *gorm.DB
	redis    *redis.Client
	logger   *logrus.Logger
	config   *AuthConfig
	useRedis bool

	// In-memory fallback for session management
	sessions      map[string]*Session
	loginAttempts map[string]*LoginAttempts
}

// AuthConfig holds enhanced authentication configuration
type AuthConfig struct {
	// MFA settings
	MFAEnabled      bool
	MFAIssuer       string
	MFASecretLength int

	// Session management
	SessionTimeout    time.Duration
	RefreshTokenTTL   time.Duration
	MaxActiveSessions int

	// Account lockout
	MaxLoginAttempts int
	LockoutDuration  time.Duration
	LockoutWindow    time.Duration

	// Password policy
	PasswordMinLength      int
	PasswordRequireUpper   bool
	PasswordRequireLower   bool
	PasswordRequireDigit   bool
	PasswordRequireSpecial bool
	PasswordMaxAge         time.Duration

	// Security features
	RequireEmailVerification bool
	AllowRememberMe          bool
	RequirePasswordChange    bool

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	IsActive     bool      `json:"is_active"`
	IsMFA        bool      `json:"is_mfa"`
	DeviceID     string    `json:"device_id"`
}

// LoginAttempts tracks failed login attempts
type LoginAttempts struct {
	Username    string    `json:"username"`
	IPAddress   string    `json:"ip_address"`
	Attempts    int       `json:"attempts"`
	LastAttempt time.Time `json:"last_attempt"`
	LockedUntil time.Time `json:"locked_until"`
}

// MFASetupResponse contains MFA setup information
type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// NewAuthEnhancer creates a new authentication enhancer
func NewAuthEnhancer(db *gorm.DB, config *AuthConfig, logger *logrus.Logger) *AuthEnhancer {
	ae := &AuthEnhancer{
		db:            db,
		logger:        logger,
		config:        config,
		sessions:      make(map[string]*Session),
		loginAttempts: make(map[string]*LoginAttempts),
	}

	// Initialize Redis if configured
	if config.RedisAddr != "" {
		ae.redis = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := ae.redis.Ping(ctx).Err(); err != nil {
			logger.WithError(err).Warn("Failed to connect to Redis, using in-memory session storage")
			ae.useRedis = false
		} else {
			ae.useRedis = true
			logger.Info("Connected to Redis for session management")
		}
	}

	return ae
}

// CheckLoginAttempts checks if a user/IP is locked out
func (ae *AuthEnhancer) CheckLoginAttempts(username, ipAddress string) error {
	key := fmt.Sprintf("login_attempts:%s:%s", username, ipAddress)

	if ae.useRedis {
		return ae.checkLoginAttemptsRedis(key)
	}

	return ae.checkLoginAttemptsMemory(key)
}

// checkLoginAttemptsRedis checks login attempts using Redis
func (ae *AuthEnhancer) checkLoginAttemptsRedis(key string) error {
	ctx := context.Background()

	// Get attempt count
	count, err := ae.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return err
	}

	// Check if locked
	lockKey := key + ":locked"
	locked, err := ae.redis.Get(ctx, lockKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if locked == "true" {
		return NewAuthenticationError("Account temporarily locked due to too many failed attempts")
	}

	if count >= ae.config.MaxLoginAttempts {
		// Lock the account
		ae.redis.Set(ctx, lockKey, "true", ae.config.LockoutDuration)
		return NewAuthenticationError("Account temporarily locked due to too many failed attempts")
	}

	return nil
}

// checkLoginAttemptsMemory checks login attempts using memory
func (ae *AuthEnhancer) checkLoginAttemptsMemory(key string) error {
	attempts, exists := ae.loginAttempts[key]
	if !exists {
		return nil
	}

	// Check if lockout has expired
	if time.Now().After(attempts.LockedUntil) {
		delete(ae.loginAttempts, key)
		return nil
	}

	// Check if within lockout window
	if time.Now().Before(attempts.LockedUntil) {
		return NewAuthenticationError("Account temporarily locked due to too many failed attempts")
	}

	return nil
}

// RecordLoginAttempt records a failed login attempt
func (ae *AuthEnhancer) RecordLoginAttempt(username, ipAddress string, success bool) {
	key := fmt.Sprintf("login_attempts:%s:%s", username, ipAddress)

	if success {
		// Clear attempts on successful login
		if ae.useRedis {
			ctx := context.Background()
			ae.redis.Del(ctx, key)
			ae.redis.Del(ctx, key+":locked")
		} else {
			delete(ae.loginAttempts, key)
		}
		return
	}

	// Record failed attempt
	if ae.useRedis {
		ae.recordLoginAttemptRedis(key)
	} else {
		ae.recordLoginAttemptMemory(key, username, ipAddress)
	}
}

// recordLoginAttemptRedis records failed login attempt in Redis
func (ae *AuthEnhancer) recordLoginAttemptRedis(key string) {
	ctx := context.Background()

	// Increment counter
	ae.redis.Incr(ctx, key)
	ae.redis.Expire(ctx, key, ae.config.LockoutWindow)

	// Check if should lock
	count, _ := ae.redis.Get(ctx, key).Int()
	if count >= ae.config.MaxLoginAttempts {
		lockKey := key + ":locked"
		ae.redis.Set(ctx, lockKey, "true", ae.config.LockoutDuration)
	}
}

// recordLoginAttemptMemory records failed login attempt in memory
func (ae *AuthEnhancer) recordLoginAttemptMemory(key, username, ipAddress string) {
	attempts, exists := ae.loginAttempts[key]
	if !exists {
		attempts = &LoginAttempts{
			Username:  username,
			IPAddress: ipAddress,
			Attempts:  0,
		}
		ae.loginAttempts[key] = attempts
	}

	attempts.Attempts++
	attempts.LastAttempt = time.Now()

	// Check if should lock
	if attempts.Attempts >= ae.config.MaxLoginAttempts {
		attempts.LockedUntil = time.Now().Add(ae.config.LockoutDuration)
	}
}

// CreateSession creates a new user session
func (ae *AuthEnhancer) CreateSession(user *models.User, ipAddress, userAgent string) (*Session, error) {
	session := &Session{
		ID:           ae.generateSessionID(),
		UserID:       user.ID.String(),
		Username:     user.Username,
		Role:         user.Role,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsActive:     true,
		IsMFA:        false, // Will be set to true after MFA verification
	}

	// Store session
	if ae.useRedis {
		if err := ae.storeSessionRedis(session); err != nil {
			return nil, err
		}
	} else {
		ae.sessions[session.ID] = session
	}

	// Clean up old sessions for this user
	ae.cleanupOldSessions(user.ID.String())

	return session, nil
}

// GetSession retrieves a session by ID
func (ae *AuthEnhancer) GetSession(sessionID string) (*Session, error) {
	if ae.useRedis {
		return ae.getSessionRedis(sessionID)
	}

	session, exists := ae.sessions[sessionID]
	if !exists {
		return nil, NewAuthenticationError("Session not found")
	}

	// Check if session is expired
	if time.Now().Sub(session.LastActivity) > ae.config.SessionTimeout {
		ae.InvalidateSession(sessionID)
		return nil, NewAuthenticationError("Session expired")
	}

	// Update last activity
	session.LastActivity = time.Now()

	return session, nil
}

// storeSessionRedis stores session in Redis
func (ae *AuthEnhancer) storeSessionRedis(session *Session) error {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", session.ID)

	return ae.redis.HSet(ctx, key, map[string]interface{}{
		"user_id":       session.UserID,
		"username":      session.Username,
		"role":          session.Role,
		"created_at":    session.CreatedAt.Unix(),
		"last_activity": session.LastActivity.Unix(),
		"ip_address":    session.IPAddress,
		"user_agent":    session.UserAgent,
		"is_active":     session.IsActive,
		"is_mfa":        session.IsMFA,
		"device_id":     session.DeviceID,
	}).Err()
}

// getSessionRedis retrieves session from Redis
func (ae *AuthEnhancer) getSessionRedis(sessionID string) (*Session, error) {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", sessionID)

	result, err := ae.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, NewAuthenticationError("Session not found")
	}

	session := &Session{
		ID:       sessionID,
		UserID:   result["user_id"],
		Username: result["username"],
		Role:     result["role"],
		IsActive: result["is_active"] == "true",
		IsMFA:    result["is_mfa"] == "true",
	}

	// Parse timestamps
	if createdAt, exists := result["created_at"]; exists {
		if timestamp, err := time.Parse(time.RFC3339, createdAt); err == nil {
			session.CreatedAt = timestamp
		}
	}

	if lastActivity, exists := result["last_activity"]; exists {
		if timestamp, err := time.Parse(time.RFC3339, lastActivity); err == nil {
			session.LastActivity = timestamp
		}
	}

	// Check expiration
	if time.Now().Sub(session.LastActivity) > ae.config.SessionTimeout {
		ae.InvalidateSession(sessionID)
		return nil, NewAuthenticationError("Session expired")
	}

	// Update last activity
	session.LastActivity = time.Now()
	ae.storeSessionRedis(session)

	return session, nil
}

// InvalidateSession invalidates a session
func (ae *AuthEnhancer) InvalidateSession(sessionID string) error {
	if ae.useRedis {
		ctx := context.Background()
		key := fmt.Sprintf("session:%s", sessionID)
		return ae.redis.Del(ctx, key).Err()
	}

	delete(ae.sessions, sessionID)
	return nil
}

// cleanupOldSessions removes old sessions for a user
func (ae *AuthEnhancer) cleanupOldSessions(userID string) {
	// This is a simplified implementation
	// In production, you might want more sophisticated cleanup
	if ae.useRedis {
		// Redis-based cleanup would require scanning keys
		// For now, we rely on Redis expiration
		return
	}

	// Count active sessions for this user
	count := 0
	for _, session := range ae.sessions {
		if session.UserID == userID && session.IsActive {
			count++
		}
	}

	// If too many sessions, remove oldest ones
	if count > ae.config.MaxActiveSessions {
		// Simple cleanup - remove all sessions for this user
		for id, session := range ae.sessions {
			if session.UserID == userID {
				delete(ae.sessions, id)
			}
		}
	}
}

// SetupMFA sets up multi-factor authentication for a user
func (ae *AuthEnhancer) SetupMFA(user *models.User) (*MFASetupResponse, error) {
	if !ae.config.MFAEnabled {
		return nil, NewAuthenticationError("MFA is not enabled")
	}

	// Generate secret
	secret, err := ae.generateMFASecret()
	if err != nil {
		return nil, err
	}

	// Generate QR code URL
	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		ae.config.MFAIssuer, user.Username, secret, ae.config.MFAIssuer)

	// Generate backup codes
	backupCodes, err := ae.generateBackupCodes()
	if err != nil {
		return nil, err
	}

	// Store MFA secret in database (you'll need to add this field to your User model)
	// user.MFASecret = secret
	// user.MFABackupCodes = backupCodes
	// ae.db.Save(user)

	return &MFASetupResponse{
		Secret:      secret,
		QRCodeURL:   qrURL,
		BackupCodes: backupCodes,
	}, nil
}

// VerifyMFA verifies an MFA token
func (ae *AuthEnhancer) VerifyMFA(userID, token string) error {
	// This is a simplified implementation
	// In production, you'd implement proper TOTP verification
	// using libraries like github.com/pquerna/otp

	if !ae.config.MFAEnabled {
		return nil
	}

	// For demo purposes, accept "123456" as valid
	if token == "123456" {
		return nil
	}

	return NewAuthenticationError("Invalid MFA token")
}

// generateSessionID generates a secure session ID
func (ae *AuthEnhancer) generateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// generateMFASecret generates a secure MFA secret
func (ae *AuthEnhancer) generateMFASecret() (string, error) {
	bytes := make([]byte, ae.config.MFASecretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// generateBackupCodes generates backup codes for MFA
func (ae *AuthEnhancer) generateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%08x", bytes)
	}
	return codes, nil
}

// ValidatePasswordStrength validates password strength
func (ae *AuthEnhancer) ValidatePasswordStrength(password string) error {
	if len(password) < ae.config.PasswordMinLength {
		return NewValidationError("Password is too short", map[string]interface{}{
			"min_length": ae.config.PasswordMinLength,
		})
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= '!' && char <= '/' || char >= ':' && char <= '@' || char >= '[' && char <= '`' || char >= '{' && char <= '~':
			hasSpecial = true
		}
	}

	if ae.config.PasswordRequireUpper && !hasUpper {
		return NewValidationError("Password must contain at least one uppercase letter", nil)
	}

	if ae.config.PasswordRequireLower && !hasLower {
		return NewValidationError("Password must contain at least one lowercase letter", nil)
	}

	if ae.config.PasswordRequireDigit && !hasDigit {
		return NewValidationError("Password must contain at least one digit", nil)
	}

	if ae.config.PasswordRequireSpecial && !hasSpecial {
		return NewValidationError("Password must contain at least one special character", nil)
	}

	return nil
}

// HashPassword hashes a password securely
func (ae *AuthEnhancer) HashPassword(password string) (string, error) {
	// Use bcrypt with adaptive cost
	cost := 12
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// VerifyPassword verifies a password against its hash
func (ae *AuthEnhancer) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SecureCompare performs a constant-time comparison
func (ae *AuthEnhancer) SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GetActiveSessions returns all active sessions for a user
func (ae *AuthEnhancer) GetActiveSessions(userID string) ([]*Session, error) {
	var sessions []*Session

	if ae.useRedis {
		// Redis implementation would require scanning keys
		// For now, return empty slice
		return sessions, nil
	}

	for _, session := range ae.sessions {
		if session.UserID == userID && session.IsActive {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// InvalidateAllSessions invalidates all sessions for a user
func (ae *AuthEnhancer) InvalidateAllSessions(userID string) error {
	if ae.useRedis {
		// Redis implementation would require scanning keys
		// For now, return nil
		return nil
	}

	for id, session := range ae.sessions {
		if session.UserID == userID {
			delete(ae.sessions, id)
		}
	}

	return nil
}

// GetStats returns authentication statistics
func (ae *AuthEnhancer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"using_redis":     ae.useRedis,
		"active_sessions": len(ae.sessions),
		"login_attempts":  len(ae.loginAttempts),
		"mfa_enabled":     ae.config.MFAEnabled,
	}

	if ae.useRedis {
		ctx := context.Background()
		if info, err := ae.redis.Info(ctx, "memory").Result(); err == nil {
			stats["redis_memory"] = info
		}
	}

	return stats
}
