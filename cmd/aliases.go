/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// KnownAliases is a list of well-known repositories to simplify binary
// installation from a release
var KnownAliases = map[string]string{
	"chromedriver": "jsnjack/chromedriver",
	"geckodriver":  "mozilla/geckodriver",
	"gotop":        "xxxserxxx/gotop",
	"grm":          "jsnjack/grm",
	"kazy":         "jsnjack/kazy-go",
}

const aliasesPattern = "%-20s %-40s\n"

// aliasesCmd represents the aliases command
var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Print table of known package aliases",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(aliasesPattern, "Alias", "Full package name")
		for k, v := range KnownAliases {
			fmt.Printf(aliasesPattern, k, v)
		}
	},
}

func init() {
	rootCmd.AddCommand(aliasesCmd)
}
