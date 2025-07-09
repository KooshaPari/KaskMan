package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kooshapari/kodevibe-go/internal/config"
)

func newInitCommand() *cobra.Command {
	var (
		force      bool
		template   string
		configOnly bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new KodeVibe project",
		Long: `Initialize a new KodeVibe project with default configuration and optional template files.

This command creates:
• Default configuration file (.kodevibe.yaml)
• Optional template files for common use cases
• Example configuration for different project types

Examples:
  kodevibe init                             # Initialize in current directory
  kodevibe init ./my-project               # Initialize in specific directory
  kodevibe init --template go              # Initialize with Go template
  kodevibe init --config-only              # Only create config file`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := "."
			if len(args) > 0 {
				directory = args[0]
			}
			return runInit(directory, force, template, configOnly)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	cmd.Flags().StringVar(&template, "template", "", "project template (go, javascript, python, java, etc.)")
	cmd.Flags().BoolVar(&configOnly, "config-only", false, "only create configuration file")

	return cmd
}

func newVersionCommand(version, buildTime, commit string) *cobra.Command {
	var (
		short bool
	)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Display detailed version information including build details and system info.

Examples:
  kodevibe version                          # Show full version info
  kodevibe version --short                  # Show only version number`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(version, buildTime, commit, short)
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "show only version number")

	return cmd
}

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for KodeVibe.

The completion script needs to be sourced or installed to enable tab completion.

Examples:
  # Bash
  kodevibe completion bash > /etc/bash_completion.d/kodevibe
  
  # Zsh (add to ~/.zshrc)
  kodevibe completion zsh > ~/.kodevibe_completion
  echo "source ~/.kodevibe_completion" >> ~/.zshrc
  
  # Fish
  kodevibe completion fish > ~/.config/fish/completions/kodevibe.fish
  
  # PowerShell (add to profile)
  kodevibe completion powershell > kodevibe.ps1`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletion(cmd.Root(), args[0])
		},
	}

	return cmd
}

func newDocsCommand() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate documentation",
		Long: `Generate documentation for KodeVibe in various formats.

Examples:
  kodevibe docs                             # Generate markdown docs
  kodevibe docs --format html              # Generate HTML docs
  kodevibe docs --output ./docs            # Output to specific directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDocs(cmd.Root(), format, output)
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "documentation format (markdown, html, man)")
	cmd.Flags().StringVar(&output, "output", "./docs", "output directory")

	return cmd
}

func runInit(directory string, force bool, template string, configOnly bool) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create configuration file
	configPath := filepath.Join(directory, ".kodevibe.yaml")
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
	}

	// Create configuration based on template
	cfg := config.Default()
	
	// Customize configuration based on template
	if template != "" {
		customizeConfigForTemplate(cfg, template)
	}

	// Save configuration
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✅ Created configuration file: %s\n", configPath)

	// Create template files if requested
	if !configOnly {
		if err := createTemplateFiles(directory, template, force); err != nil {
			return fmt.Errorf("failed to create template files: %w", err)
		}
	}

	// Print success message and next steps
	fmt.Printf("\n🎉 KodeVibe project initialized successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Review and customize the configuration: %s\n", configPath)
	fmt.Printf("  2. Run your first scan: kodevibe scan %s\n", directory)
	fmt.Printf("  3. Start the API server: kodevibe server\n")
	fmt.Printf("  4. View available checkers: kodevibe checkers list\n")

	return nil
}

func runVersion(version, buildTime, commit string, short bool) error {
	if short {
		fmt.Println(version)
		return nil
	}

	fmt.Printf("KodeVibe %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return nil
}

func runCompletion(rootCmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func runDocs(rootCmd *cobra.Command, format, output string) error {
	// Create output directory
	if err := os.MkdirAll(output, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch format {
	case "markdown":
		return generateMarkdownDocs(rootCmd, output)
	case "html":
		return generateHTMLDocs(rootCmd, output)
	case "man":
		return generateManDocs(rootCmd, output)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// Helper functions

func customizeConfigForTemplate(cfg *config.Config, template string) {
	switch strings.ToLower(template) {
	case "go", "golang":
		cfg.Scanner.IncludePatterns = []string{"*.go", "go.mod", "go.sum"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "vendor/", "*.pb.go")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "performance", "git"}
		
	case "javascript", "js", "node", "nodejs":
		cfg.Scanner.IncludePatterns = []string{"*.js", "*.jsx", "*.ts", "*.tsx", "package.json"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "node_modules/", "dist/", "build/")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "dependency", "git"}
		
	case "python", "py":
		cfg.Scanner.IncludePatterns = []string{"*.py", "requirements.txt", "setup.py", "pyproject.toml"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "__pycache__/", "*.pyc", ".venv/", "venv/")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "dependency", "git"}
		
	case "java":
		cfg.Scanner.IncludePatterns = []string{"*.java", "pom.xml", "build.gradle"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "target/", "build/", "*.class")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "performance", "dependency"}
		
	case "rust":
		cfg.Scanner.IncludePatterns = []string{"*.rs", "Cargo.toml", "Cargo.lock"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "target/")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "performance", "git"}
		
	case "cpp", "c++", "c":
		cfg.Scanner.IncludePatterns = []string{"*.cpp", "*.c", "*.h", "*.hpp", "CMakeLists.txt", "Makefile"}
		cfg.Scanner.IgnorePatterns = append(cfg.Scanner.IgnorePatterns, "build/", "*.o", "*.a", "*.so")
		cfg.Vibes.EnabledCheckers = []string{"code", "security", "performance"}
	}
}

func createTemplateFiles(directory, template string, force bool) error {
	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(directory, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) || force {
		gitignoreContent := generateGitignoreContent(template)
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		fmt.Printf("✅ Created .gitignore file\n")
	}

	// Create README.md if it doesn't exist
	readmePath := filepath.Join(directory, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) || force {
		readmeContent := generateReadmeContent(template)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
		fmt.Printf("✅ Created README.md file\n")
	}

	return nil
}

func generateGitignoreContent(template string) string {
	base := `# KodeVibe
.kodevibe/
kodevibe-reports/

# OS
.DS_Store
Thumbs.db

# IDE
.vscode/
.idea/
*.swp
*.swo
*~
`

	switch strings.ToLower(template) {
	case "go", "golang":
		return base + `
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
vendor/
`
	case "javascript", "js", "node", "nodejs":
		return base + `
# Node.js
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
.npm
.eslintcache
dist/
build/
`
	case "python", "py":
		return base + `
# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg
.venv
venv/
ENV/
env/
`
	default:
		return base
	}
}

func generateReadmeContent(template string) string {
	return fmt.Sprintf(`# Project

This project uses KodeVibe for code analysis and quality checking.

## Getting Started

1. Install KodeVibe (if not already installed)
2. Run code analysis: kodevibe scan .
3. View available checkers: kodevibe checkers list
4. Start API server: kodevibe server

## Configuration

The project configuration is stored in .kodevibe.yaml. You can modify it to customize:
- Which checkers to enable/disable
- File patterns to include/exclude
- Output formats and limits
- Scanner settings

## Analysis Results

KodeVibe will analyze your code for:
- Security vulnerabilities
- Performance issues
- Code quality problems
- File organization issues
- Git repository health
- Dependency management
- Documentation quality

## Template: %s

This project was initialized with the %s template.
`, template, template)
}

func generateMarkdownDocs(rootCmd *cobra.Command, output string) error {
	// This would generate markdown documentation
	// For simplicity, just create a basic overview
	docsContent := `# KodeVibe CLI Documentation

## Commands

### scan
Scan code for issues and vulnerabilities.

### checkers
Manage and inspect code checkers.

### config
Manage KodeVibe configuration.

### server
Start the KodeVibe API server.

### init
Initialize a new KodeVibe project.

### version
Show version information.

### completion
Generate shell completion scripts.

### docs
Generate documentation.

For detailed usage of each command, use: kodevibe <command> --help
`

	docPath := filepath.Join(output, "README.md")
	if err := os.WriteFile(docPath, []byte(docsContent), 0644); err != nil {
		return fmt.Errorf("failed to write documentation: %w", err)
	}

	fmt.Printf("✅ Generated markdown documentation: %s\n", docPath)
	return nil
}

func generateHTMLDocs(rootCmd *cobra.Command, output string) error {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <title>KodeVibe CLI Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        h2 { color: #666; }
        code { background: #f5f5f5; padding: 2px 4px; border-radius: 3px; }
        pre { background: #f5f5f5; padding: 10px; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>KodeVibe CLI Documentation</h1>
    <h2>Commands</h2>
    <p>For detailed usage information, use <code>kodevibe &lt;command&gt; --help</code></p>
</body>
</html>`

	docPath := filepath.Join(output, "index.html")
	if err := os.WriteFile(docPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML documentation: %w", err)
	}

	fmt.Printf("✅ Generated HTML documentation: %s\n", docPath)
	return nil
}

func generateManDocs(rootCmd *cobra.Command, output string) error {
	// This would generate man pages - simplified for example
	manContent := `.TH KODEVIBE 1 "January 2024" "KodeVibe 1.0.0" "User Commands"
.SH NAME
kodevibe \- Advanced code analysis and quality checking tool
.SH SYNOPSIS
.B kodevibe
[\fIOPTION\fR]... \fICOMMAND\fR [\fIARG\fR]...
.SH DESCRIPTION
KodeVibe is a comprehensive code analysis tool that provides security vulnerability detection, performance analysis, and code quality assessment.
`

	manPath := filepath.Join(output, "kodevibe.1")
	if err := os.WriteFile(manPath, []byte(manContent), 0644); err != nil {
		return fmt.Errorf("failed to write man page: %w", err)
	}

	fmt.Printf("✅ Generated man page: %s\n", manPath)
	return nil
}