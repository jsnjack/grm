/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// KnownAliases is a list of well-known repositories to simplify binary
// installation from a release
var KnownAliases = map[string]string{
	"chromedriver": "jsnjack/chromedriver",
	"geckodriver":  "mozilla/geckodriver",
	"gotop":        "xxxserxxx/gotop",
	"grm":          "jsnjack/grm",
	"k6":           "grafana/k6",
	"kazy":         "jsnjack/kazy-go",
	"mech":         "jsnjack/mech",
	"monova":       "jsnjack/monova",
	"selenium":     "SeleniumHQ/selenium",
	"sslcheck":     "jsnjack/sslcheck",
	"sup":          "jsnjack/sup",
}

// aliasesCmd represents the aliases command
var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Print table of known package aliases",
	Run: func(cmd *cobra.Command, args []string) {
		tableHeader([]int{20}, []string{"Alias", "Full package name"})
		for k, v := range KnownAliases {
			tableRow(padCol(k, 20, cyan), v)
		}
	},
}

func init() {
	rootCmd.AddCommand(aliasesCmd)
}
