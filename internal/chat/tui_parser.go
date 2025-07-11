package chat

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TUIParserImpl converts terminal UI elements to chat-friendly components
type TUIParserImpl struct {
	logger               *logrus.Logger
	componentRegistry    map[string]TUIComponentParser
	renderingRules       *RenderingRules
	adaptationEngine     *ChatAdaptationEngine
	terminalEmulator     *TerminalEmulator
	ansiProcessor        *ANSIProcessor
	layoutEngine         *LayoutEngine
}

// TUIComponentParser interface for parsing specific TUI component types
type TUIComponentParser interface {
	CanParse(input string) bool
	Parse(input string, context *TUIParseContext) (*TUIComponent, error)
	GetComponentType() string
	GetPriority() int
}

// TUIParseContext provides context for parsing TUI elements
type TUIParseContext struct {
	TerminalSize    TerminalSize         `json:"terminal_size"`
	ColorSupport    bool                 `json:"color_support"`
	UserPreferences *UserChatPreferences `json:"user_preferences"`
	ViewportSize    ViewportSize         `json:"viewport_size"`
	Theme           *TUITheme            `json:"theme"`
	ParsedElements  []*TUIComponent      `json:"parsed_elements"`
	ParentContext   *ChatContext         `json:"parent_context"`
}

// TerminalSize represents terminal dimensions
type TerminalSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ViewportSize represents chat viewport dimensions
type ViewportSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TUITheme defines visual theme for TUI adaptation
type TUITheme struct {
	PrimaryColor   string            `json:"primary_color"`
	SecondaryColor string            `json:"secondary_color"`
	BackgroundColor string           `json:"background_color"`
	TextColor      string            `json:"text_color"`
	AccentColors   map[string]string `json:"accent_colors"`
	FontFamily     string            `json:"font_family"`
	FontSize       string            `json:"font_size"`
}

// RenderingRules define how TUI elements should be adapted for chat
type RenderingRules struct {
	MaxTableRows     int                     `json:"max_table_rows"`
	MaxChartWidth    int                     `json:"max_chart_width"`
	MaxChartHeight   int                     `json:"max_chart_height"`
	SummaryThreshold int                     `json:"summary_threshold"`
	AdaptationRules  map[string]AdaptationRule `json:"adaptation_rules"`
}

// AdaptationRule defines how to adapt specific TUI elements
type AdaptationRule struct {
	ComponentType    string                 `json:"component_type"`
	ChatRepresentation string               `json:"chat_representation"`
	PreserveFidelity bool                   `json:"preserve_fidelity"`
	Transformations  []Transformation       `json:"transformations"`
	FallbackStrategy string                 `json:"fallback_strategy"`
}

// Transformation defines data transformations for chat adaptation
type Transformation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Condition  string                 `json:"condition"`
}

// Component parsers
type TableParser struct {
	logger *logrus.Logger
}

type ChartParser struct {
	logger *logrus.Logger
}

type ProgressBarParser struct {
	logger *logrus.Logger
}

type TreeViewParser struct {
	logger *logrus.Logger
}

type FormParser struct {
	logger *logrus.Logger
}

type ListParser struct {
	logger *logrus.Logger
}

// Supporting components
type ChatAdaptationEngine struct {
	logger *logrus.Logger
}

type TerminalEmulator struct {
	logger *logrus.Logger
}

type ANSIProcessor struct {
	logger *logrus.Logger
}

type LayoutEngine struct {
	logger *logrus.Logger
}

// NewTUIParserImpl creates a comprehensive TUI parsing system
func NewTUIParserImpl(logger *logrus.Logger) *TUIParserImpl {
	parser := &TUIParserImpl{
		logger:            logger,
		componentRegistry: make(map[string]TUIComponentParser),
		renderingRules:    createDefaultRenderingRules(),
		adaptationEngine:  NewChatAdaptationEngine(logger),
		terminalEmulator:  NewTerminalEmulator(logger),
		ansiProcessor:     NewANSIProcessor(logger),
		layoutEngine:      NewLayoutEngine(logger),
	}

	// Register component parsers
	parser.registerComponentParsers()

	return parser
}

// ParseTUIOutput converts TUI output to chat components
func (tui *TUIParserImpl) ParseTUIOutput(ctx context.Context, tuiOutput string, parseContext *TUIParseContext) ([]*TUIComponent, error) {
	tui.logger.WithFields(logrus.Fields{
		"output_length": len(tuiOutput),
		"terminal_size": parseContext.TerminalSize,
	}).Info("Starting TUI output parsing")

	// Pre-process the TUI output
	processedOutput, err := tui.preprocessTUIOutput(tuiOutput, parseContext)
	if err != nil {
		return nil, fmt.Errorf("TUI preprocessing failed: %w", err)
	}

	// Parse different component types
	components := []*TUIComponent{}

	// Parse in priority order
	parsers := tui.getSortedParsers()
	for _, parser := range parsers {
		if parser.CanParse(processedOutput) {
			component, err := parser.Parse(processedOutput, parseContext)
			if err != nil {
				tui.logger.WithError(err).Warnf("Failed to parse with %s", parser.GetComponentType())
				continue
			}

			if component != nil {
				// Adapt component for chat display
				adaptedComponent, err := tui.adaptForChat(component, parseContext)
				if err != nil {
					tui.logger.WithError(err).Warn("Component adaptation failed")
					components = append(components, component) // Use original
				} else {
					components = append(components, adaptedComponent)
				}
			}
		}
	}

	// Post-process components
	finalComponents, err := tui.postprocessComponents(components, parseContext)
	if err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	tui.logger.WithFields(logrus.Fields{
		"components_parsed": len(finalComponents),
		"component_types":   tui.getComponentTypes(finalComponents),
	}).Info("TUI parsing completed")

	return finalComponents, nil
}

// ParseInteractiveTUI handles interactive TUI elements
func (tui *TUIParserImpl) ParseInteractiveTUI(ctx context.Context, tuiState *TUIState, action string) (*TUIInteractionResult, error) {
	result := &TUIInteractionResult{
		UpdatedComponents: []*TUIComponent{},
		StateChanges:      make(map[string]interface{}),
		RequiredActions:   []string{},
	}

	// Process the interaction
	switch action {
	case "scroll_down":
		result = tui.handleScroll(tuiState, "down")
	case "scroll_up":
		result = tui.handleScroll(tuiState, "up")
	case "select_item":
		result = tui.handleSelection(tuiState)
	case "navigate_menu":
		result = tui.handleNavigation(tuiState)
	case "toggle_expand":
		result = tui.handleToggle(tuiState)
	default:
		return nil, fmt.Errorf("unsupported interaction: %s", action)
	}

	return result, nil
}

// preprocessTUIOutput cleans and prepares TUI output for parsing
func (tui *TUIParserImpl) preprocessTUIOutput(output string, context *TUIParseContext) (string, error) {
	// Remove ANSI escape sequences if needed
	cleaned := tui.ansiProcessor.StripANSI(output)

	// Normalize line endings
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	// Handle terminal control characters
	cleaned = tui.terminalEmulator.ProcessControlChars(cleaned, context.TerminalSize)

	// Normalize whitespace but preserve formatting
	cleaned = tui.normalizeWhitespace(cleaned)

	return cleaned, nil
}

// adaptForChat adapts TUI components for chat display
func (tui *TUIParserImpl) adaptForChat(component *TUIComponent, context *TUIParseContext) (*TUIComponent, error) {
	adapted := *component // Copy component

	// Apply adaptation rules based on component type
	rule, exists := tui.renderingRules.AdaptationRules[component.Type]
	if !exists {
		// Use default adaptation
		return tui.applyDefaultAdaptation(&adapted, context)
	}

	// Apply specific transformations
	for _, transformation := range rule.Transformations {
		if err := tui.applyTransformation(&adapted, transformation, context); err != nil {
			tui.logger.WithError(err).Warn("Transformation failed")
			continue
		}
	}

	// Generate chat summary
	summary, err := tui.generateChatSummary(&adapted, context)
	if err != nil {
		tui.logger.WithError(err).Warn("Summary generation failed")
	} else {
		adapted.ChatSummary = summary
	}

	// Generate quick actions
	quickActions := tui.generateQuickActions(&adapted, context)
	adapted.QuickActions = quickActions

	// Generate related questions
	relatedQuestions := tui.generateRelatedQuestions(&adapted, context)
	adapted.RelatedQuestions = relatedQuestions

	return &adapted, nil
}

// Component parsers implementation

// TableParser implementation
func (tp *TableParser) CanParse(input string) bool {
	// Look for table patterns
	tablePatterns := []string{
		`\|.*\|`, // Basic pipe-separated table
		`\+[-+]+\+`, // ASCII table borders
		`┌.*┐`, // Unicode table borders
		`├.*┤`, // Unicode table separators
	}

	for _, pattern := range tablePatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}

	return false
}

func (tp *TableParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) {
	lines := strings.Split(input, "\n")
	
	// Find table boundaries
	tableStart, tableEnd := tp.findTableBoundaries(lines)
	if tableStart == -1 || tableEnd == -1 {
		return nil, fmt.Errorf("table boundaries not found")
	}

	tableLines := lines[tableStart:tableEnd+1]
	
	// Parse table structure
	columns, rows, err := tp.parseTableStructure(tableLines)
	if err != nil {
		return nil, fmt.Errorf("table structure parsing failed: %w", err)
	}

	component := &TUIComponent{
		ID:    uuid.New(),
		Type:  "table",
		Title: "Data Table",
		Data: map[string]interface{}{
			"columns": columns,
			"rows":    rows,
		},
		Columns: tp.createTUIColumns(columns),
		Rows:    tp.createTUIRows(rows),
	}

	return component, nil
}

func (tp *TableParser) GetComponentType() string { return "table" }
func (tp *TableParser) GetPriority() int { return 80 }

func (tp *TableParser) findTableBoundaries(lines []string) (int, int) {
	start, end := -1, -1
	
	for i, line := range lines {
		if strings.Contains(line, "|") || strings.Contains(line, "┌") || strings.Contains(line, "├") {
			if start == -1 {
				start = i
			}
			end = i
		}
	}
	
	return start, end
}

func (tp *TableParser) parseTableStructure(lines []string) ([]string, [][]string, error) {
	if len(lines) == 0 {
		return nil, nil, fmt.Errorf("empty table")
	}

	// Find header line
	var headerLine string
	var dataStart int
	
	for i, line := range lines {
		if strings.Contains(line, "|") && !strings.Contains(line, "+") && !strings.Contains(line, "-") {
			headerLine = line
			dataStart = i + 1
			break
		}
	}

	if headerLine == "" {
		return nil, nil, fmt.Errorf("header line not found")
	}

	// Parse columns from header
	columns := tp.parseTableRow(headerLine)
	
	// Parse data rows
	var rows [][]string
	for i := dataStart; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "|") && !strings.Contains(line, "+") && !strings.Contains(line, "-") {
			row := tp.parseTableRow(line)
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
	}

	return columns, rows, nil
}

func (tp *TableParser) parseTableRow(line string) []string {
	// Remove leading/trailing pipes and split
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	
	var cells []string
	for _, part := range parts {
		cell := strings.TrimSpace(part)
		if cell != "" {
			cells = append(cells, cell)
		}
	}
	
	return cells
}

func (tp *TableParser) createTUIColumns(columns []string) []TUIColumn {
	tuiColumns := make([]TUIColumn, len(columns))
	for i, col := range columns {
		tuiColumns[i] = TUIColumn{
			Name:  col,
			Type:  "string",
			Width: len(col) + 10, // Estimate width
		}
	}
	return tuiColumns
}

func (tp *TableParser) createTUIRows(rows [][]string) []TUIRow {
	tuiRows := make([]TUIRow, len(rows))
	for i, row := range rows {
		tuiRows[i] = TUIRow{
			ID:    fmt.Sprintf("row_%d", i),
			Cells: row,
		}
	}
	return tuiRows
}

// ChartParser implementation
func (cp *ChartParser) CanParse(input string) bool {
	chartPatterns := []string{
		`\*+`, // ASCII bar charts
		`##+`, // Hash-based charts
		`▄▄+`, // Unicode block charts
		`█+`,  // Full block charts
		`[0-9]+%`, // Percentage indicators
	}

	for _, pattern := range chartPatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}

	return false
}

func (cp *ChartParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) {
	// Detect chart type
	chartType := cp.detectChartType(input)
	
	// Parse chart data
	chartData, err := cp.parseChartData(input, chartType)
	if err != nil {
		return nil, fmt.Errorf("chart data parsing failed: %w", err)
	}

	component := &TUIComponent{
		ID:    uuid.New(),
		Type:  "chart",
		Title: "Chart Visualization",
		ChartData: &ChartData{
			Type: chartType,
			Data: chartData,
		},
	}

	return component, nil
}

func (cp *ChartParser) GetComponentType() string { return "chart" }
func (cp *ChartParser) GetPriority() int { return 75 }

func (cp *ChartParser) detectChartType(input string) string {
	if strings.Contains(input, "█") || strings.Contains(input, "▄") {
		return "bar"
	} else if strings.Contains(input, "%") {
		return "progress"
	} else if strings.Contains(input, "*") {
		return "scatter"
	}
	return "bar" // Default
}

func (cp *ChartParser) parseChartData(input string, chartType string) (map[string]interface{}, error) {
	lines := strings.Split(input, "\n")
	
	data := make(map[string]interface{})
	labels := []string{}
	values := []float64{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract label and value
		label, value := cp.extractLabelValue(line)
		if label != "" {
			labels = append(labels, label)
			values = append(values, value)
		}
	}

	data["labels"] = labels
	data["values"] = values
	data["chart_type"] = chartType

	return data, nil
}

func (cp *ChartParser) extractLabelValue(line string) (string, float64) {
	// Look for patterns like "Label: ████ 75%"
	percentRegex := regexp.MustCompile(`(.+):\s*[█▄*#]+\s*([0-9.]+)%`)
	matches := percentRegex.FindStringSubmatch(line)
	
	if len(matches) >= 3 {
		label := strings.TrimSpace(matches[1])
		valueStr := matches[2]
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return label, value
		}
	}

	// Look for other patterns
	barRegex := regexp.MustCompile(`(.+):\s*([█▄*#]+)`)
	matches = barRegex.FindStringSubmatch(line)
	
	if len(matches) >= 3 {
		label := strings.TrimSpace(matches[1])
		bars := matches[2]
		value := float64(len(bars)) * 10 // Estimate value from bar length
		return label, value
	}

	return "", 0
}

// Helper methods

func (tui *TUIParserImpl) registerComponentParsers() {
	tui.componentRegistry["table"] = &TableParser{logger: tui.logger}
	tui.componentRegistry["chart"] = &ChartParser{logger: tui.logger}
	tui.componentRegistry["progress"] = &ProgressBarParser{logger: tui.logger}
	tui.componentRegistry["tree"] = &TreeViewParser{logger: tui.logger}
	tui.componentRegistry["form"] = &FormParser{logger: tui.logger}
	tui.componentRegistry["list"] = &ListParser{logger: tui.logger}
}

func (tui *TUIParserImpl) getSortedParsers() []TUIComponentParser {
	parsers := make([]TUIComponentParser, 0, len(tui.componentRegistry))
	for _, parser := range tui.componentRegistry {
		parsers = append(parsers, parser)
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(parsers)-1; i++ {
		for j := i + 1; j < len(parsers); j++ {
			if parsers[i].GetPriority() < parsers[j].GetPriority() {
				parsers[i], parsers[j] = parsers[j], parsers[i]
			}
		}
	}

	return parsers
}

func (tui *TUIParserImpl) normalizeWhitespace(input string) string {
	// Preserve important whitespace while cleaning up excess
	lines := strings.Split(input, "\n")
	var normalized []string

	for _, line := range lines {
		// Trim trailing whitespace but preserve leading whitespace for formatting
		trimmed := strings.TrimRight(line, " \t")
		normalized = append(normalized, trimmed)
	}

	return strings.Join(normalized, "\n")
}

func (tui *TUIParserImpl) applyDefaultAdaptation(component *TUIComponent, context *TUIParseContext) (*TUIComponent, error) {
	// Apply default adaptations based on viewport size
	if context.ViewportSize.Width < 400 {
		// Mobile adaptations
		component = tui.adaptForMobile(component)
	}

	return component, nil
}

func (tui *TUIParserImpl) adaptForMobile(component *TUIComponent) *TUIComponent {
	// Simplify component for mobile display
	switch component.Type {
	case "table":
		// Limit table columns for mobile
		if len(component.Columns) > 3 {
			component.Columns = component.Columns[:3]
			// Update rows accordingly
			for i := range component.Rows {
				if len(component.Rows[i].Cells) > 3 {
					component.Rows[i].Cells = component.Rows[i].Cells[:3]
				}
			}
		}
	case "chart":
		// Adjust chart size for mobile
		if component.ChartData != nil {
			component.ChartData.Data.(map[string]interface{})["mobile_optimized"] = true
		}
	}

	return component
}

func (tui *TUIParserImpl) applyTransformation(component *TUIComponent, transformation Transformation, context *TUIParseContext) error {
	switch transformation.Type {
	case "summarize":
		return tui.applySummarization(component, transformation.Parameters)
	case "format":
		return tui.applyFormatting(component, transformation.Parameters)
	case "filter":
		return tui.applyFiltering(component, transformation.Parameters)
	default:
		return fmt.Errorf("unsupported transformation: %s", transformation.Type)
	}
}

func (tui *TUIParserImpl) applySummarization(component *TUIComponent, params map[string]interface{}) error {
	// Generate summary based on component type
	switch component.Type {
	case "table":
		if len(component.Rows) > 10 {
			component.ChatSummary = fmt.Sprintf("Table with %d rows and %d columns (showing first 10 rows)", 
				len(component.Rows), len(component.Columns))
			component.Rows = component.Rows[:10]
		}
	case "chart":
		component.ChatSummary = "Chart visualization of data trends"
	}
	
	return nil
}

func (tui *TUIParserImpl) applyFormatting(component *TUIComponent, params map[string]interface{}) error {
	// Apply formatting transformations
	return nil
}

func (tui *TUIParserImpl) applyFiltering(component *TUIComponent, params map[string]interface{}) error {
	// Apply filtering transformations
	return nil
}

func (tui *TUIParserImpl) generateChatSummary(component *TUIComponent, context *TUIParseContext) (string, error) {
	switch component.Type {
	case "table":
		return fmt.Sprintf("Table showing %d rows of data with columns: %s",
			len(component.Rows), tui.getColumnNames(component.Columns)), nil
	case "chart":
		return "Visual chart representation of project data", nil
	case "progress":
		return "Progress indicator showing completion status", nil
	default:
		return fmt.Sprintf("%s component", component.Type), nil
	}
}

func (tui *TUIParserImpl) generateQuickActions(component *TUIComponent, context *TUIParseContext) []*ChatAction {
	var actions []*ChatAction

	switch component.Type {
	case "table":
		actions = append(actions, &ChatAction{
			ID:   "export_table",
			Text: "📊 Export Data",
			Type: "action",
		})
		actions = append(actions, &ChatAction{
			ID:   "filter_table",
			Text: "🔍 Filter Rows",
			Type: "action",
		})
	case "chart":
		actions = append(actions, &ChatAction{
			ID:   "chart_details",
			Text: "📈 View Details",
			Type: "action",
		})
	}

	return actions
}

func (tui *TUIParserImpl) generateRelatedQuestions(component *TUIComponent, context *TUIParseContext) []string {
	switch component.Type {
	case "table":
		return []string{
			"Can you explain what this data shows?",
			"What are the key insights from this table?",
			"How can I filter this data?",
		}
	case "chart":
		return []string{
			"What trends does this chart show?",
			"Can you create a different chart type?",
			"What do these values represent?",
		}
	default:
		return []string{
			"Can you explain this component?",
			"How can I interact with this?",
		}
	}
}

func (tui *TUIParserImpl) postprocessComponents(components []*TUIComponent, context *TUIParseContext) ([]*TUIComponent, error) {
	// Remove duplicates
	uniqueComponents := tui.removeDuplicateComponents(components)
	
	// Sort by relevance
	sortedComponents := tui.sortComponentsByRelevance(uniqueComponents, context)
	
	return sortedComponents, nil
}

func (tui *TUIParserImpl) removeDuplicateComponents(components []*TUIComponent) []*TUIComponent {
	seen := make(map[string]bool)
	var unique []*TUIComponent

	for _, component := range components {
		key := fmt.Sprintf("%s_%s", component.Type, component.Title)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, component)
		}
	}

	return unique
}

func (tui *TUIParserImpl) sortComponentsByRelevance(components []*TUIComponent, context *TUIParseContext) []*TUIComponent {
	// Simple relevance sorting - can be enhanced with ML
	priorities := map[string]int{
		"chart":    100,
		"table":    90,
		"progress": 80,
		"tree":     70,
		"list":     60,
		"form":     50,
	}

	for i := 0; i < len(components)-1; i++ {
		for j := i + 1; j < len(components); j++ {
			priorityI := priorities[components[i].Type]
			priorityJ := priorities[components[j].Type]
			
			if priorityI < priorityJ {
				components[i], components[j] = components[j], components[i]
			}
		}
	}

	return components
}

func (tui *TUIParserImpl) getComponentTypes(components []*TUIComponent) []string {
	var types []string
	for _, component := range components {
		types = append(types, component.Type)
	}
	return types
}

func (tui *TUIParserImpl) getColumnNames(columns []TUIColumn) string {
	var names []string
	for _, col := range columns {
		names = append(names, col.Name)
	}
	return strings.Join(names, ", ")
}

func (tui *TUIParserImpl) handleScroll(state *TUIState, direction string) *TUIInteractionResult {
	return &TUIInteractionResult{
		UpdatedComponents: []*TUIComponent{},
		StateChanges:      map[string]interface{}{"scroll_direction": direction},
		RequiredActions:   []string{"update_viewport"},
	}
}

func (tui *TUIParserImpl) handleSelection(state *TUIState) *TUIInteractionResult {
	return &TUIInteractionResult{
		UpdatedComponents: []*TUIComponent{},
		StateChanges:      map[string]interface{}{"selected": true},
		RequiredActions:   []string{"show_details"},
	}
}

func (tui *TUIParserImpl) handleNavigation(state *TUIState) *TUIInteractionResult {
	return &TUIInteractionResult{
		UpdatedComponents: []*TUIComponent{},
		StateChanges:      map[string]interface{}{"navigation": "active"},
		RequiredActions:   []string{"update_menu"},
	}
}

func (tui *TUIParserImpl) handleToggle(state *TUIState) *TUIInteractionResult {
	return &TUIInteractionResult{
		UpdatedComponents: []*TUIComponent{},
		StateChanges:      map[string]interface{}{"expanded": true},
		RequiredActions:   []string{"refresh_view"},
	}
}

// Supporting types implementations

type TUIColumn struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Width int    `json:"width"`
}

type TUIRow struct {
	ID    string   `json:"id"`
	Cells []string `json:"cells"`
}

type TUIInteractionResult struct {
	UpdatedComponents []*TUIComponent        `json:"updated_components"`
	StateChanges      map[string]interface{} `json:"state_changes"`
	RequiredActions   []string               `json:"required_actions"`
}

// Helper functions for supporting components
func NewChatAdaptationEngine(logger *logrus.Logger) *ChatAdaptationEngine {
	return &ChatAdaptationEngine{logger: logger}
}

func NewTerminalEmulator(logger *logrus.Logger) *TerminalEmulator {
	return &TerminalEmulator{logger: logger}
}

func NewANSIProcessor(logger *logrus.Logger) *ANSIProcessor {
	return &ANSIProcessor{logger: logger}
}

func NewLayoutEngine(logger *logrus.Logger) *LayoutEngine {
	return &LayoutEngine{logger: logger}
}

func createDefaultRenderingRules() *RenderingRules {
	return &RenderingRules{
		MaxTableRows:     50,
		MaxChartWidth:    400,
		MaxChartHeight:   300,
		SummaryThreshold: 100,
		AdaptationRules: map[string]AdaptationRule{
			"table": {
				ComponentType:    "table",
				ChatRepresentation: "interactive_table",
				PreserveFidelity: true,
				Transformations: []Transformation{
					{Type: "summarize", Parameters: map[string]interface{}{"max_rows": 10}},
				},
				FallbackStrategy: "text_summary",
			},
			"chart": {
				ComponentType:    "chart",
				ChatRepresentation: "interactive_chart",
				PreserveFidelity: true,
				Transformations: []Transformation{
					{Type: "format", Parameters: map[string]interface{}{"responsive": true}},
				},
				FallbackStrategy: "data_table",
			},
		},
	}
}

// Method implementations for supporting components
func (ansi *ANSIProcessor) StripANSI(input string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(input, "")
}

func (term *TerminalEmulator) ProcessControlChars(input string, termSize TerminalSize) string {
	// Process terminal control characters
	return input
}

// Placeholder implementations for other parsers
func (pb *ProgressBarParser) CanParse(input string) bool { return strings.Contains(input, "[") && strings.Contains(input, "]") }
func (pb *ProgressBarParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) { return nil, nil }
func (pb *ProgressBarParser) GetComponentType() string { return "progress" }
func (pb *ProgressBarParser) GetPriority() int { return 70 }

func (tv *TreeViewParser) CanParse(input string) bool { return strings.Contains(input, "├") || strings.Contains(input, "└") }
func (tv *TreeViewParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) { return nil, nil }
func (tv *TreeViewParser) GetComponentType() string { return "tree" }
func (tv *TreeViewParser) GetPriority() int { return 65 }

func (fp *FormParser) CanParse(input string) bool { return strings.Contains(input, "[   ]") || strings.Contains(input, "Input:") }
func (fp *FormParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) { return nil, nil }
func (fp *FormParser) GetComponentType() string { return "form" }
func (fp *FormParser) GetPriority() int { return 60 }

func (lp *ListParser) CanParse(input string) bool { return strings.Contains(input, "•") || strings.Contains(input, "-") }
func (lp *ListParser) Parse(input string, context *TUIParseContext) (*TUIComponent, error) { return nil, nil }
func (lp *ListParser) GetComponentType() string { return "list" }
func (lp *ListParser) GetPriority() int { return 55 }