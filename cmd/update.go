package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v93/github"
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
		sort.Slice(toCheck, func(i, j int) bool {
			return strings.ToLower(toCheck[i].GetFullName()) < strings.ToLower(toCheck[j].GetFullName())
		})

		// Phase 1: check all packages for updates in parallel
		started := time.Now()
		results := make([]checkResult, len(toCheck))
		var wg sync.WaitGroup
		var done atomic.Int32
		for i, p := range toCheck {
			wg.Add(1)
			go func(i int, p Package) {
				defer wg.Done()
				defer done.Add(1)
				r := checkResult{pkg: p}
				if p.Locked {
					r.locked = true
					results[i] = r
					return
				}
				release, err := selectRelease(&Package{Owner: p.Owner, Repo: p.Repo, VersionFilter: p.VersionFilter})
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
		waitWithProgress(&wg, &done, len(toCheck), started)

		// Phase 2: print grouped results
		var toUpdate, locked, failed []checkResult
		current := 0
		var updateErrors []error
		for _, r := range results {
			switch {
			case r.locked:
				locked = append(locked, r)
			case r.err != nil:
				failed = append(failed, r)
				updateErrors = append(updateErrors, fmt.Errorf("check %s for updates: %w", r.pkg.GetFullName(), r.err))
			case r.latest:
				current++
			default:
				toUpdate = append(toUpdate, r)
			}
		}
		if len(toUpdate) > 0 {
			msgSection("Updates", cyan(fmt.Sprint(len(toUpdate))))
			var rows [][]string
			for _, r := range toUpdate {
				released := ""
				if !r.release.GetPublishedAt().IsZero() {
					released = "released " + r.release.GetPublishedAt().Format("2006-01-02")
				}
				version := r.pkg.Version
				if r.pkg.Incomplete && version == r.release.GetTagName() {
					version = "incomplete"
				}
				rows = append(rows, []string{pkgLabel(r.pkg.GetFullName()), versionChange(version, r.release.GetTagName()), dim(released)})
			}
			printTable("    ", nil, rows)
		}
		if len(locked) > 0 {
			msgSection("Locked", yellow(fmt.Sprint(len(locked))))
			var rows [][]string
			for _, r := range locked {
				rows = append(rows, []string{yellow(sLocked) + " " + pkgLabel(r.pkg.GetFullName()), cyan(r.pkg.Version), dim("not checked " + sDot + " grm unlock " + r.pkg.GetFullName())})
			}
			printTable("    ", nil, rows)
		}
		if len(failed) > 0 {
			msgSection("Failed", red(fmt.Sprint(len(failed))))
			for _, r := range failed {
				fmt.Printf("    %s %s  %s\n", red(sCross), pkgLabel(r.pkg.GetFullName()), red(r.err.Error()))
				if fix := errorFix(r.err); fix != "" {
					fmt.Printf("      %s\n", dim("Fix: "+fix))
				}
			}
		}
		msgSection("Up to date", green(fmt.Sprint(current)))
		fmt.Println()

		if len(toUpdate) == 0 {
			if len(updateErrors) != 0 {
				msgResult(false, "No updates %s", dim(joinDetails(plural(len(failed), "check failed", "checks failed"), "exit status 1")))
				return silentError{errors.Join(updateErrors...)}
			}
			msgResult(true, "All packages are up to date")
			return nil
		}

		// Phase 3: single confirmation
		ok, err := askForConfirmation(fmt.Sprintf("Update %s?", plural(len(toUpdate), "package", "packages")), true)
		if err != nil {
			return errors.Join(append(updateErrors, err)...)
		}
		if !ok {
			msgResult(len(updateErrors) == 0, "Nothing updated")
			if len(updateErrors) != 0 {
				return silentError{errors.Join(updateErrors...)}
			}
			return nil
		}

		// Phase 4: install sequentially (interactive prompts + config writes)
		updated, updateFailed := 0, 0
		for _, r := range toUpdate {
			fmt.Println()
			err := installRelease(r.release, &r.pkg, installOptions{})
			if err != nil {
				printError(os.Stdout, err)
				updateFailed++
				updateErrors = append(updateErrors, fmt.Errorf("update %s: %w", r.pkg.GetFullName(), err))
				continue
			}
			updated++
		}
		fmt.Println()
		summary := joinDetails(plural(updated, "package updated", "packages updated"))
		var problems []string
		if updateFailed > 0 {
			problems = append(problems, fmt.Sprintf("%d failed", updateFailed))
		}
		if len(failed) > 0 {
			problems = append(problems, plural(len(failed), "check failed", "checks failed"))
		}
		if len(updateErrors) == 0 {
			msgResult(true, "%s %s", summary, dim(sDot+" "+formatDuration(time.Since(started))))
			return nil
		}
		msgResult(false, "%s %s", summary, dim(joinDetails(append(problems, "exit status 1")...)))
		return silentError{errors.Join(updateErrors...)}
	},
}

// waitWithProgress shows one progress line while the checks run. Without a
// terminal it only waits.
func waitWithProgress(wg *sync.WaitGroup, done *atomic.Int32, total int, started time.Time) {
	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()
	render := func(n int) string {
		const width = 24
		filled := width * n / total
		bar := cyan(strings.Repeat("━", filled)) + dim(strings.Repeat("─", width-filled))
		return fmt.Sprintf("  %s  %s %s", dim(fmt.Sprintf("Checking %s", plural(total, "package", "packages"))), bar, dim(fmt.Sprintf("%d/%d", n, total)))
	}
	if !isTTY {
		<-finished
		fmt.Printf("  Checked %s in %s\n", plural(total, "package", "packages"), formatDuration(time.Since(started)))
		return
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-finished:
			fmt.Printf("\r\033[2K%s %s\n", render(total), dim(formatDuration(time.Since(started))))
			return
		case <-ticker.C:
			fmt.Printf("\r\033[2K%s", render(int(done.Load())))
		}
	}
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
