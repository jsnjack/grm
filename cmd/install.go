package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v2"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <repository>",
	Short: "Install a package from github releases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a fully qualified pakage name, e.g. jsnjack/kazy-go")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func downloadFile(url string) error {
	urlToGet := "https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip"
	req, err := http.NewRequest("GET", urlToGet, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out io.Writer
	f, _ := os.OpenFile("tmp", os.O_CREATE|os.O_WRONLY, 0644)
	out = f
	defer f.Close()

	bar := progressbar.NewOptions(
		int(resp.ContentLength),
		progressbar.OptionSetBytes(int(resp.ContentLength)),
	)
	out = io.MultiWriter(out, bar)
	io.Copy(out, resp.Body)
	return nil
}
