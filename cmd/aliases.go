/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"sort"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		names := make([]string, 0, len(KnownAliases))
		for name := range KnownAliases {
			names = append(names, name)
		}
		sort.Strings(names)
		var rows [][]string
		for _, name := range names {
			status := ""
			if _, ok := config.Packages[KnownAliases[name]]; ok {
				status = green(sLocked + " installed")
			}
			rows = append(rows, []string{cyan(name), pkgLabel(KnownAliases[name]), status})
		}
		printTable("", []string{"ALIAS", "PACKAGE", ""}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aliasesCmd)
}
