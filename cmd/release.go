package cmd

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/go-github/v30/github"
	"github.com/schollz/progressbar/v2"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var releaseFilename []string
var releaseTag string
var releaseGithubToken string

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release <package>",
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

		// Retrieve GitHub API token
		if releaseGithubToken == "" {
			releaseGithubToken = os.Getenv("GITHUB_TOKEN")
		}
		if releaseGithubToken == "" {
			return fmt.Errorf("Provide GitHub API token to create a release")
		}

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: releaseGithubToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		client := github.NewClient(tc)

		// Create a release first
		release, _, err := client.Repositories.CreateRelease(ctx, pkg.Owner, pkg.Repo, &github.RepositoryRelease{
			TagName: &releaseTag,
		})
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
				return fmt.Errorf("The asset to upload can't be a directory")
			}

			bar := progressbar.NewOptions(
				int(stat.Size()),
				progressbar.OptionSetBytes(int(stat.Size())),
			)

			reader := &ProgressReader{
				r:   f,
				bar: bar,
			}

			u := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", pkg.Owner, pkg.Repo, release.GetID(), filepath.Base(item))

			mediaType := mime.TypeByExtension(filepath.Ext(f.Name()))

			req, err := client.NewUploadRequest(u, reader, stat.Size(), mediaType)
			if err != nil {
				return err
			}

			asset := new(github.ReleaseAsset)
			_, err = client.Do(ctx, req, asset)
			if err != nil {
				return err
			}
			fmt.Println("")
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

	releaseCmd.Flags().StringVarP(&releaseGithubToken, "token", "g", "", "GitHub API token")
}
