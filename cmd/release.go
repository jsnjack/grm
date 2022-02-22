package cmd

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/go-github/v32/github"
	"github.com/schollz/progressbar/v2"
	"github.com/spf13/cobra"
)

var releaseFilename []string
var releaseTag string

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release <package> -f <filename> [-f <filename>] -t v<version>",
	Short: "Create a release in GitHub",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		switch len(args) {
		case 0:
			return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go)")
		case 1:
			_, err := CreatePackage(args[0])
			if err != nil {
				return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go), got %s", args[0])
			}
		default:
			return fmt.Errorf("expected 1 argument, got %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		pkg, err := CreatePackage(args[0])
		if err != nil {
			return err
		}

		client := CreateClient()

		// Try to get existing release
		release, _, err := client.Repositories.GetReleaseByTag(context.Background(), pkg.Owner, pkg.Repo, releaseTag)
		if err != nil {
			// Create a release first
			release, _, err = client.Repositories.CreateRelease(context.Background(), pkg.Owner, pkg.Repo, &github.RepositoryRelease{
				TagName: &releaseTag,
			})
		}

		if err != nil {
			return err
		}

		// Upload assets
		for _, item := range releaseFilename {
			fmt.Printf("Uploading %s...\n", item)
			f, err := os.Open(item)
			if err != nil {
				return err
			}
			defer f.Close()

			stat, err := f.Stat()
			if err != nil {
				return err
			}

			if stat.IsDir() {
				return fmt.Errorf("the asset to upload can't be a directory")
			}

			bar := progressbar.NewOptions(
				int(stat.Size()),
				progressbar.OptionSetBytes(int(stat.Size())),
			)

			reader := &ProgressReader{
				r:   f,
				bar: bar,
			}
			defer bar.Clear()

			u := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", pkg.Owner, pkg.Repo, release.GetID(), filepath.Base(item))

			mediaType := mime.TypeByExtension(filepath.Ext(f.Name()))

			req, err := client.NewUploadRequest(u, reader, stat.Size(), mediaType)
			if err != nil {
				return err
			}

			asset := new(github.ReleaseAsset)
			_, err = client.Do(context.Background(), req, asset)
			if err != nil {
				return err
			}
			fmt.Println("  done")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
	releaseCmd.Flags().StringArrayVarP(&releaseFilename, "filename", "f", releaseFilename, "Location of an asset to upload")
	releaseCmd.MarkFlagFilename("filename")
	releaseCmd.MarkFlagRequired("filename")

	releaseCmd.Flags().StringVarP(&releaseTag, "tag", "t", "", "Tag name")
	releaseCmd.MarkFlagRequired("tag")
}
