package cmd

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func installSystemPackage(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	var program string
	var args []string
	switch ext {
	case ".deb":
		if _, err := exec.LookPath("apt"); err == nil {
			program, args = "apt", []string{"install", "-y", filename}
		} else {
			program, args = "dpkg", []string{"-i", filename}
		}
	case ".rpm":
		if _, err := exec.LookPath("dnf"); err == nil {
			program, args = "dnf", []string{"install", "-y", filename}
		} else {
			program, args = "rpm", []string{"-U", "--replacepkgs", filename}
		}
	default:
		return fmt.Errorf("unsupported system package extension: %s", ext)
	}

	msgStep("Installing system package %s", bold(filepath.Base(filename)))
	if err := runPrivileged(program, args...); err != nil {
		return fmt.Errorf("install %s: %w", filename, err)
	}
	return nil
}

func removeSystemPackage(pkg InstalledComponent) error {
	var program string
	var args []string
	switch pkg.SystemPkgType {
	case "deb":
		if _, err := exec.LookPath("apt"); err == nil {
			program, args = "apt", []string{"remove", "-y", pkg.SystemPkgName}
		} else {
			program, args = "dpkg", []string{"-r", pkg.SystemPkgName}
		}
	case "rpm":
		if _, err := exec.LookPath("dnf"); err == nil {
			program, args = "dnf", []string{"remove", "-y", pkg.SystemPkgName}
		} else {
			program, args = "rpm", []string{"-e", pkg.SystemPkgName}
		}
	default:
		return fmt.Errorf("unknown system package type: %q", pkg.SystemPkgType)
	}

	if err := runPrivileged(program, args...); err != nil {
		return fmt.Errorf("remove system package %s: %w", pkg.SystemPkgName, err)
	}
	return nil
}

func checkSystemPackageSupport(ext string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("system packages require Linux")
	}
	program := "rpm"
	if ext == ".deb" {
		program = "dpkg"
	}
	if _, err := exec.LookPath(program); err != nil {
		return fmt.Errorf("cannot install %s: %s is unavailable", ext, program)
	}
	return nil
}

func checkBinaryOwnership(filename string) error {
	if _, err := os.Lstat(filename); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect destination %s: %w", filename, err)
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		if out, err := exec.Command("rpm", "-qf", "--queryformat", "%{NAME}", filename).Output(); err == nil {
			return fmt.Errorf("installation conflict: %s is owned by RPM %s", filename, strings.TrimSpace(string(out)))
		}
	}
	if _, err := exec.LookPath("dpkg-query"); err == nil {
		if out, err := exec.Command("dpkg-query", "-S", filename).Output(); err == nil {
			return fmt.Errorf("installation conflict: %s is owned by a system package: %s", filename, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func getSystemPackageFiles(filename string) ([]string, error) {
	if strings.EqualFold(filepath.Ext(filename), ".rpm") {
		out, err := exec.Command("rpm", "-qpl", filename).Output()
		if err != nil {
			return nil, fmt.Errorf("list files in %s: %w", filename, err)
		}
		return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
	}
	cmd := exec.Command("dpkg-deb", "--fsys-tarfile", filename)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open package listing: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("read package contents: %w", err)
	}
	var files []string
	reader := tar.NewReader(pipe)
	var readErr error
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			readErr = err
			break
		}
		files = append(files, filepath.Join("/", header.Name))
	}
	_, drainErr := io.Copy(io.Discard, pipe)
	waitErr := cmd.Wait()
	if readErr != nil || drainErr != nil || waitErr != nil {
		return nil, fmt.Errorf("list files in %s: %w", filename, errors.Join(readErr, drainErr, waitErr))
	}
	return files, nil
}

// getSystemPackageName extracts the package name from a .deb or .rpm file
func getSystemPackageName(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".deb":
		out, err := exec.Command("dpkg-deb", "--field", filename, "Package").Output()
		if err != nil {
			return "", fmt.Errorf("read package name from %s: %w", filename, err)
		}
		return strings.TrimSpace(string(out)), nil
	case ".rpm":
		out, err := exec.Command("rpm", "-qp", "--queryformat", "%{NAME}", filename).Output()
		if err != nil {
			return "", fmt.Errorf("read package name from %s: %w", filename, err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("unknown package type: %s", ext)
}
