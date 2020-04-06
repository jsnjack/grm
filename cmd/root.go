package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

// DB is the Bolt db
var DB *bolt.DB

// PackagesBucket a bucket with all info about installed packages
var PackagesBucket = []byte("packages")

var cfgFile string

var rootYes bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grm",
	Short: "A package installer for github releases",
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
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().BoolVarP(&rootYes, "yes", "y", false, "Confirm all")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

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
		return err
	})

	if err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".grm" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".grm")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
