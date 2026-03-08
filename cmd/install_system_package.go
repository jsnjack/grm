package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installSystemPackage(filename string, pkg *Package) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	pkgName, err := getSystemPackageName(filename)
	if err != nil {
		return "", fmt.Errorf("could not determine package name: %w", err)
	}
	pkg.SystemPkgName = pkgName
	pkg.SystemPkgType = strings.TrimPrefix(ext, ".")

	var installCmd string
	switch ext {
	case ".deb":
		if _, err := exec.LookPath("apt"); err == nil {
			installCmd = fmt.Sprintf("%sapt install -y %s", sudo(), filename)
		} else {
			installCmd = fmt.Sprintf("%sdpkg -i %s", sudo(), filename)
		}
	case ".rpm":
		if _, err := exec.LookPath("dnf"); err == nil {
			installCmd = fmt.Sprintf("%sdnf install -y %s", sudo(), filename)
		} else {
			installCmd = fmt.Sprintf("%srpm -i %s", sudo(), filename)
		}
	default:
		return "", fmt.Errorf("unsupported system package extension: %s", ext)
	}

	fmt.Printf("Installing system package %s...\n", filepath.Base(filename))
	cmd := exec.Command("/bin/sh", "-c", installCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Clean up tmp dir
	tmpDir := getTmpDir(filename)
	if strings.HasPrefix(tmpDir, DefaultTmpDirPattern) {
		logf("Removing %s...\n", tmpDir)
		exec.Command("/bin/sh", "-c", fmt.Sprintf("rm -rf %s", tmpDir)).Run()
	}

	// No file to track — version tag + SystemPkgName are sufficient
	return "", nil
}

func removeSystemPackage(pkg *Package) error {
	var removeCmd string
	switch pkg.SystemPkgType {
	case "deb":
		if _, err := exec.LookPath("apt"); err == nil {
			removeCmd = fmt.Sprintf("%sapt remove -y %s", sudo(), pkg.SystemPkgName)
		} else {
			removeCmd = fmt.Sprintf("%sdpkg -r %s", sudo(), pkg.SystemPkgName)
		}
	case "rpm":
		if _, err := exec.LookPath("dnf"); err == nil {
			removeCmd = fmt.Sprintf("%sdnf remove -y %s", sudo(), pkg.SystemPkgName)
		} else {
			removeCmd = fmt.Sprintf("%srpm -e %s", sudo(), pkg.SystemPkgName)
		}
	default:
		return fmt.Errorf("unknown system package type: %q", pkg.SystemPkgType)
	}

	cmd := exec.Command("/bin/sh", "-c", removeCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getSystemPackageName extracts the package name from a .deb or .rpm file
func getSystemPackageName(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".deb":
		out, err := exec.Command("dpkg-deb", "--field", filename, "Package").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case ".rpm":
		out, err := exec.Command("rpm", "-qp", "--queryformat", "%{NAME}", filename).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("unknown package type: %s", ext)
}
