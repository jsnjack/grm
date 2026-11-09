package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
)

type installOperations struct {
	binary func(string, string) error
	system func(string) error
	remove func(InstalledComponent) error
}

func componentOperations() installOperations {
	return installOperations{binary: installBinaryTo, system: installSystemPackage, remove: removeComponent}
}

func removeComponent(component InstalledComponent) error {
	if component.SystemPkgName != "" {
		return removeSystemPackage(component)
	}
	return removeBinary(component.Filename)
}

// confirmInstallPlan prints the plan and asks for confirmation only when it
// removes previously installed components. Installing exactly what was
// requested proceeds without a prompt, so non-interactive runs keep working.
func confirmInstallPlan(pkg *Package, tag string, plan []plannedComponent) (bool, error) {
	installed := make(map[string]InstalledComponent)
	for _, old := range pkg.installedComponents() {
		installed[old.identity()] = old
	}
	wanted := make(map[string]bool)
	movedTo := make(map[string]string)
	var rows [][]string
	for _, item := range plan {
		component := item.component
		wanted[component.identity()] = true
		if component.Filename != "" {
			movedTo[filepath.Base(component.Filename)] = filepath.Dir(component.Filename)
		}
		symbol, change := green(sAdd), ""
		if old, ok := installed[component.identity()]; ok {
			symbol, change = cyan(sUp), "reinstall"
			if old.Version != "" && old.Version != tag {
				change = old.Version + " " + sArrow + " " + tag
			}
		}
		rows = append(rows, []string{symbol + " " + componentLabel(component), change, dim(componentTarget(component))})
	}
	removals := 0
	for _, old := range pkg.installedComponents() {
		if !wanted[old.identity()] {
			reason := "no longer selected"
			if dir, ok := movedTo[filepath.Base(old.Filename)]; ok && old.Filename != "" {
				reason = "moved to " + dir
			}
			rows = append(rows, []string{red(sRemove) + " " + red(componentName(old)) + componentKind(old), "", dim(componentTarget(old) + " " + sDot + " " + reason)})
			removals++
		}
	}
	msgSection("Plan", "")
	printTable("    ", nil, dropEmptyColumns(rows))
	if removals == 0 {
		return true, nil
	}
	fmt.Println()
	ok, err := askForConfirmation(fmt.Sprintf("Remove %s and apply the plan?", plural(removals, "component", "components")), false)
	if err == nil && !ok {
		msgResult(false, "Nothing changed %s", dim(sDot+" plan cancelled"))
	}
	return ok, err
}

func componentName(c InstalledComponent) string {
	if c.SystemPkgName != "" {
		return c.SystemPkgName
	}
	return filepath.Base(c.Filename)
}

func componentKind(c InstalledComponent) string {
	if c.SystemPkgName != "" {
		return " " + dim(c.SystemPkgType)
	}
	return ""
}

func componentLabel(c InstalledComponent) string {
	return green(componentName(c)) + componentKind(c)
}

// componentTarget says where a component goes and whether that needs sudo.
func componentTarget(c InstalledComponent) string {
	if c.SystemPkgName != "" {
		target := systemPackageTool(c.SystemPkgType)
		if os.Geteuid() != 0 {
			target += " " + sDot + " needs sudo"
		}
		return target
	}
	if !isWritableDir(existingDir(filepath.Dir(c.Filename))) {
		return c.Filename + " " + sDot + " needs sudo"
	}
	return c.Filename
}

func componentWhere(c InstalledComponent) string {
	if c.SystemPkgName != "" {
		return c.Version
	}
	return c.Filename
}

func countComponents(plan []plannedComponent) string {
	binaries, packages := 0, 0
	for _, item := range plan {
		if item.component.SystemPkgName != "" {
			packages++
		} else {
			binaries++
		}
	}
	var parts []string
	if binaries > 0 {
		parts = append(parts, plural(binaries, "binary", "binaries"))
	}
	if packages > 0 {
		parts = append(parts, plural(packages, "system package", "system packages"))
	}
	return strings.Join(parts, ", ")
}

func validateInstallPlan(plan []plannedComponent, pkg *Package, config *GrmConfig) error {
	destinations := make(map[string]string)
	systemFiles := make(map[string]string)
	for _, item := range plan {
		component := item.component
		identity := component.identity()
		if previous, ok := destinations[identity]; ok {
			return fmt.Errorf("installation conflict: %s and %s both install %s", previous, component.Asset, identity)
		}
		destinations[identity] = component.Asset
		for _, filename := range item.systemFiles {
			systemFiles[filepath.Clean(filename)] = component.SystemPkgName
		}
	}
	for _, item := range plan {
		component := item.component
		if component.Filename != "" {
			if name, ok := systemFiles[component.Filename]; ok {
				return fmt.Errorf("installation conflict: binary %s is also supplied by system package %s", component.Filename, name)
			}
		}
	}
	for name, installed := range config.Packages {
		for _, component := range installed.installedComponents() {
			if name != pkg.GetFullName() {
				if _, ok := destinations[component.identity()]; ok {
					return fmt.Errorf("installation conflict: %s is already tracked by %s", component.identity(), name)
				}
			}
			if component.Filename != "" {
				if systemName, ok := systemFiles[component.Filename]; ok {
					return fmt.Errorf("installation conflict: system package %s would overwrite binary %s tracked by %s", systemName, component.Filename, name)
				}
			}
		}
	}
	return nil
}

func applyInstallPlan(config *GrmConfig, pkg *Package, tag string, plan []plannedComponent, operations installOperations) error {
	pkg.Incomplete = true
	pkg.Components = append([]InstalledComponent(nil), pkg.installedComponents()...)
	pkg.Filename, pkg.MD5, pkg.SystemPkgName, pkg.SystemPkgType = "", "", "", ""
	if err := config.PutPackage(pkg); err != nil {
		return fmt.Errorf("record installation plan: %w", err)
	}
	wanted := make(map[string]bool)
	for _, item := range plan {
		component := item.component
		var err error
		if component.SystemPkgName != "" {
			err = operations.system(item.source)
		} else {
			err = operations.binary(item.source, component.Filename)
		}
		if err != nil {
			fmt.Printf("    %s %s\n", red(sCross), componentLabel(component))
			return fmt.Errorf("install %s from %s (completed components remain tracked): %w", component.identity(), component.Asset, err)
		}
		fmt.Printf("    %s %s  %s\n", green(sCheck), componentLabel(component), dim(componentWhere(component)))
		wanted[component.identity()] = true
		replaced := false
		for i := range pkg.Components {
			if pkg.Components[i].identity() == component.identity() {
				pkg.Components[i], replaced = component, true
				break
			}
		}
		if !replaced {
			pkg.Components = append(pkg.Components, component)
		}
		if err := config.PutPackage(pkg); err != nil {
			return fmt.Errorf("record installed component %s: %w", component.identity(), err)
		}
	}
	for i := 0; i < len(pkg.Components); {
		component := pkg.Components[i]
		if wanted[component.identity()] {
			i++
			continue
		}
		if err := operations.remove(component); err != nil {
			return fmt.Errorf("remove obsolete component %s: %w", component.identity(), err)
		}
		fmt.Printf("    %s %s  %s\n", red(sRemove), red(componentName(component))+componentKind(component), dim("removed"))
		pkg.Components = append(pkg.Components[:i], pkg.Components[i+1:]...)
		if err := config.PutPackage(pkg); err != nil {
			return err
		}
	}
	pkg.Version = tag
	pkg.Incomplete = false
	return config.PutPackage(pkg)
}

func installRelease(release *github.RepositoryRelease, pkg *Package, options installOptions) error {
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		return err
	}
	started := time.Now()
	previous := ""
	if len(pkg.installedComponents()) > 0 {
		previous = pkg.Version
	}
	header := pkgLabel(pkg.GetFullName()) + "  " + versionChange(previous, release.GetTagName())
	var details []string
	if !release.GetPublishedAt().IsZero() {
		details = append(details, "released "+release.GetPublishedAt().Format("2006-01-02"))
	}
	if previous == "" {
		details = append(details, "new install")
	}
	if len(details) > 0 {
		header += "  " + dim(strings.Join(details, " "+sDot+" "))
	}
	fmt.Println(header)
	legacy := len(pkg.Selections) == 0 && len(pkg.Components) == 0 && (pkg.Filename != "" || pkg.SystemPkgName != "") && len(options.assets) == 0 && len(options.binaries) == 0 && !options.reselect
	candidates := release.Assets
	var legacyPackage *Package
	if legacy {
		candidates = legacyAssetCandidates(candidates, pkg)
		legacyPackage = pkg
	}
	saved := generalizeSelections(pkg.Selections, pkg.Version)
	assets, selections, err := resolveAssetSelections(candidates, saved, options.assets, pkg.Filter, release.GetTagName())
	if err != nil {
		var hinted *hintedError
		if errors.As(err, &hinted) && hinted.fix == "" {
			hinted.fix = fmt.Sprintf("grm install %s --asset '<asset name or glob>'", pkg.GetFullName())
		}
		return err
	}
	fmt.Println()
	if len(options.assets) != 0 || len(saved) != 0 {
		reason := "saved selection"
		if len(options.assets) != 0 {
			reason = "--asset"
		}
		for i, asset := range assets {
			fmt.Printf("  %s   %s %s  %s\n", dim("Asset"), cyan(sSelect), asset.GetName(), dim(joinDetails(formatSize(asset.GetSize()), reason+" "+selections[i].Pattern)))
		}
	}
	var plan []plannedComponent
	binDir := config.PackageBinDir(pkg)
	for i, asset := range assets {
		filename, err := downloadFile(asset, pkg)
		if err != nil {
			return err
		}
		defer cleanupDownload(filename)
		items, remembered, err := prepareAsset(filename, asset.GetName(), release.GetTagName(), binDir, selections[i], options.binaries, legacyPackage)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", asset.GetName(), err)
		}
		if legacy && pkg.SystemPkgName != "" && items[0].component.SystemPkgName != pkg.SystemPkgName {
			return fmt.Errorf("asset %s contains system package %s, expected %s; use --asset to select again", asset.GetName(), items[0].component.SystemPkgName, pkg.SystemPkgName)
		}
		selections[i] = remembered
		plan = append(plan, items...)
	}
	for _, request := range options.binaries {
		if !request.matched {
			return fmt.Errorf("binary selector %q did not match any selected asset", request.pattern)
		}
	}
	rename := options.rename
	if rename != "" {
		if len(plan) != 1 || plan[0].component.SystemPkgName != "" {
			return fmt.Errorf("--rename requires exactly one binary; use --binary 'path=name' for individual renames")
		}
		if err := validateBinaryName(rename); err != nil {
			return err
		}
		plan[0].component.Filename = filepath.Join(binDir, rename)
		selections[0].Binaries[0].Rename = rename
	}
	if err := validateInstallPlan(plan, pkg, config); err != nil {
		return err
	}
	for _, item := range plan {
		if item.component.Filename != "" {
			if err := checkBinaryOwnership(item.component.Filename); err != nil {
				return err
			}
		}
	}
	confirmed, err := confirmInstallPlan(pkg, release.GetTagName(), plan)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	pkg.Selections = selections
	pkg.RenameBinaryTo = ""
	if err := applyInstallPlan(config, pkg, release.GetTagName(), plan, componentOperations()); err != nil {
		return err
	}
	if options.lock {
		pkg.Locked = true
		if err := config.PutPackage(pkg); err != nil {
			return err
		}
	}
	verb := "Installed"
	switch {
	case previous == release.GetTagName():
		verb = "Reinstalled"
	case previous != "":
		verb = "Updated"
	}
	fmt.Println()
	msgResult(true, "%s %s %s %s", verb, bold(pkg.Repo), boldCyan(release.GetTagName()), dim(sDot+" "+joinDetails(countComponents(plan), formatDuration(time.Since(started)))))
	return nil
}
