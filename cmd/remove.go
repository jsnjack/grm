package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <package> [<package>...]",
	Short: "Remove a package",
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
			var pkg *Package
			// Retrieve binary location
			err := DB.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(PackagesBucket))
				c := b.Cursor()
				for key, _ := c.First(); key != nil; key, _ = c.Next() {
					if string(key) == item {
						pb := b.Bucket(key)
						if pb == nil {
							return fmt.Errorf("Bucket %s doesn't exist", item)
						}
						p, err := createPackageFromDB(item, pb)
						if err != nil {
							return err
						}
						pkg = p
						return nil
					}
				}
				return fmt.Errorf("Package %s is not installed", item)
			})
			if err != nil {
				return err
			}
			if pkg.IsLocked() {
				fmt.Printf("Package %s is locked\n", pkg.GetFullName())
				continue
			}
			if ok := askForConfirmation(fmt.Sprintf("Are you sure you want to remove %s?", item)); !ok {
				return nil
			}
			// Remove binary
			err = removeBinary(pkg.Filename)
			if err != nil {
				return err
			}
			// Clean db
			err = DB.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(PackagesBucket)
				return bucket.DeleteBucket([]byte(item))
			})
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
