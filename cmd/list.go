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

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgs, err := loadAllInstalledFromDB()
		if err != nil {
			return err
		}
		if len(pkgs) > 0 {
			if listRepoDescription {
				// Print list of all packages and fetch their description from github
				fmt.Printf(listRepoDescriptionPattern, "Package", "Description")
				client := CreateClient()
				for _, p := range pkgs {
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
			} else {
				fmt.Printf(listPattern, "Package", "Version", "Locked", "Filter")
				for _, p := range pkgs {
					fmt.Printf(
						listPattern,
						p.GetFullName(),
						p.Version,
						p.GetVerboseLocked(),
						fmt.Sprintf(strings.Join(p.Filter, ", ")),
					)
				}
			}
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listRepoDescription, "description", "d", false, "Print repository description")
	rootCmd.AddCommand(listCmd)
}
