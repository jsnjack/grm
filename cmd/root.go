package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// traceLogPath is where --trace writes diagnostic output, truncated on
// every run.
const traceLogPath = "/tmp/grm.log"

// ConfigFile is the path to grm's configuration file, resolved by
// resolveConfigFile once flags have been parsed.
var ConfigFile string

var rootYes bool
var rootToken string
var rootDebug bool
var rootTrace bool
var rootConfigFile string
var rootNoProgress bool

var loggerCleanup = func() {}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grm",
	Short: "A package manager for GitHub releases",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loggerCleanup = initLogger(rootDebug, rootTrace, traceLogPath)

		path, err := resolveConfigFile(rootConfigFile)
		if err != nil {
			return fmt.Errorf("resolve config file: %w", err)
		}
		ConfigFile = path
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer loggerCleanup()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().BoolVarP(&rootYes, "yes", "y", false, "Confirm all")
	rootCmd.PersistentFlags().BoolVarP(&rootDebug, "debug", "d", false, "Debug-level logging on stderr.")
	rootCmd.PersistentFlags().BoolVar(&rootTrace, "trace", false, "Trace-level logs to "+traceLogPath+" (truncated each run).")
	rootCmd.PersistentFlags().StringVar(&rootToken, "token", "", "GitHub API token")
	rootCmd.PersistentFlags().StringVarP(&rootConfigFile, "config", "c", "", "Path to the config file (default ~/.config/grm/grm.yaml)")
	rootCmd.PersistentFlags().BoolVar(&rootNoProgress, "no-progress", false, "Disable progress bar")

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	// Registered explicitly (no shorthand) so cobra doesn't claim the -v
	// shorthand for it — -v is reserved for future use, long-form only.
	rootCmd.Flags().Bool("version", false, "Print the version and exit")
}
