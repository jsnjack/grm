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
	Use:     "update [<package>]",
	Aliases: []string{"u"},
	Short:   "Update installed packages",
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

		// Show animated pending state while goroutines run (TTY only)
		pendingLines := 0
		if colorEnabled {
			fmt.Println()
			pendingLines++
			msgSync("Checking %d package(s) for updates...", len(toCheck))
			pendingLines++
			fmt.Println()
			pendingLines++
			for _, p := range toCheck {
				msgSync("%-40s %s", p.GetFullName(), dim("checking..."))
				pendingLines++
			}
		}

		wg.Wait()

		// Phase 2: overwrite pending state, then print results
		if pendingLines > 0 {
			cursorUp(pendingLines)
		}
		// Clear-line prefix: only needed when overwriting animation
		cl := ""
		if pendingLines > 0 {
			cl = "\033[2K"
		}

		fmt.Printf("%s\n", cl)
		fmt.Printf("%s", cl)
		msgOK("Checked %d package(s)", len(toCheck))
		fmt.Printf("%s\n", cl)

		var toUpdate []checkResult
		for _, r := range results {
			name := r.pkg.GetFullName()
			fmt.Print(cl)
			switch {
			case r.locked:
				msgLocked("%-40s %s", name, yellow("locked"))
			case r.err != nil:
				msgFail("%-40s %s", name, red("error: "+r.err.Error()))
			case r.latest:
				msgOK("%-40s %s", name, dim("latest ("+r.pkg.Version+")"))
			default:
				msgUpdate("%-40s %s %s %s", name, r.pkg.Version, cyan(sArrow), boldCyan(r.release.GetTagName()))
				toUpdate = append(toUpdate, r)
			}
		}

		if len(toUpdate) == 0 {
			fmt.Println()
			msgOK("All packages are up to date")
			return nil
		}

		// Phase 3: single confirmation
		fmt.Println()
		msgUpdate("%d package(s) can be updated", len(toUpdate))
		ok, err := askForConfirmation("Proceed with update?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		// Phase 4: install sequentially (interactive prompts + config writes)
		fmt.Println()
		for _, r := range toUpdate {
			msgInfo("Updating %s...", bold(r.pkg.GetFullName()))
			err := installRelease(r.release, &r.pkg)
			if err != nil {
				msgFail("%s", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
