package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const listPattern = "%-40s %-20s %-20s %s\n"
const listRepoDescriptionPattern = "%-40s %s\n"

var listRepoDescription bool
var listFlat bool

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		if len(config.Packages) > 0 {
			if listRepoDescription {
				// Print list of all packages and fetch their description from github
				fmt.Printf(listRepoDescriptionPattern, "Package", "Description")
				client := CreateClient()
				for _, p := range config.Packages {
					var description string
					repo, _, err := client.Repositories.Get(context.Background(), p.Owner, p.Repo)
					if err != nil {
						description = err.Error()
					} else {
						description = repo.GetDescription()
					}
					fmt.Printf(
						listRepoDescriptionPattern,
						p.GetFullName(),
						description,
					)
				}
				return nil
			} else if listFlat {
				for _, p := range config.Packages {
					fmt.Printf("%s ", p.GetFullName())
				}
				fmt.Println()
				return nil
			} else {
				fmt.Printf(listPattern, "Package", "Version", "Locked", "Filter")
				for _, p := range config.Packages {
					fmt.Printf(
						listPattern,
						p.GetFullName(),
						p.Version,
						p.GetVerboseLocked(),
						strings.Join(p.Filter, ", "),
					)
				}
				return nil
			}
		} else {
			cmd.SilenceUsage = true
			fmt.Println("No installed packages")
			return nil
		}
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listRepoDescription, "description", "d", false, "Print description of the repositories")
	listCmd.Flags().BoolVarP(&listFlat, "flat", "f", false, "Print installed packages in flat form")
	rootCmd.AddCommand(listCmd)
}
