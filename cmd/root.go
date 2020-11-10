package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	bolt "go.etcd.io/bbolt"
)

// DB is the Bolt db
var DB *bolt.DB

// PackagesBucket a bucket with all info about installed packages
var PackagesBucket = []byte("packages")

// SettingsBucket a bucket with settings
var SettingsBucket = []byte("settings")

var cfgFile string

var rootYes bool
var rootToken string
var rootVerbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grm",
	Short: "A package manager for GitHub releases",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer DB.Close()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().BoolVarP(&rootYes, "yes", "y", false, "Confirm all")
	rootCmd.PersistentFlags().BoolVarP(&rootVerbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&rootToken, "token", "", "GitHub API token")

	var err error
	homedir, err := os.UserHomeDir()
	workdir := homedir + "/.config/grm/"
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(workdir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	DB, err = bolt.Open(workdir+"grm.db", 0644, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Bootstrap DB
	err = DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(PackagesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(SettingsBucket)
		return err
	})

	if err != nil {
		log.Fatal(err)
	}
}
