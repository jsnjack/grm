package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// setTokenCmd represents the setToken command
var setTokenCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Modify settings",
	Long:  "Available keys are:\n" + generateSettingsHelp(),
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		if len(args) < 2 {
			return fmt.Errorf("not enough arguments")
		}
		for key := range Settings {
			if key == args[0] {
				return nil
			}
		}
		return fmt.Errorf("Unknown key: %s", args[0])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		err = config.PutSetting(args[0], args[1])
		if err == nil {
			fmt.Println("ok")
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(setTokenCmd)
}

func generateSettingsHelp() string {
	var msg string
	for key, value := range Settings {
		if msg != "" {
			msg += "\n"
		}
		msg += fmt.Sprintf(" - %s - %s", key, value)
	}
	return msg
}
