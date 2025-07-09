package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/time/rate"

	"github.com/kooshapari/kodevibe-go/internal/config"
	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
)

// Server represents the HTTP API server
type Server struct {
	config    *config.Config
	registry  *vibes.Registry
	version   string
	buildTime string
	commit    string
	router    *mux.Router
	limiter   *rate.Limiter
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *APIMeta    `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta represents API metadata
type APIMeta struct {
	Version   string `json:"version"`
	RequestID string `json:"request_id,omitempty"`
	Total     int    `json:"total,omitempty"`
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
}

// ScanRequest represents a scan request
type ScanRequest struct {
	Paths        []string               `json:"paths"`
	Checkers     []string               `json:"checkers,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	OutputFormat string                 `json:"output_format,omitempty"`
	MaxIssues    int                    `json:"max_issues,omitempty"`
}

// ScanResponse represents a scan response
type ScanResponse struct {
	Issues     []models.Issue     `json:"issues"`
	Statistics ScanStatistics    `json:"statistics"`
	Duration   time.Duration     `json:"duration"`
	Metadata   ScanMetadata      `json:"metadata"`
}

// ScanStatistics represents scan statistics
type ScanStatistics struct {
	TotalFiles      int            `json:"total_files"`
	ScannedFiles    int            `json:"scanned_files"`
	TotalIssues     int            `json:"total_issues"`
	IssuesBySeverity map[string]int `json:"issues_by_severity"`
	IssuesByType    map[string]int `json:"issues_by_type"`
	IssuesByChecker map[string]int `json:"issues_by_checker"`
}

// ScanMetadata represents scan metadata
type ScanMetadata struct {
	Timestamp    time.Time          `json:"timestamp"`
	Version      string             `json:"version"`
	Checkers     []CheckerInfo      `json:"checkers"`
	Configuration map[string]interface{} `json:"configuration"`
}

// CheckerInfo represents information about a checker
type CheckerInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, registry *vibes.Registry, version, buildTime, commit string) *Server {
	server := &Server{
		config:    cfg,
		registry:  registry,
		version:   version,
		buildTime: buildTime,
		commit:    commit,
		router:    mux.NewRouter(),
	}

	// Setup rate limiter
	if cfg.Server.RateLimit.Enabled {
		server.limiter = rate.NewLimiter(
			rate.Limit(cfg.Server.RateLimit.RequestsPerMinute)/60, // per second
			cfg.Server.RateLimit.BurstSize,
		)
	}

	server.setupRoutes()
	return server
}

// Router returns the HTTP router
func (s *Server) Router() http.Handler {
	// Setup CORS if enabled
	if s.config.Server.CORS.Enabled {
		c := cors.New(cors.Options{
			AllowedOrigins:   s.config.Server.CORS.AllowedOrigins,
			AllowedMethods:   s.config.Server.CORS.AllowedMethods,
			AllowedHeaders:   s.config.Server.CORS.AllowedHeaders,
			ExposedHeaders:   s.config.Server.CORS.ExposedHeaders,
			AllowCredentials: s.config.Server.CORS.AllowCredentials,
			MaxAge:           s.config.Server.CORS.MaxAge,
		})
		return c.Handler(s.router)
	}

	return s.router
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Apply common middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.rateLimitMiddleware)
	s.router.Use(s.recoveryMiddleware)
	s.router.Use(s.contentTypeMiddleware)

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Health and info endpoints
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/info", s.handleInfo).Methods("GET")
	api.HandleFunc("/version", s.handleVersion).Methods("GET")
	
	// Checker endpoints
	api.HandleFunc("/checkers", s.handleGetCheckers).Methods("GET")
	api.HandleFunc("/checkers/{name}", s.handleGetChecker).Methods("GET")
	api.HandleFunc("/checkers/{name}/config", s.handleGetCheckerConfig).Methods("GET")
	api.HandleFunc("/checkers/{name}/config", s.handleUpdateCheckerConfig).Methods("PUT")
	
	// Scan endpoints
	api.HandleFunc("/scan", s.handleScan).Methods("POST")
	api.HandleFunc("/scan/validate", s.handleValidateScan).Methods("POST")
	
	// Configuration endpoints
	api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")
	
	// Statistics endpoints
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")

	// Serve static files for documentation (if in development mode)
	if s.config.Server.Development {
		s.router.PathPrefix("/docs/").Handler(http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs/"))))
		s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	}
}

// Success response helper
func (s *Server) successResponse(w http.ResponseWriter, data interface{}, meta *APIMeta) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now(),
	}
	
	if meta == nil {
		response.Meta = &APIMeta{Version: s.version}
	} else if response.Meta.Version == "" {
		response.Meta.Version = s.version
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Error response helper
func (s *Server) errorResponse(w http.ResponseWriter, statusCode int, code, message, details string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta:      &APIMeta{Version: s.version},
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   s.version,
		"uptime":    time.Since(time.Now()).String(), // This would be calculated from start time
		"checks": map[string]interface{}{
			"registry": map[string]interface{}{
				"status":   "healthy",
				"checkers": len(s.registry.GetAllCheckers()),
			},
		},
	}
	
	s.successResponse(w, health, nil)
}

// Info endpoint
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "KodeVibe API",
		"description": "Code analysis and quality checking API",
		"version":     s.version,
		"build_time":  s.buildTime,
		"commit":      s.commit,
		"api_version": "v1",
		"endpoints": map[string]interface{}{
			"health":   "/api/v1/health",
			"info":     "/api/v1/info",
			"version":  "/api/v1/version",
			"checkers": "/api/v1/checkers",
			"scan":     "/api/v1/scan",
			"config":   "/api/v1/config",
			"stats":    "/api/v1/stats",
		},
		"features": []string{
			"code_analysis",
			"security_scanning",
			"performance_analysis",
			"file_organization",
			"git_analysis",
			"dependency_checking",
			"documentation_analysis",
		},
	}
	
	s.successResponse(w, info, nil)
}

// Version endpoint
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":    s.version,
		"build_time": s.buildTime,
		"commit":     s.commit,
		"api_version": "v1",
	}
	
	s.successResponse(w, version, nil)
}

// Get all checkers endpoint
func (s *Server) handleGetCheckers(w http.ResponseWriter, r *http.Request) {
	checkers := s.registry.GetAllCheckers()
	checkerInfos := make([]CheckerInfo, 0, len(checkers))
	
	for _, checker := range checkers {
		info := CheckerInfo{
			Name:        checker.Name(),
			Type:        string(checker.Type()),
			Enabled:     true, // This would come from configuration
		}
		
		// Add description if the checker implements a Description method
		if desc, ok := checker.(interface{ Description() string }); ok {
			info.Description = desc.Description()
		}
		
		// Add version if the checker implements a Version method
		if ver, ok := checker.(interface{ Version() string }); ok {
			info.Version = ver.Version()
		}
		
		checkerInfos = append(checkerInfos, info)
	}
	
	s.successResponse(w, checkerInfos, &APIMeta{Total: len(checkerInfos)})
}

// Get specific checker endpoint
func (s *Server) handleGetChecker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	checker, err := s.registry.GetChecker(models.VibeType(name))
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "CHECKER_NOT_FOUND", 
			fmt.Sprintf("Checker '%s' not found", name), err.Error())
		return
	}
	
	info := CheckerInfo{
		Name:    checker.Name(),
		Type:    string(checker.Type()),
		Enabled: true,
	}
	
	if desc, ok := checker.(interface{ Description() string }); ok {
		info.Description = desc.Description()
	}
	
	if ver, ok := checker.(interface{ Version() string }); ok {
		info.Version = ver.Version()
	}
	
	s.successResponse(w, info, nil)
}

// Get checker configuration endpoint
func (s *Server) handleGetCheckerConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	_, err := s.registry.GetChecker(models.VibeType(name))
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "CHECKER_NOT_FOUND", 
			fmt.Sprintf("Checker '%s' not found", name), err.Error())
		return
	}
	
	// Get configuration from config
	var config interface{}
	if checkerConfig, exists := s.config.Vibes.CheckerConfigs[name]; exists {
		config = checkerConfig
	} else {
		config = map[string]interface{}{"enabled": true}
	}
	
	s.successResponse(w, config, nil)
}

// Update checker configuration endpoint
func (s *Server) handleUpdateCheckerConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	_, err := s.registry.GetChecker(models.VibeType(name))
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "CHECKER_NOT_FOUND", 
			fmt.Sprintf("Checker '%s' not found", name), err.Error())
		return
	}
	
	var newConfig config.CheckerConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_JSON", 
			"Invalid JSON in request body", err.Error())
		return
	}
	
	// Update configuration
	s.config.Vibes.CheckerConfigs[name] = newConfig
	
	s.successResponse(w, map[string]string{"message": "Configuration updated successfully"}, nil)
}

// Get configuration endpoint
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.successResponse(w, s.config, nil)
}

// Update configuration endpoint
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_JSON", 
			"Invalid JSON in request body", err.Error())
		return
	}
	
	if err := newConfig.Validate(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_CONFIG", 
			"Invalid configuration", err.Error())
		return
	}
	
	// Update configuration (in a real implementation, you'd want to be more careful about this)
	*s.config = newConfig
	
	s.successResponse(w, map[string]string{"message": "Configuration updated successfully"}, nil)
}

// Get statistics endpoint
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"checkers": map[string]interface{}{
			"total":     len(s.registry.GetAllCheckers()),
			"enabled":   len(s.config.Vibes.EnabledCheckers),
			"available": s.registry.ListAvailableVibes(),
		},
		"configuration": map[string]interface{}{
			"max_file_size": s.config.Scanner.MaxFileSize,
			"max_files":     s.config.Scanner.MaxFiles,
			"max_issues":    s.config.Vibes.MaxIssues,
			"output_format": s.config.Vibes.OutputFormat,
		},
		"runtime": map[string]interface{}{
			"version":    s.version,
			"build_time": s.buildTime,
			"commit":     s.commit,
		},
	}
	
	s.successResponse(w, stats, nil)
}

// Root endpoint for development
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>KodeVibe API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { color: #333; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: #007acc; }
    </style>
</head>
<body>
    <h1 class="header">KodeVibe API Server</h1>
    <p>Version: %s | Build: %s | Commit: %s</p>
    
    <h2>Available Endpoints:</h2>
    <div class="endpoint"><span class="method">GET</span> /api/v1/health - Health check</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/info - Server information</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/version - Version information</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/checkers - List all checkers</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/checkers/{name} - Get specific checker</div>
    <div class="endpoint"><span class="method">POST</span> /api/v1/scan - Perform code scan</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/config - Get configuration</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/stats - Get statistics</div>
    
    <h2>Documentation:</h2>
    <p><a href="/docs/">API Documentation</a></p>
</body>
</html>
`, s.version, s.buildTime, s.commit)
	
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}