package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

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
				tableHeader([]int{40}, []string{"Package", "Description"})
				client := CreateClient()
				for _, p := range config.Packages {
					var description string
					repo, _, err := client.Repositories.Get(context.Background(), p.Owner, p.Repo)
					if err != nil {
						description = red(err.Error())
					} else {
						description = repo.GetDescription()
					}
					tableRow(padCol(p.GetFullName(), 40, nil), description)
				}
				return nil
			} else if listFlat {
				for _, p := range config.Packages {
					fmt.Printf("%s ", p.GetFullName())
				}
				fmt.Println()
				return nil
			} else {
				tableHeader([]int{40, 20, 20}, []string{"Package", "Version", "Locked", "Filter"})
				for _, p := range config.Packages {
					var lockedStyle func(string) string
					if p.Locked {
						lockedStyle = yellow
					}
					tableRow(
						padCol(p.GetFullName(), 40, nil),
						padCol(p.Version, 20, cyan),
						padCol(p.GetVerboseLocked(), 20, lockedStyle),
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
