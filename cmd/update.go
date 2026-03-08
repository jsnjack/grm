package cmd

import (
	"fmt"
	"sync"

	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
)

// checkResult holds the outcome of checking a single package for updates
type checkResult struct {
	pkg     Package
	release *github.RepositoryRelease
	err     error
	locked  bool
	latest  bool
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [<package>]",
	Short: "Update installed packages",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		switch len(args) {
		case 0:
			break
		case 1:
			for _, item := range args {
				_, err := CreatePackage(item)
				if err != nil {
					return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go), got %s", item)
				}
			}
		default:
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}

		// Collect packages to check
		var toCheck []Package
		for _, p := range config.Packages {
			if len(args) == 1 {
				if args[0] != p.Owner+"/"+p.Repo {
					continue
				}
			}
			toCheck = append(toCheck, p)
		}

		if len(toCheck) == 0 {
			fmt.Println("No packages to check")
			return nil
		}

		// Phase 1: check all packages for updates in parallel
		fmt.Printf("Checking %d package(s) for updates...\n", len(toCheck))
		results := make([]checkResult, len(toCheck))
		var wg sync.WaitGroup
		for i, p := range toCheck {
			wg.Add(1)
			go func(i int, p Package) {
				defer wg.Done()
				r := checkResult{pkg: p}
				if p.Locked {
					r.locked = true
					results[i] = r
					return
				}
				release, err := selectRelease(&Package{Owner: p.Owner, Repo: p.Repo})
				if err != nil {
					r.err = err
					results[i] = r
					return
				}
				r.release = release
				if p.VerifyVersion(release.GetTagName()) == nil {
					r.latest = true
				}
				results[i] = r
			}(i, p)
		}
		wg.Wait()

		// Phase 2: print summary
		fmt.Println()
		var toUpdate []checkResult
		for _, r := range results {
			name := r.pkg.GetFullName()
			switch {
			case r.locked:
				fmt.Printf("  %-40s locked\n", name)
			case r.err != nil:
				fmt.Printf("  %-40s error: %s\n", name, r.err)
			case r.latest:
				fmt.Printf("  %-40s latest (%s)\n", name, r.pkg.Version)
			default:
				fmt.Printf("  %-40s %s -> %s\n", name, r.pkg.Version, r.release.GetTagName())
				toUpdate = append(toUpdate, r)
			}
		}

		if len(toUpdate) == 0 {
			fmt.Println("\nAll packages are up to date")
			return nil
		}

		// Phase 3: single confirmation
		fmt.Printf("\n%d package(s) can be updated\n", len(toUpdate))
		if !askForConfirmation("Proceed with update?") {
			return nil
		}

		// Phase 4: install sequentially (interactive prompts + config writes)
		fmt.Println()
		for _, r := range toUpdate {
			fmt.Printf("Updating %s...\n", r.pkg.GetFullName())
			err := installRelease(r.release, &r.pkg)
			if err != nil {
				fmt.Printf("  error: %s\n", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
