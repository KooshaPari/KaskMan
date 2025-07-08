package projects

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/patterns"
)

// ProjectCategory represents different project categories
type ProjectCategory string

const (
	CategoryAutomation     ProjectCategory = "automation"
	CategoryOptimization   ProjectCategory = "optimization"
	CategoryInnovation     ProjectCategory = "innovation"
	CategoryIntegration    ProjectCategory = "integration"
	CategoryMaintenance    ProjectCategory = "maintenance"
	CategoryResearch       ProjectCategory = "research"
	CategorySecurity       ProjectCategory = "security"
	CategoryPerformance    ProjectCategory = "performance"
	CategoryDataAnalysis   ProjectCategory = "data_analysis"
	CategoryInfrastructure ProjectCategory = "infrastructure"
)

// ProjectComplexity represents project complexity levels
type ProjectComplexity string

const (
	ComplexityLow    ProjectComplexity = "low"
	ComplexityMedium ProjectComplexity = "medium"
	ComplexityHigh   ProjectComplexity = "high"
	ComplexityExpert ProjectComplexity = "expert"
)

// ProjectTemplate represents a project template
type ProjectTemplate struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Category             ProjectCategory        `json:"category"`
	Description          string                 `json:"description"`
	Complexity           ProjectComplexity      `json:"complexity"`
	EstimatedHours       int                    `json:"estimated_hours"`
	RequiredSkills       []string               `json:"required_skills"`
	RequiredTechnologies []string               `json:"required_technologies"`
	ExpectedOutcomes     []string               `json:"expected_outcomes"`
	Prerequisites        []string               `json:"prerequisites"`
	RiskFactors          []string               `json:"risk_factors"`
	SuccessMetrics       []string               `json:"success_metrics"`
	Tags                 []string               `json:"tags"`
	Priority             string                 `json:"priority"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// GeneratedProject represents a generated project proposal
type GeneratedProject struct {
	ID                   string                 `json:"id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	Category             ProjectCategory        `json:"category"`
	Complexity           ProjectComplexity      `json:"complexity"`
	EstimatedEffort      int                    `json:"estimated_effort"`
	EstimatedDuration    int                    `json:"estimated_duration_days"`
	Priority             string                 `json:"priority"`
	ImpactScore          float64                `json:"impact_score"`
	FeasibilityScore     float64                `json:"feasibility_score"`
	AlignmentScore       float64                `json:"alignment_score"`
	InnovationScore      float64                `json:"innovation_score"`
	PracticalityScore    float64                `json:"practicality_score"`
	OverallScore         float64                `json:"overall_score"`
	ConfidenceLevel      float64                `json:"confidence_level"`
	RequiredSkills       []string               `json:"required_skills"`
	RequiredTechnologies []string               `json:"required_technologies"`
	Prerequisites        []string               `json:"prerequisites"`
	ExpectedOutcomes     []string               `json:"expected_outcomes"`
	RiskFactors          []RiskFactor           `json:"risk_factors"`
	Justification        string                 `json:"justification"`
	RecommendedTeamSize  int                    `json:"recommended_team_size"`
	EstimatedROI         float64                `json:"estimated_roi"`
	Milestones           []ProjectMilestone     `json:"milestones"`
	ResourceRequirements ResourceRequirements   `json:"resource_requirements"`
	GeneratedAt          time.Time              `json:"generated_at"`
	BasedOnPatterns      []string               `json:"based_on_patterns"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// RiskFactor represents a project risk
type RiskFactor struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Impact      float64 `json:"impact"`
	Mitigation  string  `json:"mitigation"`
}

// ProjectMilestone represents a project milestone
type ProjectMilestone struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	DaysFromStart int      `json:"days_from_start"`
	Deliverables  []string `json:"deliverables"`
}

// ResourceRequirements represents project resource needs
type ResourceRequirements struct {
	DeveloperHours   int      `json:"developer_hours"`
	QAHours          int      `json:"qa_hours"`
	DevOpsHours      int      `json:"devops_hours"`
	DesignHours      int      `json:"design_hours"`
	RequiredTools    []string `json:"required_tools"`
	ExternalServices []string `json:"external_services"`
	Budget           float64  `json:"budget"`
	CloudResources   []string `json:"cloud_resources"`
}

// OpportunityAnalysis represents an identified opportunity
type OpportunityAnalysis struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Description    string                 `json:"description"`
	ImpactLevel    float64                `json:"impact_level"`
	EffortRequired float64                `json:"effort_required"`
	Urgency        float64                `json:"urgency"`
	StrategicValue float64                `json:"strategic_value"`
	DataSources    []string               `json:"data_sources"`
	SupportingData map[string]interface{} `json:"supporting_data"`
	IdentifiedAt   time.Time              `json:"identified_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
}

// Generator handles intelligent project generation
type Generator struct {
	db                *gorm.DB
	logger            *logrus.Logger
	patternRecognizer *patterns.Recognizer
	templates         map[string]*ProjectTemplate
	opportunities     map[string]*OpportunityAnalysis
	diversityEngine   *DiversityEngine
	scoringEngine     *ScoringEngine
	riskAssessor      *RiskAssessor

	// Configuration
	maxSuggestions  int
	diversityWeight float64
	noveltyWeight   float64
	alignmentWeight float64

	// Concurrency
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	// Statistics
	stats *GeneratorStats
}

// GeneratorStats tracks generation statistics
type GeneratorStats struct {
	ProjectsGenerated  int64     `json:"projects_generated"`
	OpportunitiesFound int64     `json:"opportunities_found"`
	TemplatesUsed      int64     `json:"templates_used"`
	AverageScore       float64   `json:"average_score"`
	AcceptanceRate     float64   `json:"acceptance_rate"`
	LastGeneration     time.Time `json:"last_generation"`
	ProcessingTimeMs   float64   `json:"avg_processing_time_ms"`
}

// GeneratorConfig holds configuration for the project generator
type GeneratorConfig struct {
	MaxSuggestions       int     `mapstructure:"max_suggestions" json:"max_suggestions"`
	DiversityWeight      float64 `mapstructure:"diversity_weight" json:"diversity_weight"`
	NoveltyWeight        float64 `mapstructure:"novelty_weight" json:"novelty_weight"`
	AlignmentWeight      float64 `mapstructure:"alignment_weight" json:"alignment_weight"`
	MinConfidenceLevel   float64 `mapstructure:"min_confidence_level" json:"min_confidence_level"`
	MaxComplexity        string  `mapstructure:"max_complexity" json:"max_complexity"`
	RefreshIntervalHours int     `mapstructure:"refresh_interval_hours" json:"refresh_interval_hours"`
}

// DiversityEngine ensures diverse project suggestions
type DiversityEngine struct {
	categoryDistribution   map[ProjectCategory]float64
	complexityDistribution map[ProjectComplexity]float64
	recentSuggestions      []string
	maxRecent              int
}

// ScoringEngine evaluates project proposals
type ScoringEngine struct {
	impactWeights      map[string]float64
	feasibilityFactors map[string]float64
	alignmentCriteria  map[string]float64
	innovationMetrics  map[string]float64
}

// RiskAssessor evaluates project risks
type RiskAssessor struct {
	riskFactors          map[string]float64
	mitigationStrategies map[string]string
	complexityRisks      map[ProjectComplexity]float64
}

// NewGenerator creates a new project generator
func NewGenerator(db *gorm.DB, logger *logrus.Logger, patternRecognizer *patterns.Recognizer, config *GeneratorConfig) *Generator {
	ctx, cancel := context.WithCancel(context.Background())

	if config == nil {
		config = &GeneratorConfig{
			MaxSuggestions:       10,
			DiversityWeight:      0.3,
			NoveltyWeight:        0.2,
			AlignmentWeight:      0.4,
			MinConfidenceLevel:   0.6,
			MaxComplexity:        "high",
			RefreshIntervalHours: 24,
		}
	}

	generator := &Generator{
		db:                db,
		logger:            logger,
		patternRecognizer: patternRecognizer,
		templates:         make(map[string]*ProjectTemplate),
		opportunities:     make(map[string]*OpportunityAnalysis),
		maxSuggestions:    config.MaxSuggestions,
		diversityWeight:   config.DiversityWeight,
		noveltyWeight:     config.NoveltyWeight,
		alignmentWeight:   config.AlignmentWeight,
		ctx:               ctx,
		cancel:            cancel,
		stats:             &GeneratorStats{},
	}

	// Initialize engines
	generator.diversityEngine = &DiversityEngine{
		categoryDistribution:   make(map[ProjectCategory]float64),
		complexityDistribution: make(map[ProjectComplexity]float64),
		recentSuggestions:      make([]string, 0),
		maxRecent:              50,
	}

	generator.scoringEngine = &ScoringEngine{
		impactWeights: map[string]float64{
			"user_productivity":    0.25,
			"system_efficiency":    0.25,
			"cost_reduction":       0.20,
			"quality_improvement":  0.15,
			"innovation_potential": 0.15,
		},
		feasibilityFactors: map[string]float64{
			"technical_complexity":  0.30,
			"resource_availability": 0.25,
			"time_constraints":      0.20,
			"skill_requirements":    0.15,
			"external_dependencies": 0.10,
		},
		alignmentCriteria: map[string]float64{
			"strategic_goals":       0.30,
			"business_priorities":   0.25,
			"current_capabilities":  0.20,
			"market_trends":         0.15,
			"competitive_advantage": 0.10,
		},
		innovationMetrics: map[string]float64{
			"technology_novelty":     0.30,
			"approach_uniqueness":    0.25,
			"market_differentiation": 0.20,
			"learning_opportunity":   0.15,
			"future_potential":       0.10,
		},
	}

	generator.riskAssessor = &RiskAssessor{
		riskFactors: map[string]float64{
			"technical_risk":   0.25,
			"resource_risk":    0.20,
			"timeline_risk":    0.20,
			"integration_risk": 0.15,
			"market_risk":      0.10,
			"regulatory_risk":  0.10,
		},
		mitigationStrategies: map[string]string{
			"technical_risk":   "Conduct proof of concept, use proven technologies",
			"resource_risk":    "Secure resource commitments, plan for contingencies",
			"timeline_risk":    "Add buffer time, implement in phases",
			"integration_risk": "Early integration testing, define clear interfaces",
			"market_risk":      "Market research, user feedback loops",
			"regulatory_risk":  "Early compliance review, legal consultation",
		},
		complexityRisks: map[ProjectComplexity]float64{
			ComplexityLow:    0.1,
			ComplexityMedium: 0.3,
			ComplexityHigh:   0.6,
			ComplexityExpert: 0.8,
		},
	}

	return generator
}

// Start initializes the project generator
func (g *Generator) Start() error {
	g.logger.Info("Starting Project Generator")

	// Load project templates
	if err := g.loadTemplates(); err != nil {
		g.logger.WithError(err).Warn("Failed to load project templates")
	}

	// Initialize default templates if none exist
	if len(g.templates) == 0 {
		g.initializeDefaultTemplates()
	}

	// Start opportunity analysis routine
	go g.opportunityAnalysisRoutine()

	// Start statistics update routine
	go g.statsUpdateRoutine()

	g.logger.WithField("template_count", len(g.templates)).Info("Project Generator started")
	return nil
}

// Stop shuts down the project generator
func (g *Generator) Stop() error {
	g.logger.Info("Stopping Project Generator")
	g.cancel()
	return nil
}

// GenerateProjectSuggestions generates project suggestions based on current analysis
func (g *Generator) GenerateProjectSuggestions(userID string, preferences *GenerationPreferences) ([]*GeneratedProject, error) {
	startTime := time.Now()
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if preferences == nil {
		preferences = &GenerationPreferences{
			Categories:        []ProjectCategory{},
			MaxComplexity:     ComplexityHigh,
			MinImpactScore:    0.6,
			FocusAreas:        []string{},
			ExcludeCategories: []ProjectCategory{},
		}
	}

	// Analyze current opportunities
	opportunities := g.analyzeOpportunities()

	// Generate project ideas
	ideas := g.generateIdeas(opportunities, preferences)

	// Score and rank ideas
	scoredProjects := g.scoreProjects(ideas)

	// Apply diversity filtering
	diverseProjects := g.diversityEngine.filterForDiversity(scoredProjects, g.maxSuggestions)

	// Generate detailed project proposals
	suggestions := make([]*GeneratedProject, 0, len(diverseProjects))
	for _, project := range diverseProjects {
		if project.OverallScore >= preferences.MinImpactScore {
			detailed := g.generateDetailedProject(project, opportunities)
			suggestions = append(suggestions, detailed)
		}
	}

	// Update statistics
	g.stats.ProjectsGenerated += int64(len(suggestions))
	g.stats.LastGeneration = time.Now()
	g.stats.ProcessingTimeMs = float64(time.Since(startTime).Nanoseconds()) / 1000000.0

	if len(suggestions) > 0 {
		totalScore := 0.0
		for _, suggestion := range suggestions {
			totalScore += suggestion.OverallScore
		}
		g.stats.AverageScore = totalScore / float64(len(suggestions))
	}

	g.logger.WithFields(logrus.Fields{
		"suggestions_count": len(suggestions),
		"processing_time":   time.Since(startTime),
		"user_id":           userID,
	}).Info("Generated project suggestions")

	return suggestions, nil
}

// GenerationPreferences represents user preferences for project generation
type GenerationPreferences struct {
	Categories        []ProjectCategory `json:"categories"`
	MaxComplexity     ProjectComplexity `json:"max_complexity"`
	MinImpactScore    float64           `json:"min_impact_score"`
	FocusAreas        []string          `json:"focus_areas"`
	ExcludeCategories []ProjectCategory `json:"exclude_categories"`
	PreferredDuration int               `json:"preferred_duration_days"`
	TeamSize          int               `json:"team_size"`
	Budget            float64           `json:"budget"`
}

// analyzeOpportunities identifies current opportunities based on patterns and data
func (g *Generator) analyzeOpportunities() []*OpportunityAnalysis {
	opportunities := make([]*OpportunityAnalysis, 0)

	// Analyze patterns from pattern recognizer
	if g.patternRecognizer != nil {
		patterns := g.patternRecognizer.GetPatterns()
		for _, pattern := range patterns {
			if opportunity := g.extractOpportunityFromPattern(pattern); opportunity != nil {
				opportunities = append(opportunities, opportunity)
			}
		}
	}

	// Analyze system metrics
	systemOpportunities := g.analyzeSystemOpportunities()
	opportunities = append(opportunities, systemOpportunities...)

	// Analyze project portfolio gaps
	portfolioOpportunities := g.analyzePortfolioGaps()
	opportunities = append(opportunities, portfolioOpportunities...)

	// Store opportunities
	for _, opp := range opportunities {
		g.opportunities[opp.ID] = opp
	}

	g.stats.OpportunitiesFound = int64(len(opportunities))
	return opportunities
}

// extractOpportunityFromPattern creates opportunities from recognized patterns
func (g *Generator) extractOpportunityFromPattern(pattern *patterns.Pattern) *OpportunityAnalysis {
	if pattern.Confidence < 0.7 {
		return nil
	}

	opportunity := &OpportunityAnalysis{
		ID:             uuid.New().String(),
		Type:           string(pattern.Type),
		Description:    fmt.Sprintf("Opportunity based on %s pattern", pattern.Type),
		ImpactLevel:    pattern.Confidence,
		EffortRequired: g.estimateEffortFromPattern(pattern),
		Urgency:        g.calculateUrgency(pattern),
		StrategicValue: pattern.PredictiveValue,
		DataSources:    []string{"pattern_recognition"},
		SupportingData: map[string]interface{}{
			"pattern_id":        pattern.ID,
			"pattern_frequency": pattern.Frequency,
			"pattern_age":       pattern.Age.Hours(),
			"confidence":        pattern.Confidence,
		},
		IdentifiedAt: time.Now(),
	}

	return opportunity
}

// analyzeSystemOpportunities identifies opportunities from system analysis
func (g *Generator) analyzeSystemOpportunities() []*OpportunityAnalysis {
	opportunities := make([]*OpportunityAnalysis, 0)

	// Performance optimization opportunities
	perfOpp := &OpportunityAnalysis{
		ID:             uuid.New().String(),
		Type:           "performance_optimization",
		Description:    "System performance optimization based on metrics analysis",
		ImpactLevel:    0.8,
		EffortRequired: 0.6,
		Urgency:        0.7,
		StrategicValue: 0.8,
		DataSources:    []string{"system_metrics"},
		SupportingData: map[string]interface{}{
			"analysis_type": "performance",
		},
		IdentifiedAt: time.Now(),
	}
	opportunities = append(opportunities, perfOpp)

	// Security enhancement opportunities
	securityOpp := &OpportunityAnalysis{
		ID:             uuid.New().String(),
		Type:           "security_enhancement",
		Description:    "Security improvements based on threat analysis",
		ImpactLevel:    0.9,
		EffortRequired: 0.7,
		Urgency:        0.8,
		StrategicValue: 0.9,
		DataSources:    []string{"security_analysis"},
		SupportingData: map[string]interface{}{
			"analysis_type": "security",
		},
		IdentifiedAt: time.Now(),
	}
	opportunities = append(opportunities, securityOpp)

	return opportunities
}

// analyzePortfolioGaps identifies gaps in the current project portfolio
func (g *Generator) analyzePortfolioGaps() []*OpportunityAnalysis {
	opportunities := make([]*OpportunityAnalysis, 0)

	// Get current projects from database
	var projects []models.Project
	g.db.Find(&projects)

	// Analyze category distribution
	categoryCount := make(map[string]int)
	for _, project := range projects {
		categoryCount[project.Type]++
	}

	// Identify underrepresented categories
	totalProjects := len(projects)
	if totalProjects > 0 {
		for category := range categoryCount {
			percentage := float64(categoryCount[category]) / float64(totalProjects)
			if percentage < 0.2 { // Less than 20% representation
				opportunity := &OpportunityAnalysis{
					ID:             uuid.New().String(),
					Type:           "portfolio_gap",
					Description:    fmt.Sprintf("Underrepresented category: %s", category),
					ImpactLevel:    0.7,
					EffortRequired: 0.5,
					Urgency:        0.6,
					StrategicValue: 0.7,
					DataSources:    []string{"portfolio_analysis"},
					SupportingData: map[string]interface{}{
						"category":       category,
						"percentage":     percentage,
						"total_projects": totalProjects,
					},
					IdentifiedAt: time.Now(),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities
}

// generateIdeas creates initial project ideas from opportunities
func (g *Generator) generateIdeas(opportunities []*OpportunityAnalysis, preferences *GenerationPreferences) []*GeneratedProject {
	ideas := make([]*GeneratedProject, 0)

	for _, opportunity := range opportunities {
		// Find relevant templates
		relevantTemplates := g.findRelevantTemplates(opportunity, preferences)

		for _, template := range relevantTemplates {
			idea := g.createProjectFromTemplate(template, opportunity)
			if idea != nil {
				ideas = append(ideas, idea)
			}
		}
	}

	return ideas
}

// findRelevantTemplates finds templates that match the opportunity
func (g *Generator) findRelevantTemplates(opportunity *OpportunityAnalysis, preferences *GenerationPreferences) []*ProjectTemplate {
	relevant := make([]*ProjectTemplate, 0)

	for _, template := range g.templates {
		// Check category preferences
		if len(preferences.Categories) > 0 {
			categoryMatch := false
			for _, prefCategory := range preferences.Categories {
				if template.Category == prefCategory {
					categoryMatch = true
					break
				}
			}
			if !categoryMatch {
				continue
			}
		}

		// Check excluded categories
		excluded := false
		for _, excCategory := range preferences.ExcludeCategories {
			if template.Category == excCategory {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check complexity limits
		if g.isComplexityTooHigh(template.Complexity, preferences.MaxComplexity) {
			continue
		}

		// Check opportunity relevance
		relevanceScore := g.calculateTemplateRelevance(template, opportunity)
		if relevanceScore > 0.5 {
			relevant = append(relevant, template)
		}
	}

	return relevant
}

// createProjectFromTemplate creates a project idea from a template and opportunity
func (g *Generator) createProjectFromTemplate(template *ProjectTemplate, opportunity *OpportunityAnalysis) *GeneratedProject {
	project := &GeneratedProject{
		ID:                   uuid.New().String(),
		Title:                g.generateProjectTitle(template, opportunity),
		Description:          g.generateProjectDescription(template, opportunity),
		Category:             template.Category,
		Complexity:           template.Complexity,
		EstimatedEffort:      template.EstimatedHours,
		EstimatedDuration:    g.estimateDuration(template.EstimatedHours),
		Priority:             template.Priority,
		RequiredSkills:       template.RequiredSkills,
		RequiredTechnologies: template.RequiredTechnologies,
		Prerequisites:        template.Prerequisites,
		ExpectedOutcomes:     template.ExpectedOutcomes,
		GeneratedAt:          time.Now(),
		BasedOnPatterns:      []string{opportunity.ID},
		Metadata:             make(map[string]interface{}),
	}

	// Add opportunity context to metadata
	project.Metadata["opportunity_id"] = opportunity.ID
	project.Metadata["opportunity_type"] = opportunity.Type
	project.Metadata["template_id"] = template.ID

	return project
}

// scoreProjects evaluates and scores all project ideas
func (g *Generator) scoreProjects(projects []*GeneratedProject) []*GeneratedProject {
	for _, project := range projects {
		project.ImpactScore = g.scoringEngine.calculateImpactScore(project)
		project.FeasibilityScore = g.scoringEngine.calculateFeasibilityScore(project)
		project.AlignmentScore = g.scoringEngine.calculateAlignmentScore(project)
		project.InnovationScore = g.scoringEngine.calculateInnovationScore(project)
		project.PracticalityScore = g.scoringEngine.calculatePracticalityScore(project)

		// Calculate overall score
		project.OverallScore = (project.ImpactScore*0.3 +
			project.FeasibilityScore*0.25 +
			project.AlignmentScore*0.2 +
			project.InnovationScore*0.15 +
			project.PracticalityScore*0.1)

		// Calculate confidence level
		project.ConfidenceLevel = g.calculateConfidenceLevel(project)
	}

	// Sort by overall score
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].OverallScore > projects[j].OverallScore
	})

	return projects
}

// generateDetailedProject creates a detailed project proposal
func (g *Generator) generateDetailedProject(project *GeneratedProject, opportunities []*OpportunityAnalysis) *GeneratedProject {
	// Generate risk factors
	project.RiskFactors = g.riskAssessor.assessRisks(project)

	// Generate justification
	project.Justification = g.generateJustification(project, opportunities)

	// Calculate team size and ROI
	project.RecommendedTeamSize = g.calculateTeamSize(project)
	project.EstimatedROI = g.calculateROI(project)

	// Generate milestones
	project.Milestones = g.generateMilestones(project)

	// Generate resource requirements
	project.ResourceRequirements = g.generateResourceRequirements(project)

	return project
}

// Helper functions for scoring engine
func (s *ScoringEngine) calculateImpactScore(project *GeneratedProject) float64 {
	// Base score on category and complexity
	baseScore := 0.5

	switch project.Category {
	case CategoryAutomation:
		baseScore = 0.8
	case CategoryOptimization:
		baseScore = 0.7
	case CategoryInnovation:
		baseScore = 0.9
	case CategorySecurity:
		baseScore = 0.8
	case CategoryPerformance:
		baseScore = 0.7
	}

	// Adjust for complexity
	complexityMultiplier := map[ProjectComplexity]float64{
		ComplexityLow:    0.8,
		ComplexityMedium: 1.0,
		ComplexityHigh:   1.2,
		ComplexityExpert: 1.4,
	}

	return math.Min(1.0, baseScore*complexityMultiplier[project.Complexity])
}

func (s *ScoringEngine) calculateFeasibilityScore(project *GeneratedProject) float64 {
	// Base feasibility on complexity (inverse relationship)
	feasibilityMap := map[ProjectComplexity]float64{
		ComplexityLow:    0.9,
		ComplexityMedium: 0.7,
		ComplexityHigh:   0.5,
		ComplexityExpert: 0.3,
	}

	baseScore := feasibilityMap[project.Complexity]

	// Adjust based on required skills and technologies
	skillsPenalty := math.Min(0.3, float64(len(project.RequiredSkills))*0.05)
	techPenalty := math.Min(0.2, float64(len(project.RequiredTechnologies))*0.03)

	return math.Max(0.1, baseScore-skillsPenalty-techPenalty)
}

func (s *ScoringEngine) calculateAlignmentScore(project *GeneratedProject) float64 {
	// Base alignment on category strategic value
	alignmentMap := map[ProjectCategory]float64{
		CategoryAutomation:     0.8,
		CategoryOptimization:   0.7,
		CategoryInnovation:     0.9,
		CategoryIntegration:    0.6,
		CategoryMaintenance:    0.5,
		CategoryResearch:       0.7,
		CategorySecurity:       0.8,
		CategoryPerformance:    0.7,
		CategoryDataAnalysis:   0.8,
		CategoryInfrastructure: 0.6,
	}

	return alignmentMap[project.Category]
}

func (s *ScoringEngine) calculateInnovationScore(project *GeneratedProject) float64 {
	// Innovation score based on category and novelty
	innovationMap := map[ProjectCategory]float64{
		CategoryInnovation:     0.9,
		CategoryResearch:       0.8,
		CategoryDataAnalysis:   0.7,
		CategoryAutomation:     0.6,
		CategoryOptimization:   0.5,
		CategorySecurity:       0.6,
		CategoryPerformance:    0.5,
		CategoryIntegration:    0.4,
		CategoryMaintenance:    0.2,
		CategoryInfrastructure: 0.3,
	}

	baseScore := innovationMap[project.Category]

	// Boost for high complexity (more innovative potential)
	complexityBoost := map[ProjectComplexity]float64{
		ComplexityLow:    0.0,
		ComplexityMedium: 0.1,
		ComplexityHigh:   0.2,
		ComplexityExpert: 0.3,
	}

	return math.Min(1.0, baseScore+complexityBoost[project.Complexity])
}

func (s *ScoringEngine) calculatePracticalityScore(project *GeneratedProject) float64 {
	// Practicality is inverse to complexity and skill requirements
	baseScore := 0.8

	complexityPenalty := map[ProjectComplexity]float64{
		ComplexityLow:    0.0,
		ComplexityMedium: 0.1,
		ComplexityHigh:   0.3,
		ComplexityExpert: 0.5,
	}

	skillsPenalty := math.Min(0.3, float64(len(project.RequiredSkills))*0.04)

	return math.Max(0.1, baseScore-complexityPenalty[project.Complexity]-skillsPenalty)
}

// Helper functions continued...

// filterForDiversity ensures diverse suggestions across categories and complexity
func (d *DiversityEngine) filterForDiversity(projects []*GeneratedProject, maxSuggestions int) []*GeneratedProject {
	if len(projects) <= maxSuggestions {
		return projects
	}

	selected := make([]*GeneratedProject, 0, maxSuggestions)
	categoryCount := make(map[ProjectCategory]int)
	complexityCount := make(map[ProjectComplexity]int)

	// First pass: select top projects ensuring category diversity
	for _, project := range projects {
		if len(selected) >= maxSuggestions {
			break
		}

		// Check if this category is underrepresented
		if categoryCount[project.Category] < maxSuggestions/4 {
			selected = append(selected, project)
			categoryCount[project.Category]++
			complexityCount[project.Complexity]++
		}
	}

	// Second pass: fill remaining slots with highest scoring projects
	for _, project := range projects {
		if len(selected) >= maxSuggestions {
			break
		}

		// Check if project is already selected
		alreadySelected := false
		for _, sel := range selected {
			if sel.ID == project.ID {
				alreadySelected = true
				break
			}
		}

		if !alreadySelected {
			selected = append(selected, project)
		}
	}

	return selected
}

// Risk assessment functions
func (r *RiskAssessor) assessRisks(project *GeneratedProject) []RiskFactor {
	risks := make([]RiskFactor, 0)

	// Technical risk based on complexity
	techRisk := RiskFactor{
		Type:        "technical",
		Description: "Technical implementation challenges",
		Probability: r.complexityRisks[project.Complexity],
		Impact:      0.7,
		Mitigation:  r.mitigationStrategies["technical_risk"],
	}
	risks = append(risks, techRisk)

	// Resource risk based on team size and skills
	resourceRisk := RiskFactor{
		Type:        "resource",
		Description: "Resource availability and skill gaps",
		Probability: math.Min(0.8, float64(len(project.RequiredSkills))*0.1),
		Impact:      0.6,
		Mitigation:  r.mitigationStrategies["resource_risk"],
	}
	risks = append(risks, resourceRisk)

	// Timeline risk based on complexity and estimated effort
	timelineRisk := RiskFactor{
		Type:        "timeline",
		Description: "Project timeline overruns",
		Probability: math.Min(0.7, float64(project.EstimatedEffort)/1000.0),
		Impact:      0.5,
		Mitigation:  r.mitigationStrategies["timeline_risk"],
	}
	risks = append(risks, timelineRisk)

	return risks
}

// Utility functions
func (g *Generator) estimateEffortFromPattern(pattern *patterns.Pattern) float64 {
	// Estimate effort based on pattern frequency and complexity
	baseEffort := 0.5
	complexityFactor := math.Min(1.0, float64(pattern.Frequency)/100.0)
	return baseEffort * (1.0 + complexityFactor)
}

func (g *Generator) calculateUrgency(pattern *patterns.Pattern) float64 {
	// Urgency based on pattern age and frequency
	ageHours := pattern.Age.Hours()
	ageFactor := math.Max(0.1, 1.0-ageHours/168.0) // Decreases over a week
	frequencyFactor := math.Min(1.0, float64(pattern.Frequency)/50.0)

	return (ageFactor + frequencyFactor) / 2.0
}

func (g *Generator) isComplexityTooHigh(templateComplexity, maxComplexity ProjectComplexity) bool {
	complexityLevels := map[ProjectComplexity]int{
		ComplexityLow:    1,
		ComplexityMedium: 2,
		ComplexityHigh:   3,
		ComplexityExpert: 4,
	}

	return complexityLevels[templateComplexity] > complexityLevels[maxComplexity]
}

func (g *Generator) calculateTemplateRelevance(template *ProjectTemplate, opportunity *OpportunityAnalysis) float64 {
	// Simple relevance based on category matching
	if string(template.Category) == opportunity.Type {
		return 1.0
	}

	// Partial matches based on keywords in descriptions
	// This is a simplified implementation
	return 0.6
}

func (g *Generator) generateProjectTitle(template *ProjectTemplate, opportunity *OpportunityAnalysis) string {
	return fmt.Sprintf("%s: %s Enhancement", template.Name, opportunity.Type)
}

func (g *Generator) generateProjectDescription(template *ProjectTemplate, opportunity *OpportunityAnalysis) string {
	return fmt.Sprintf("%s This project addresses the identified opportunity in %s with an estimated impact level of %.1f.",
		template.Description, opportunity.Type, opportunity.ImpactLevel)
}

func (g *Generator) estimateDuration(estimatedHours int) int {
	// Assume 8 hours per day, 5 days per week
	workingDaysPerWeek := 5
	hoursPerDay := 8

	totalDays := estimatedHours / hoursPerDay
	weeks := totalDays / workingDaysPerWeek

	return weeks * 7 // Convert to calendar days
}

func (g *Generator) calculateConfidenceLevel(project *GeneratedProject) float64 {
	// Confidence based on various factors
	scoreConfidence := project.OverallScore
	complexityConfidence := map[ProjectComplexity]float64{
		ComplexityLow:    0.9,
		ComplexityMedium: 0.7,
		ComplexityHigh:   0.5,
		ComplexityExpert: 0.3,
	}[project.Complexity]

	return (scoreConfidence + complexityConfidence) / 2.0
}

func (g *Generator) calculateTeamSize(project *GeneratedProject) int {
	baseSize := 2

	complexitySize := map[ProjectComplexity]int{
		ComplexityLow:    1,
		ComplexityMedium: 2,
		ComplexityHigh:   3,
		ComplexityExpert: 5,
	}

	skillsSize := len(project.RequiredSkills) / 3

	return baseSize + complexitySize[project.Complexity] + skillsSize
}

func (g *Generator) calculateROI(project *GeneratedProject) float64 {
	// Simplified ROI calculation
	baseROI := project.ImpactScore * 2.0
	effortPenalty := float64(project.EstimatedEffort) / 1000.0

	return math.Max(0.1, baseROI-effortPenalty)
}

func (g *Generator) generateJustification(project *GeneratedProject, opportunities []*OpportunityAnalysis) string {
	return fmt.Sprintf("This project is recommended based on analysis showing high potential impact (%.1f) and good feasibility (%.1f). "+
		"It addresses current system opportunities and aligns with strategic objectives.",
		project.ImpactScore, project.FeasibilityScore)
}

func (g *Generator) generateMilestones(project *GeneratedProject) []ProjectMilestone {
	milestones := make([]ProjectMilestone, 0)

	totalDuration := project.EstimatedDuration

	milestones = append(milestones, ProjectMilestone{
		Name:          "Project Planning Complete",
		Description:   "Requirements gathered, design approved, team assembled",
		DaysFromStart: int(float64(totalDuration) * 0.15),
		Deliverables:  []string{"Requirements document", "Technical design", "Project plan"},
	})

	milestones = append(milestones, ProjectMilestone{
		Name:          "Development Phase 1",
		Description:   "Core functionality implemented",
		DaysFromStart: int(float64(totalDuration) * 0.5),
		Deliverables:  []string{"Core features", "Unit tests", "Integration tests"},
	})

	milestones = append(milestones, ProjectMilestone{
		Name:          "Testing and Integration",
		Description:   "System testing and integration complete",
		DaysFromStart: int(float64(totalDuration) * 0.8),
		Deliverables:  []string{"Test results", "Performance metrics", "Integration complete"},
	})

	milestones = append(milestones, ProjectMilestone{
		Name:          "Project Completion",
		Description:   "Project delivered and deployed",
		DaysFromStart: totalDuration,
		Deliverables:  []string{"Final deliverable", "Documentation", "Training materials"},
	})

	return milestones
}

func (g *Generator) generateResourceRequirements(project *GeneratedProject) ResourceRequirements {
	baseHours := project.EstimatedEffort

	return ResourceRequirements{
		DeveloperHours: int(float64(baseHours) * 0.6),
		QAHours:        int(float64(baseHours) * 0.2),
		DevOpsHours:    int(float64(baseHours) * 0.1),
		DesignHours:    int(float64(baseHours) * 0.1),
		RequiredTools:  []string{"Development IDE", "Testing framework", "Version control"},
		Budget:         float64(baseHours) * 100.0, // $100 per hour average
		CloudResources: []string{"Development environment", "Testing environment"},
	}
}

// Routine functions
func (g *Generator) opportunityAnalysisRoutine() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.analyzeOpportunities()
		}
	}
}

func (g *Generator) statsUpdateRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			// Update any derived statistics
		}
	}
}

// Template management
func (g *Generator) loadTemplates() error {
	// In a real implementation, this would load from database or files
	return nil
}

func (g *Generator) initializeDefaultTemplates() {
	templates := []*ProjectTemplate{
		{
			ID:                   "automation_001",
			Name:                 "Process Automation",
			Category:             CategoryAutomation,
			Description:          "Automate manual processes to improve efficiency and reduce errors",
			Complexity:           ComplexityMedium,
			EstimatedHours:       120,
			RequiredSkills:       []string{"scripting", "process_analysis", "testing"},
			RequiredTechnologies: []string{"automation_framework", "monitoring"},
			ExpectedOutcomes:     []string{"Reduced manual effort", "Improved accuracy", "Cost savings"},
			Prerequisites:        []string{"Process documentation", "Stakeholder approval"},
			RiskFactors:          []string{"Integration complexity", "User adoption"},
			SuccessMetrics:       []string{"Time saved", "Error reduction", "User satisfaction"},
			Priority:             "high",
		},
		{
			ID:                   "optimization_001",
			Name:                 "Performance Optimization",
			Category:             CategoryOptimization,
			Description:          "Optimize system performance and resource utilization",
			Complexity:           ComplexityHigh,
			EstimatedHours:       200,
			RequiredSkills:       []string{"performance_analysis", "system_tuning", "monitoring"},
			RequiredTechnologies: []string{"profiling_tools", "monitoring_systems"},
			ExpectedOutcomes:     []string{"Improved response times", "Better resource utilization", "Enhanced user experience"},
			Prerequisites:        []string{"Performance baseline", "Monitoring setup"},
			RiskFactors:          []string{"System stability", "Rollback complexity"},
			SuccessMetrics:       []string{"Response time improvement", "Resource usage reduction"},
			Priority:             "medium",
		},
		{
			ID:                   "security_001",
			Name:                 "Security Enhancement",
			Category:             CategorySecurity,
			Description:          "Enhance system security and compliance",
			Complexity:           ComplexityHigh,
			EstimatedHours:       160,
			RequiredSkills:       []string{"security_analysis", "compliance", "testing"},
			RequiredTechnologies: []string{"security_tools", "monitoring", "encryption"},
			ExpectedOutcomes:     []string{"Improved security posture", "Compliance achievement", "Risk reduction"},
			Prerequisites:        []string{"Security audit", "Compliance requirements"},
			RiskFactors:          []string{"Implementation complexity", "User impact"},
			SuccessMetrics:       []string{"Vulnerability reduction", "Compliance score"},
			Priority:             "high",
		},
	}

	for _, template := range templates {
		g.templates[template.ID] = template
	}

	g.logger.WithField("template_count", len(templates)).Info("Initialized default project templates")
}

// Public API methods
func (g *Generator) GetStats() *GeneratorStats {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	stats := *g.stats
	return &stats
}

func (g *Generator) GetOpportunities() map[string]*OpportunityAnalysis {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	opportunities := make(map[string]*OpportunityAnalysis)
	for k, v := range g.opportunities {
		opportunities[k] = v
	}
	return opportunities
}

func (g *Generator) GetTemplates() map[string]*ProjectTemplate {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	templates := make(map[string]*ProjectTemplate)
	for k, v := range g.templates {
		templates[k] = v
	}
	return templates
}

// Health returns the health status of the generator
func (g *Generator) Health() map[string]interface{} {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return map[string]interface{}{
		"status":            "healthy",
		"template_count":    len(g.templates),
		"opportunity_count": len(g.opportunities),
		"stats":             g.stats,
	}
}
