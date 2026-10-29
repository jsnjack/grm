package cmd

import (
	"fmt"
	"path/filepath"

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
	msgStep("Installation plan for %s %s", pkg.GetFullName(), tag)
	wanted := make(map[string]bool)
	for _, item := range plan {
		fmt.Printf("    %s -> %s\n", item.component.Asset, item.component.identity())
		wanted[item.component.identity()] = true
	}
	removes := false
	for _, old := range pkg.installedComponents() {
		if !wanted[old.identity()] {
			fmt.Printf("    remove %s\n", old.identity())
			removes = true
		}
	}
	if !removes {
		return true, nil
	}
	return askForConfirmation("Apply installation plan?")
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
			return fmt.Errorf("install %s from %s (completed components remain tracked): %w", component.identity(), component.Asset, err)
		}
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
	msgStep("Inspecting assets...")
	legacy := len(pkg.Selections) == 0 && len(pkg.Components) == 0 && (pkg.Filename != "" || pkg.SystemPkgName != "") && len(options.assets) == 0 && len(options.binaries) == 0 && !options.reselect
	candidates := release.Assets
	var legacyPackage *Package
	if legacy {
		candidates = legacyAssetCandidates(candidates, pkg)
		legacyPackage = pkg
	}
	assets, selections, err := resolveAssetSelections(candidates, generalizeSelections(pkg.Selections, pkg.Version), options.assets, pkg.Filter, release.GetTagName())
	if err != nil {
		return err
	}
	var plan []plannedComponent
	binDir := config.BinDir()
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
	msgDone()
	return nil
}
