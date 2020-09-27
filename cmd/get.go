package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
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
	rootCmd.AddCommand(getCmd)
}
