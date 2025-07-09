package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Global flags
var (
	configFile string
	verbose    bool
	quiet      bool
	outputFormat string
)

// NewRootCommand creates the root command for the CLI
func NewRootCommand(version, buildTime, commit string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kodevibe",
		Short: "KodeVibe - Advanced code analysis and quality checking tool",
		Long: `KodeVibe is a comprehensive code analysis tool that provides:
• Security vulnerability detection
• Performance analysis and optimization suggestions  
• Code quality and complexity analysis
• File organization and structure checks
• Git repository health analysis
• Dependency management analysis
• Documentation quality assessment

Use KodeVibe to improve code quality, security, and maintainability.`,
		Version: fmt.Sprintf("%s (built %s, commit %s)", version, buildTime, commit),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.kodevibe.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output (only errors)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml, csv)")

	// Add subcommands
	rootCmd.AddCommand(
		newScanCommand(),
		newCheckersCommand(),
		newConfigCommand(),
		newServerCommand(),
		newInitCommand(),
		newVersionCommand(version, buildTime, commit),
		newCompletionCommand(),
		newDocsCommand(),
	)

	return rootCmd
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kodevibe" (without extension)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kodevibe")
	}

	// Environment variables
	viper.SetEnvPrefix("KODEVIBE")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// getOutputWriter returns the appropriate output writer based on quiet flag
func getOutputWriter() *os.File {
	if quiet {
		return nil
	}
	return os.Stdout
}

// getErrorWriter returns the error writer
func getErrorWriter() *os.File {
	return os.Stderr
}