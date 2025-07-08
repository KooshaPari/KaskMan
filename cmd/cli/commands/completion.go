package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewCompletionCommand creates the completion command
func NewCompletionCommand() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `Generate completion script for your shell.

To load completions for different shells:

Bash:
  $ source <(kaskman completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kaskman completion bash > /etc/bash_completion.d/kaskman
  # macOS:
  $ kaskman completion bash > /usr/local/etc/bash_completion.d/kaskman

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kaskman completion zsh > "${fpath[1]}/_kaskman"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ kaskman completion fish | source

  # To load completions for each session, execute once:
  $ kaskman completion fish > ~/.config/fish/completions/kaskman.fish

PowerShell:
  PS> kaskman completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> kaskman completion powershell > kaskman.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]

			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", shell)
			}
		},
	}

	// Install command
	installCmd := &cobra.Command{
		Use:       "install [bash|zsh|fish]",
		Short:     "Install completion script",
		Long:      "Install completion script to the appropriate location for your shell",
		ValidArgs: []string{"bash", "zsh", "fish"},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				// Try to detect shell
				shell = detectShell()
				if shell == "" {
					return fmt.Errorf("could not detect shell. Please specify: bash, zsh, or fish")
				}
			}

			return installCompletion(cmd.Root(), shell)
		},
	}

	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:       "uninstall [bash|zsh|fish]",
		Short:     "Uninstall completion script",
		Long:      "Remove completion script from your shell",
		ValidArgs: []string{"bash", "zsh", "fish"},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell()
				if shell == "" {
					return fmt.Errorf("could not detect shell. Please specify: bash, zsh, or fish")
				}
			}

			return uninstallCompletion(shell)
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show completion status",
		Long:  "Show the installation status of completion scripts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showCompletionStatus()
		},
	}

	// Add subcommands
	completionCmd.AddCommand(installCmd)
	completionCmd.AddCommand(uninstallCmd)
	completionCmd.AddCommand(statusCmd)

	return completionCmd
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ""
	}

	switch filepath.Base(shell) {
	case "bash":
		return "bash"
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	default:
		return ""
	}
}

func installCompletion(rootCmd *cobra.Command, shell string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	var completionDir, completionFile string

	switch shell {
	case "bash":
		// Try different locations
		completionDirs := []string{
			"/usr/local/etc/bash_completion.d",
			"/etc/bash_completion.d",
			filepath.Join(homeDir, ".bash_completion.d"),
		}

		for _, dir := range completionDirs {
			if _, err := os.Stat(dir); err == nil {
				completionDir = dir
				break
			}
		}

		if completionDir == "" {
			// Create local completion directory
			completionDir = filepath.Join(homeDir, ".bash_completion.d")
			if err := os.MkdirAll(completionDir, 0755); err != nil {
				return fmt.Errorf("failed to create completion directory: %w", err)
			}
		}

		completionFile = filepath.Join(completionDir, "kaskman")

	case "zsh":
		// Check if oh-my-zsh is installed
		if _, err := os.Stat(filepath.Join(homeDir, ".oh-my-zsh")); err == nil {
			completionDir = filepath.Join(homeDir, ".oh-my-zsh", "completions")
		} else {
			// Use default zsh completion directory
			completionDir = filepath.Join(homeDir, ".zsh", "completions")
		}

		if err := os.MkdirAll(completionDir, 0755); err != nil {
			return fmt.Errorf("failed to create completion directory: %w", err)
		}

		completionFile = filepath.Join(completionDir, "_kaskman")

	case "fish":
		completionDir = filepath.Join(homeDir, ".config", "fish", "completions")
		if err := os.MkdirAll(completionDir, 0755); err != nil {
			return fmt.Errorf("failed to create completion directory: %w", err)
		}

		completionFile = filepath.Join(completionDir, "kaskman.fish")

	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Generate completion
	file, err := os.Create(completionFile)
	if err != nil {
		return fmt.Errorf("failed to create completion file: %w", err)
	}
	defer file.Close()

	switch shell {
	case "bash":
		err = rootCmd.GenBashCompletion(file)
	case "zsh":
		err = rootCmd.GenZshCompletion(file)
	case "fish":
		err = rootCmd.GenFishCompletion(file, true)
	}

	if err != nil {
		os.Remove(completionFile)
		return fmt.Errorf("failed to generate completion: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Completion installed for %s at %s", shell, completionFile))

	// Show instructions
	switch shell {
	case "bash":
		utils.PrintInfo("To load completions, add this to your ~/.bashrc:")
		fmt.Printf("  source %s\n", completionFile)
		utils.PrintInfo("Or reload your shell to activate completions")

	case "zsh":
		utils.PrintInfo("Completion installed. You may need to:")
		fmt.Println("  1. Add the completion directory to your fpath in ~/.zshrc:")
		fmt.Printf("     fpath=(%s $fpath)\n", completionDir)
		fmt.Println("  2. Reload your shell or run: autoload -U compinit && compinit")

	case "fish":
		utils.PrintInfo("Completion installed. Restart your shell or run:")
		fmt.Println("  source ~/.config/fish/config.fish")
	}

	return nil
}

func uninstallCompletion(shell string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	var completionFiles []string

	switch shell {
	case "bash":
		completionFiles = []string{
			"/usr/local/etc/bash_completion.d/kaskman",
			"/etc/bash_completion.d/kaskman",
			filepath.Join(homeDir, ".bash_completion.d", "kaskman"),
		}

	case "zsh":
		completionFiles = []string{
			filepath.Join(homeDir, ".oh-my-zsh", "completions", "_kaskman"),
			filepath.Join(homeDir, ".zsh", "completions", "_kaskman"),
		}

	case "fish":
		completionFiles = []string{
			filepath.Join(homeDir, ".config", "fish", "completions", "kaskman.fish"),
		}

	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	removed := false
	for _, file := range completionFiles {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				utils.PrintWarning(fmt.Sprintf("Failed to remove %s: %v", file, err))
			} else {
				utils.PrintSuccess(fmt.Sprintf("Removed completion file: %s", file))
				removed = true
			}
		}
	}

	if !removed {
		utils.PrintInfo(fmt.Sprintf("No completion files found for %s", shell))
	}

	return nil
}

func showCompletionStatus() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	utils.PrintHeader("Completion Status")

	// Check bash
	bashFiles := []string{
		"/usr/local/etc/bash_completion.d/kaskman",
		"/etc/bash_completion.d/kaskman",
		filepath.Join(homeDir, ".bash_completion.d", "kaskman"),
	}

	bashInstalled := false
	for _, file := range bashFiles {
		if _, err := os.Stat(file); err == nil {
			utils.PrintSuccess(fmt.Sprintf("Bash completion: %s", file))
			bashInstalled = true
			break
		}
	}
	if !bashInstalled {
		utils.PrintInfo("Bash completion: Not installed")
	}

	// Check zsh
	zshFiles := []string{
		filepath.Join(homeDir, ".oh-my-zsh", "completions", "_kaskman"),
		filepath.Join(homeDir, ".zsh", "completions", "_kaskman"),
	}

	zshInstalled := false
	for _, file := range zshFiles {
		if _, err := os.Stat(file); err == nil {
			utils.PrintSuccess(fmt.Sprintf("Zsh completion: %s", file))
			zshInstalled = true
			break
		}
	}
	if !zshInstalled {
		utils.PrintInfo("Zsh completion: Not installed")
	}

	// Check fish
	fishFile := filepath.Join(homeDir, ".config", "fish", "completions", "kaskman.fish")
	if _, err := os.Stat(fishFile); err == nil {
		utils.PrintSuccess(fmt.Sprintf("Fish completion: %s", fishFile))
	} else {
		utils.PrintInfo("Fish completion: Not installed")
	}

	// Show current shell
	currentShell := detectShell()
	if currentShell != "" {
		utils.PrintInfo(fmt.Sprintf("Current shell: %s", currentShell))
	}

	return nil
}
