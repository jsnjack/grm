package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const listPattern = "%-40s %s\n"

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgs, err := loadInstalledFromDB()
		if err != nil {
			return err
		}
		if len(pkgs) > 0 {
			fmt.Printf(listPattern, "Package", "Version")
			for _, p := range pkgs {
				fmt.Printf(listPattern, p.Owner+"/"+p.Repo, p.Version)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
