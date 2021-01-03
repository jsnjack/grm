package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// ConfigFile is a file with configuration
var ConfigFile string

var rootYes bool
var rootToken string
var rootVerbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grm",
	Short: "A package manager for GitHub releases",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
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
	rootCmd.PersistentFlags().BoolVarP(&rootVerbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&rootToken, "token", "", "GitHub API token")

	var err error
	homedir, err := os.UserHomeDir()
	workdir := homedir + "/.config/grm/"
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(workdir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	ConfigFile = workdir + "grm.yaml"
}
