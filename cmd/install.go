package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v30/github"
	"github.com/schollz/progressbar/v2"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <repository>",
	Short: "Install a package from github releases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a pakage name, e.g. jsnjack/kazy-go")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		owner, repo, err := cleanPackage(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		client := github.NewClient(nil)
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Found release %s\n", release.GetTagName())
		getDownloadURL(release.Assets)
		// for _, item := range release.Assets {
		// 	fmt.Printf("  Assets: %s %s", item.GetName(), item.GetContentType())
		// 	fmt.Println()
		// 	downloadFile(item.GetBrowserDownloadURL())
		// }
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
	req, err := http.NewRequest("GET", url, nil)
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

func getDownloadURL(assets []*github.ReleaseAsset) {
	for _, item := range assets {
		fmt.Printf("  Asset: %s %s", item.GetName(), item.GetContentType())
	}
}
