package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

// Colors for output
var (
	ColorRed     = color.New(color.FgRed)
	ColorGreen   = color.New(color.FgGreen)
	ColorYellow  = color.New(color.FgYellow)
	ColorBlue    = color.New(color.FgBlue)
	ColorMagenta = color.New(color.FgMagenta)
	ColorCyan    = color.New(color.FgCyan)
	ColorBold    = color.New(color.Bold)
)

// Auth configuration
type AuthConfig struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
	User    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".kaskman")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// SaveAuthConfig saves authentication configuration
func SaveAuthConfig(config *AuthConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "auth.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save auth config: %w", err)
	}

	return nil
}

// LoadAuthConfig loads authentication configuration
func LoadAuthConfig() (*AuthConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "auth.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated, please run 'kaskman auth login'")
		}
		return nil, fmt.Errorf("failed to read auth config: %w", err)
	}

	var config AuthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth config: %w", err)
	}

	return &config, nil
}

// ClearAuthConfig removes authentication configuration
func ClearAuthConfig() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "auth.json")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove auth config: %w", err)
	}

	return nil
}

// PromptString prompts for a string input
func PromptString(message string) (string, error) {
	fmt.Print(message + ": ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// PromptPassword prompts for a password input (hidden)
func PromptPassword(message string) (string, error) {
	fmt.Print(message + ": ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(password), nil
}

// PromptConfirm prompts for a yes/no confirmation
func PromptConfirm(message string) (bool, error) {
	fmt.Print(message + " (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

// PromptChoice prompts for a choice from a list
func PromptChoice(message string, choices []string) (int, error) {
	fmt.Println(message)
	for i, choice := range choices {
		fmt.Printf("%d. %s\n", i+1, choice)
	}

	input, err := PromptString("Enter your choice (1-" + strconv.Itoa(len(choices)) + ")")
	if err != nil {
		return 0, err
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(choices) {
		return 0, fmt.Errorf("invalid choice")
	}

	return choice - 1, nil
}

// NewSpinner creates a new spinner with default settings
func NewSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan")
	return s
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	ColorGreen.Printf("✓ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	ColorRed.Printf("✗ %s\n", message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	ColorYellow.Printf("⚠ %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	ColorBlue.Printf("ℹ %s\n", message)
}

// PrintHeader prints a header message
func PrintHeader(message string) {
	ColorBold.Printf("\n%s\n", message)
	fmt.Println(strings.Repeat("=", len(message)))
}

// PrintSubHeader prints a sub-header message
func PrintSubHeader(message string) {
	ColorBold.Printf("\n%s\n", message)
	fmt.Println(strings.Repeat("-", len(message)))
}

// FormatTable creates a formatted table
func FormatTable(headers []string, data [][]string) string {
	var output strings.Builder

	table := tablewriter.NewWriter(&output)
	table.SetHeader(headers)
	table.SetBorder(true)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
	)

	for _, row := range data {
		table.Append(row)
	}

	table.Render()
	return output.String()
}

// FormatProject formats a project for display
func FormatProject(project *models.Project) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("ID: %s\n", project.ID))
	output.WriteString(fmt.Sprintf("Name: %s\n", project.Name))
	output.WriteString(fmt.Sprintf("Description: %s\n", project.Description))
	output.WriteString(fmt.Sprintf("Type: %s\n", project.Type))
	output.WriteString(fmt.Sprintf("Status: %s\n", project.Status))
	output.WriteString(fmt.Sprintf("Priority: %s\n", project.Priority))
	output.WriteString(fmt.Sprintf("Progress: %d%%\n", project.Progress))
	output.WriteString(fmt.Sprintf("Estimated Hours: %d\n", project.EstimatedHours))
	output.WriteString(fmt.Sprintf("Actual Hours: %d\n", project.ActualHours))
	output.WriteString(fmt.Sprintf("Budget: $%.2f\n", project.Budget))
	output.WriteString(fmt.Sprintf("Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Updated: %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05")))

	if project.StartDate != nil {
		output.WriteString(fmt.Sprintf("Start Date: %s\n", project.StartDate.Format("2006-01-02")))
	}
	if project.EndDate != nil {
		output.WriteString(fmt.Sprintf("End Date: %s\n", project.EndDate.Format("2006-01-02")))
	}

	return output.String()
}

// FormatTask formats a task for display
func FormatTask(task *models.Task) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("ID: %s\n", task.ID))
	output.WriteString(fmt.Sprintf("Title: %s\n", task.Title))
	output.WriteString(fmt.Sprintf("Description: %s\n", task.Description))
	output.WriteString(fmt.Sprintf("Type: %s\n", task.Type))
	output.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	output.WriteString(fmt.Sprintf("Priority: %s\n", task.Priority))
	output.WriteString(fmt.Sprintf("Progress: %d%%\n", task.Progress))
	output.WriteString(fmt.Sprintf("Estimated Time: %d minutes\n", task.EstimatedTime))
	output.WriteString(fmt.Sprintf("Actual Time: %d minutes\n", task.ActualTime))
	output.WriteString(fmt.Sprintf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Updated: %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05")))

	if task.StartedAt != nil {
		output.WriteString(fmt.Sprintf("Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if task.CompletedAt != nil {
		output.WriteString(fmt.Sprintf("Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05")))
	}
	if task.Result != "" {
		output.WriteString(fmt.Sprintf("Result: %s\n", task.Result))
	}
	if task.ErrorMessage != "" {
		output.WriteString(fmt.Sprintf("Error: %s\n", task.ErrorMessage))
	}

	return output.String()
}

// FormatAgent formats an agent for display
func FormatAgent(agent *models.Agent) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("ID: %s\n", agent.ID))
	output.WriteString(fmt.Sprintf("Name: %s\n", agent.Name))
	output.WriteString(fmt.Sprintf("Type: %s\n", agent.Type))
	output.WriteString(fmt.Sprintf("Status: %s\n", agent.Status))
	output.WriteString(fmt.Sprintf("Task Count: %d\n", agent.TaskCount))
	output.WriteString(fmt.Sprintf("Success Rate: %.2f%%\n", agent.SuccessRate))
	output.WriteString(fmt.Sprintf("Avg Response Time: %.2f ms\n", agent.AvgResponseTime))
	output.WriteString(fmt.Sprintf("Created: %s\n", agent.CreatedAt.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Updated: %s\n", agent.UpdatedAt.Format("2006-01-02 15:04:05")))

	if agent.LastActive != nil {
		output.WriteString(fmt.Sprintf("Last Active: %s\n", agent.LastActive.Format("2006-01-02 15:04:05")))
	}

	return output.String()
}

// FormatProjectsTable formats projects as a table
func FormatProjectsTable(projects []models.Project) string {
	if len(projects) == 0 {
		return "No projects found."
	}

	headers := []string{"ID", "Name", "Type", "Status", "Priority", "Progress", "Created"}
	var data [][]string

	for _, project := range projects {
		data = append(data, []string{
			project.ID.String()[:8] + "...",
			project.Name,
			project.Type,
			project.Status,
			project.Priority,
			fmt.Sprintf("%d%%", project.Progress),
			project.CreatedAt.Format("2006-01-02"),
		})
	}

	return FormatTable(headers, data)
}

// FormatTasksTable formats tasks as a table
func FormatTasksTable(tasks []models.Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	headers := []string{"ID", "Title", "Type", "Status", "Priority", "Progress", "Created"}
	var data [][]string

	for _, task := range tasks {
		data = append(data, []string{
			task.ID.String()[:8] + "...",
			task.Title,
			task.Type,
			task.Status,
			task.Priority,
			fmt.Sprintf("%d%%", task.Progress),
			task.CreatedAt.Format("2006-01-02"),
		})
	}

	return FormatTable(headers, data)
}

// FormatAgentsTable formats agents as a table
func FormatAgentsTable(agents []models.Agent) string {
	if len(agents) == 0 {
		return "No agents found."
	}

	headers := []string{"ID", "Name", "Type", "Status", "Tasks", "Success Rate", "Created"}
	var data [][]string

	for _, agent := range agents {
		data = append(data, []string{
			agent.ID.String()[:8] + "...",
			agent.Name,
			agent.Type,
			agent.Status,
			fmt.Sprintf("%d", agent.TaskCount),
			fmt.Sprintf("%.1f%%", agent.SuccessRate),
			agent.CreatedAt.Format("2006-01-02"),
		})
	}

	return FormatTable(headers, data)
}

// FormatJSON formats a struct as pretty JSON
func FormatJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(data), nil
}

// FormatDuration formats a duration in a human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// FormatBytes formats bytes in a human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ValidateID validates a UUID string
func ValidateID(id string) error {
	if len(id) != 36 {
		return fmt.Errorf("invalid ID format: must be 36 characters")
	}
	return nil
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
