package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

// unlockCmd represents the unlock command
var unlockCmd = &cobra.Command{
	Use:   "unlock <package> [<package>...]",
	Short: "Unlock a package",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		if len(args) == 0 {
			return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go)")
		}
		for _, item := range args {
			_, err := CreatePackage(item)
			if err != nil {
				return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go), got %s", item)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		for _, item := range args {
			err := DB.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(PackagesBucket))
				c := b.Cursor()
				for key, _ := c.First(); key != nil; key, _ = c.Next() {
					if string(key) == item {
						pb := b.Bucket(key)
						if pb == nil {
							return fmt.Errorf("Bucket %s doesn't exist", item)
						}
						pb.Put([]byte("locked"), []byte(""))
						return nil
					}
				}
				return fmt.Errorf("Package %s is not installed", item)
			})
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}
