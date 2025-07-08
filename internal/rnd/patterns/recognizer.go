package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
)

// PatternType represents different types of patterns
type PatternType string

const (
	PatternTypeUserBehavior  PatternType = "user_behavior"
	PatternTypeSystemUsage   PatternType = "system_usage"
	PatternTypeProjectTrend  PatternType = "project_trend"
	PatternTypeTechAdoption  PatternType = "tech_adoption"
	PatternTypeWorkflow      PatternType = "workflow"
	PatternTypeTemporal      PatternType = "temporal"
	PatternTypeCommand       PatternType = "command"
	PatternTypePerformance   PatternType = "performance"
	PatternTypeCollaboration PatternType = "collaboration"
	PatternTypeResource      PatternType = "resource"
)

// Pattern represents a recognized pattern with metadata
type Pattern struct {
	ID              string                 `json:"id"`
	Type            PatternType            `json:"type"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Confidence      float64                `json:"confidence"`
	Frequency       int                    `json:"frequency"`
	LastSeen        time.Time              `json:"last_seen"`
	FirstSeen       time.Time              `json:"first_seen"`
	Age             time.Duration          `json:"age"`
	Correlations    []string               `json:"correlations"`
	Features        map[string]float64     `json:"features"`
	Metadata        map[string]interface{} `json:"metadata"`
	IsAnomaly       bool                   `json:"is_anomaly"`
	PredictiveValue float64                `json:"predictive_value"`
}

// PatternMatcher defines interface for pattern matching algorithms
type PatternMatcher interface {
	Match(data []float64) (float64, error)
	Update(data []float64, feedback float64) error
	Features() map[string]float64
}

// TimeSeriesPattern represents temporal patterns
type TimeSeriesPattern struct {
	Sequence    []float64 `json:"sequence"`
	Period      int       `json:"period"`
	Amplitude   float64   `json:"amplitude"`
	Phase       float64   `json:"phase"`
	Trend       float64   `json:"trend"`
	Seasonality float64   `json:"seasonality"`
}

// CorrelationMatrix represents pattern correlations
type CorrelationMatrix struct {
	Patterns  []string    `json:"patterns"`
	Matrix    [][]float64 `json:"matrix"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Recognizer handles pattern recognition and analysis
type Recognizer struct {
	db               *gorm.DB
	logger           *logrus.Logger
	patterns         map[string]*Pattern
	matchers         map[PatternType][]PatternMatcher
	correlations     *CorrelationMatrix
	anomalyThreshold float64
	confidenceMin    float64
	agingFactor      float64
	maxPatterns      int
	mutex            sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	workerCount      int

	// Processing channels
	dataChannel   chan *DataPoint
	resultChannel chan *ProcessingResult

	// Statistics
	stats *RecognizerStats
}

// DataPoint represents input data for pattern recognition
type DataPoint struct {
	ID        string                 `json:"id"`
	Type      PatternType            `json:"type"`
	Features  map[string]float64     `json:"features"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// ProcessingResult represents pattern recognition results
type ProcessingResult struct {
	DataPoint      *DataPoint    `json:"data_point"`
	MatchedPattern *Pattern      `json:"matched_pattern,omitempty"`
	Confidence     float64       `json:"confidence"`
	IsNovelty      bool          `json:"is_novelty"`
	Anomaly        bool          `json:"anomaly"`
	Correlations   []string      `json:"correlations"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// RecognizerStats tracks recognition statistics
type RecognizerStats struct {
	PatternsRecognized int64     `json:"patterns_recognized"`
	NovelPatternsFound int64     `json:"novel_patterns_found"`
	AnomaliesDetected  int64     `json:"anomalies_detected"`
	ProcessingTime     float64   `json:"avg_processing_time_ms"`
	ConfidenceAvg      float64   `json:"avg_confidence"`
	LastUpdate         time.Time `json:"last_update"`
	TotalDataPoints    int64     `json:"total_data_points"`
}

// RecognizerConfig holds configuration for the pattern recognizer
type RecognizerConfig struct {
	AnomalyThreshold float64 `mapstructure:"anomaly_threshold" json:"anomaly_threshold"`
	ConfidenceMin    float64 `mapstructure:"confidence_min" json:"confidence_min"`
	AgingFactor      float64 `mapstructure:"aging_factor" json:"aging_factor"`
	MaxPatterns      int     `mapstructure:"max_patterns" json:"max_patterns"`
	WorkerCount      int     `mapstructure:"worker_count" json:"worker_count"`
	ProcessingBuffer int     `mapstructure:"processing_buffer" json:"processing_buffer"`
}

// NewRecognizer creates a new pattern recognizer
func NewRecognizer(db *gorm.DB, logger *logrus.Logger, config *RecognizerConfig) *Recognizer {
	ctx, cancel := context.WithCancel(context.Background())

	if config == nil {
		config = &RecognizerConfig{
			AnomalyThreshold: 0.8,
			ConfidenceMin:    0.6,
			AgingFactor:      0.95,
			MaxPatterns:      1000,
			WorkerCount:      4,
			ProcessingBuffer: 100,
		}
	}

	return &Recognizer{
		db:               db,
		logger:           logger,
		patterns:         make(map[string]*Pattern),
		matchers:         make(map[PatternType][]PatternMatcher),
		anomalyThreshold: config.AnomalyThreshold,
		confidenceMin:    config.ConfidenceMin,
		agingFactor:      config.AgingFactor,
		maxPatterns:      config.MaxPatterns,
		ctx:              ctx,
		cancel:           cancel,
		workerCount:      config.WorkerCount,
		dataChannel:      make(chan *DataPoint, config.ProcessingBuffer),
		resultChannel:    make(chan *ProcessingResult, config.ProcessingBuffer),
		stats:            &RecognizerStats{},
	}
}

// Start begins pattern recognition processing
func (r *Recognizer) Start() error {
	r.logger.Info("Starting Pattern Recognizer")

	// Initialize correlations matrix
	r.correlations = &CorrelationMatrix{
		Patterns:  make([]string, 0),
		Matrix:    make([][]float64, 0),
		UpdatedAt: time.Now(),
	}

	// Load existing patterns from database
	if err := r.loadPatterns(); err != nil {
		r.logger.WithError(err).Warn("Failed to load existing patterns")
	}

	// Start worker goroutines
	for i := 0; i < r.workerCount; i++ {
		go r.processingWorker()
	}

	// Start pattern aging routine
	go r.patternAgingRoutine()

	// Start correlation update routine
	go r.correlationUpdateRoutine()

	// Start statistics update routine
	go r.statsUpdateRoutine()

	r.logger.WithField("worker_count", r.workerCount).Info("Pattern Recognizer started")
	return nil
}

// Stop shuts down the pattern recognizer
func (r *Recognizer) Stop() error {
	r.logger.Info("Stopping Pattern Recognizer")

	r.cancel()
	close(r.dataChannel)

	// Save patterns to database
	if err := r.savePatterns(); err != nil {
		r.logger.WithError(err).Error("Failed to save patterns")
	}

	r.logger.Info("Pattern Recognizer stopped")
	return nil
}

// ProcessDataPoint processes a single data point for pattern recognition
func (r *Recognizer) ProcessDataPoint(data *DataPoint) (*ProcessingResult, error) {
	startTime := time.Now()

	// Add to processing queue
	select {
	case r.dataChannel <- data:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("processing queue full")
	}

	// Wait for result
	select {
	case result := <-r.resultChannel:
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("processing timeout")
	}
}

// processingWorker handles pattern recognition processing
func (r *Recognizer) processingWorker() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case data := <-r.dataChannel:
			if data == nil {
				return // Channel closed
			}

			result := r.processData(data)

			select {
			case r.resultChannel <- result:
			case <-time.After(1 * time.Second):
				r.logger.Warn("Result channel full, dropping result")
			}
		}
	}
}

// processData performs the actual pattern recognition
func (r *Recognizer) processData(data *DataPoint) *ProcessingResult {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	result := &ProcessingResult{
		DataPoint:    data,
		Confidence:   0.0,
		IsNovelty:    false,
		Anomaly:      false,
		Correlations: make([]string, 0),
	}

	// Find best matching pattern
	bestMatch, maxConfidence := r.findBestMatch(data)

	if bestMatch != nil && maxConfidence >= r.confidenceMin {
		// Update existing pattern
		result.MatchedPattern = bestMatch
		result.Confidence = maxConfidence
		r.updatePattern(bestMatch, data, maxConfidence)
	} else if maxConfidence < r.confidenceMin {
		// Create new pattern for novel data
		newPattern := r.createNewPattern(data)
		if newPattern != nil {
			result.MatchedPattern = newPattern
			result.IsNovelty = true
			result.Confidence = 1.0
			r.patterns[newPattern.ID] = newPattern
		}
	}

	// Check for anomalies
	if maxConfidence < r.anomalyThreshold && !result.IsNovelty {
		result.Anomaly = true
		r.stats.AnomaliesDetected++
	}

	// Update correlations
	if result.MatchedPattern != nil {
		result.Correlations = r.findCorrelations(result.MatchedPattern)
	}

	// Update statistics
	r.stats.TotalDataPoints++
	if result.MatchedPattern != nil {
		r.stats.PatternsRecognized++
		if result.IsNovelty {
			r.stats.NovelPatternsFound++
		}
	}

	return result
}

// findBestMatch finds the best matching pattern for given data
func (r *Recognizer) findBestMatch(data *DataPoint) (*Pattern, float64) {
	var bestPattern *Pattern
	maxConfidence := 0.0

	for _, pattern := range r.patterns {
		if pattern.Type != data.Type {
			continue
		}

		confidence := r.calculateSimilarity(pattern.Features, data.Features)

		// Apply age decay to confidence
		age := time.Since(pattern.LastSeen)
		decayFactor := math.Pow(r.agingFactor, age.Hours()/24.0)
		confidence *= decayFactor

		if confidence > maxConfidence {
			maxConfidence = confidence
			bestPattern = pattern
		}
	}

	return bestPattern, maxConfidence
}

// calculateSimilarity computes similarity between feature vectors
func (r *Recognizer) calculateSimilarity(features1, features2 map[string]float64) float64 {
	// Cosine similarity
	dotProduct := 0.0
	magnitude1 := 0.0
	magnitude2 := 0.0

	// Get all unique keys
	allKeys := make(map[string]bool)
	for k := range features1 {
		allKeys[k] = true
	}
	for k := range features2 {
		allKeys[k] = true
	}

	for key := range allKeys {
		val1, exists1 := features1[key]
		val2, exists2 := features2[key]

		if !exists1 {
			val1 = 0.0
		}
		if !exists2 {
			val2 = 0.0
		}

		dotProduct += val1 * val2
		magnitude1 += val1 * val1
		magnitude2 += val2 * val2
	}

	if magnitude1 == 0.0 || magnitude2 == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(magnitude1) * math.Sqrt(magnitude2))
}

// createNewPattern creates a new pattern from data
func (r *Recognizer) createNewPattern(data *DataPoint) *Pattern {
	// Check if we've reached max patterns
	if len(r.patterns) >= r.maxPatterns {
		r.evictOldestPattern()
	}

	patternID := fmt.Sprintf("%s_%d", data.Type, time.Now().UnixNano())

	pattern := &Pattern{
		ID:              patternID,
		Type:            data.Type,
		Name:            fmt.Sprintf("Pattern_%s", patternID),
		Description:     r.generatePatternDescription(data),
		Confidence:      1.0,
		Frequency:       1,
		LastSeen:        data.Timestamp,
		FirstSeen:       data.Timestamp,
		Age:             0,
		Correlations:    make([]string, 0),
		Features:        make(map[string]float64),
		Metadata:        data.Metadata,
		IsAnomaly:       false,
		PredictiveValue: 0.5,
	}

	// Copy features
	for k, v := range data.Features {
		pattern.Features[k] = v
	}

	return pattern
}

// updatePattern updates an existing pattern with new data
func (r *Recognizer) updatePattern(pattern *Pattern, data *DataPoint, confidence float64) {
	pattern.Frequency++
	pattern.LastSeen = data.Timestamp
	pattern.Age = time.Since(pattern.FirstSeen)
	pattern.Confidence = (pattern.Confidence + confidence) / 2.0

	// Update features with weighted average
	weight := 1.0 / float64(pattern.Frequency)
	for key, newValue := range data.Features {
		if oldValue, exists := pattern.Features[key]; exists {
			pattern.Features[key] = oldValue*(1-weight) + newValue*weight
		} else {
			pattern.Features[key] = newValue
		}
	}

	// Update metadata
	if pattern.Metadata == nil {
		pattern.Metadata = make(map[string]interface{})
	}
	for k, v := range data.Metadata {
		pattern.Metadata[k] = v
	}
}

// generatePatternDescription creates a description for a pattern
func (r *Recognizer) generatePatternDescription(data *DataPoint) string {
	switch data.Type {
	case PatternTypeUserBehavior:
		return "User behavior pattern detected"
	case PatternTypeSystemUsage:
		return "System usage pattern identified"
	case PatternTypeProjectTrend:
		return "Project trend pattern discovered"
	case PatternTypeTechAdoption:
		return "Technology adoption pattern found"
	case PatternTypeWorkflow:
		return "Workflow pattern recognized"
	case PatternTypeTemporal:
		return "Temporal pattern detected"
	case PatternTypeCommand:
		return "Command usage pattern identified"
	case PatternTypePerformance:
		return "Performance pattern discovered"
	case PatternTypeCollaboration:
		return "Collaboration pattern found"
	case PatternTypeResource:
		return "Resource usage pattern detected"
	default:
		return "Unknown pattern type detected"
	}
}

// evictOldestPattern removes the oldest pattern to make room for new ones
func (r *Recognizer) evictOldestPattern() {
	var oldestID string
	var oldestTime time.Time = time.Now()

	for id, pattern := range r.patterns {
		if pattern.LastSeen.Before(oldestTime) {
			oldestTime = pattern.LastSeen
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(r.patterns, oldestID)
		r.logger.WithField("pattern_id", oldestID).Debug("Evicted oldest pattern")
	}
}

// findCorrelations finds patterns correlated with the given pattern
func (r *Recognizer) findCorrelations(pattern *Pattern) []string {
	correlations := make([]string, 0)

	if r.correlations == nil {
		return correlations
	}

	patternIndex := -1
	for i, p := range r.correlations.Patterns {
		if p == pattern.ID {
			patternIndex = i
			break
		}
	}

	if patternIndex == -1 {
		return correlations
	}

	// Find highly correlated patterns (correlation > 0.7)
	for i, correlation := range r.correlations.Matrix[patternIndex] {
		if i != patternIndex && correlation > 0.7 {
			correlations = append(correlations, r.correlations.Patterns[i])
		}
	}

	return correlations
}

// patternAgingRoutine applies aging to patterns periodically
func (r *Recognizer) patternAgingRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.agePatterns()
		}
	}
}

// agePatterns applies aging factor to all patterns
func (r *Recognizer) agePatterns() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for _, pattern := range r.patterns {
		age := now.Sub(pattern.LastSeen)
		ageFactor := math.Pow(r.agingFactor, age.Hours()/24.0)
		pattern.Confidence *= ageFactor
		pattern.Age = age

		// Remove patterns with very low confidence
		if pattern.Confidence < 0.1 {
			delete(r.patterns, pattern.ID)
		}
	}

	r.logger.WithField("pattern_count", len(r.patterns)).Debug("Applied pattern aging")
}

// correlationUpdateRoutine updates pattern correlations periodically
func (r *Recognizer) correlationUpdateRoutine() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.updateCorrelations()
		}
	}
}

// updateCorrelations recalculates pattern correlations
func (r *Recognizer) updateCorrelations() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	patterns := make([]*Pattern, 0, len(r.patterns))
	patternIDs := make([]string, 0, len(r.patterns))

	for id, pattern := range r.patterns {
		patterns = append(patterns, pattern)
		patternIDs = append(patternIDs, id)
	}

	n := len(patterns)
	if n < 2 {
		return
	}

	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	// Calculate correlations between all pattern pairs
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else {
				correlation := r.calculateSimilarity(patterns[i].Features, patterns[j].Features)
				matrix[i][j] = correlation
			}
		}
	}

	r.correlations = &CorrelationMatrix{
		Patterns:  patternIDs,
		Matrix:    matrix,
		UpdatedAt: time.Now(),
	}

	r.logger.WithField("pattern_count", n).Debug("Updated pattern correlations")
}

// statsUpdateRoutine updates statistics periodically
func (r *Recognizer) statsUpdateRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.updateStats()
		}
	}
}

// updateStats updates recognition statistics
func (r *Recognizer) updateStats() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.stats.LastUpdate = time.Now()

	// Calculate average confidence
	totalConfidence := 0.0
	for _, pattern := range r.patterns {
		totalConfidence += pattern.Confidence
	}

	if len(r.patterns) > 0 {
		r.stats.ConfidenceAvg = totalConfidence / float64(len(r.patterns))
	}
}

// GetStats returns current recognition statistics
func (r *Recognizer) GetStats() *RecognizerStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := *r.stats // Copy
	return &stats
}

// GetPatterns returns all recognized patterns
func (r *Recognizer) GetPatterns() map[string]*Pattern {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	patterns := make(map[string]*Pattern)
	for k, v := range r.patterns {
		patterns[k] = v
	}
	return patterns
}

// GetPatternsByType returns patterns of a specific type
func (r *Recognizer) GetPatternsByType(patternType PatternType) []*Pattern {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	patterns := make([]*Pattern, 0)
	for _, pattern := range r.patterns {
		if pattern.Type == patternType {
			patterns = append(patterns, pattern)
		}
	}

	// Sort by confidence
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	return patterns
}

// GetCorrelations returns the correlation matrix
func (r *Recognizer) GetCorrelations() *CorrelationMatrix {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.correlations == nil {
		return nil
	}

	// Deep copy
	matrix := make([][]float64, len(r.correlations.Matrix))
	for i, row := range r.correlations.Matrix {
		matrix[i] = make([]float64, len(row))
		copy(matrix[i], row)
	}

	patterns := make([]string, len(r.correlations.Patterns))
	copy(patterns, r.correlations.Patterns)

	return &CorrelationMatrix{
		Patterns:  patterns,
		Matrix:    matrix,
		UpdatedAt: r.correlations.UpdatedAt,
	}
}

// loadPatterns loads patterns from database
func (r *Recognizer) loadPatterns() error {
	var dbPatterns []models.Pattern
	if err := r.db.Find(&dbPatterns).Error; err != nil {
		return err
	}

	for _, dbPattern := range dbPatterns {
		pattern := &Pattern{
			ID:              dbPattern.ID.String(),
			Type:            PatternType(dbPattern.Type),
			Name:            dbPattern.Name,
			Description:     dbPattern.Description,
			Confidence:      dbPattern.Confidence,
			Frequency:       dbPattern.Frequency,
			LastSeen:        dbPattern.LastSeen,
			FirstSeen:       dbPattern.CreatedAt, // Use CreatedAt as FirstSeen
			Age:             time.Since(dbPattern.CreatedAt),
			Correlations:    make([]string, 0),
			Features:        make(map[string]float64),
			Metadata:        make(map[string]interface{}),
			IsAnomaly:       false,                  // Default value
			PredictiveValue: dbPattern.Significance, // Use Significance as PredictiveValue
		}

		// Parse features JSON from Data field
		if dbPattern.Data != "" {
			if err := json.Unmarshal([]byte(dbPattern.Data), &pattern.Features); err != nil {
				r.logger.WithError(err).Warn("Failed to parse pattern features from data")
			}
		}

		// Parse metadata JSON from Context field
		if dbPattern.Context != "" {
			if err := json.Unmarshal([]byte(dbPattern.Context), &pattern.Metadata); err != nil {
				r.logger.WithError(err).Warn("Failed to parse pattern metadata from context")
			}
		}

		r.patterns[pattern.ID] = pattern
	}

	r.logger.WithField("pattern_count", len(r.patterns)).Info("Loaded patterns from database")
	return nil
}

// savePatterns saves patterns to database
func (r *Recognizer) savePatterns() error {
	for _, pattern := range r.patterns {
		// Convert to database model
		dbPattern := models.Pattern{
			Name:         pattern.Name,
			Type:         string(pattern.Type),
			Description:  pattern.Description,
			Confidence:   pattern.Confidence,
			Frequency:    pattern.Frequency,
			LastSeen:     pattern.LastSeen,
			Significance: pattern.PredictiveValue, // Map PredictiveValue to Significance
		}

		// Serialize features to Data field
		if featuresJSON, err := json.Marshal(pattern.Features); err == nil {
			dbPattern.Data = string(featuresJSON)
		}

		// Serialize metadata to Context field
		if metadataJSON, err := json.Marshal(pattern.Metadata); err == nil {
			dbPattern.Context = string(metadataJSON)
		}

		// Save to database
		if err := r.db.Save(&dbPattern).Error; err != nil {
			r.logger.WithError(err).WithField("pattern_id", pattern.ID).Error("Failed to save pattern")
		}
	}

	r.logger.WithField("pattern_count", len(r.patterns)).Info("Saved patterns to database")
	return nil
}

// Health returns the health status of the pattern recognizer
func (r *Recognizer) Health() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"status":                  "healthy",
		"pattern_count":           len(r.patterns),
		"worker_count":            r.workerCount,
		"queue_size":              len(r.dataChannel),
		"result_queue":            len(r.resultChannel),
		"stats":                   r.stats,
		"last_correlation_update": r.correlations.UpdatedAt,
	}
}
