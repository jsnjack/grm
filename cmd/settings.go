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
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		for key := range Settings {
			fmt.Printf("%s: %s\n", key, config.Settings[key])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
