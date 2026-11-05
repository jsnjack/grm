package cmd

import (
	"os"
	"strings"

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
		token, source := config.Settings["token"], "from config file"
		if rootToken != "" {
			token, source = rootToken, "from --token"
		} else if token == "" && os.Getenv("GITHUB_TOKEN") != "" {
			token, source = os.Getenv("GITHUB_TOKEN"), "from GITHUB_TOKEN"
		}
		tokenValue := maskToken(token)
		if token == "" {
			tokenValue, source = dim("not set"), "anonymous requests are rate limited"
		}
		binDir, binSource := config.Settings["bin_dir"], "from config file"
		if binDir == "" {
			binDir, binSource = strings.TrimSuffix(DefaultBinDir, "/"), "default"
		}
		printTable("  ", nil, [][]string{
			{dim("token"), tokenValue, dim(source)},
			{dim("bin_dir"), binDir, dim(binSource)},
			{dim("config"), ConfigFile, ""},
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}

// maskToken keeps the token's prefix and last four characters, enough to
// tell tokens apart without revealing them.
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("•", len(token))
	}
	prefix := ""
	if i := strings.Index(token, "_"); i > 0 && i < 12 {
		prefix = token[:i+1]
	}
	return prefix + strings.Repeat("•", 8) + token[len(token)-4:]
}
