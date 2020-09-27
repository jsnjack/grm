package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// settingsCmd represents the get command
var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Print settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		for key := range Settings {
			fmt.Printf("%s: %s\n", key, loadSettingsFromDB(key))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
