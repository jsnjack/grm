package cmd

import (
	"os"
	"testing"
)

func TestUtils_CreatePackage_empty(t *testing.T) {
	_, err := CreatePackage("")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "invalid package: expected <owner>/<repo>[==<version>|~=<filter>], got " {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_oneEl(t *testing.T) {
	_, err := CreatePackage("jsnjack")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "invalid package: expected <owner>/<repo>[==<version>|~=<filter>], got jsnjack" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_oneSlash(t *testing.T) {
	_, err := CreatePackage("/")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "got empty <owner> from /" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_onlyOwner(t *testing.T) {
	_, err := CreatePackage("jsnjack/")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "got empty <repo> from jsnjack/" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_ok(t *testing.T) {
	p, err := CreatePackage("jsnjack/kazy-go")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "kazy-go" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
	if p.Version != "" {
		t.Errorf("Expected empty string, got %s", p.Version)
	}
}

func TestUtils_CreatePackage_okVersion(t *testing.T) {
	p, err := CreatePackage("jsnjack/kazy-go==v1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "kazy-go" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
	if p.Version != "v1" {
		t.Errorf("Expected v1, got %s", p.Version)
	}
}

func TestUtils_CreatePackage_okVersion2(t *testing.T) {
	p, err := CreatePackage("jsnjack/kazy-go==v1==")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "kazy-go" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
	if p.Version != "v1==" {
		t.Errorf("Expected v1==, got %s", p.Version)
	}
}

func TestUtils_CreatePackage_alias(t *testing.T) {
	p, err := CreatePackage("grm")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "grm" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
}

func TestPackage_IsSystemPackage_false_when_no_name(t *testing.T) {
	p := Package{Filename: "/usr/local/bin/tool"}
	if p.IsSystemPackage() {
		t.Errorf("Expected false for package without SystemPkgName")
	}
}

func TestPackage_IsSystemPackage_true_when_name_set(t *testing.T) {
	p := Package{SystemPkgName: "github-desktop-plus", SystemPkgType: "rpm"}
	if !p.IsSystemPackage() {
		t.Errorf("Expected true for package with SystemPkgName set")
	}
}

func TestPackage_VerifyVersion_mismatch(t *testing.T) {
	p := Package{Version: "v1.0.0"}
	err := p.VerifyVersion("v2.0.0")
	if err == nil {
		t.Errorf("Expected error on version mismatch, got nil")
	}
}

func TestPackage_VerifyVersion_system_package_match(t *testing.T) {
	// System packages have no Filename — version tag is the only check
	p := Package{Version: "v1.0.0", SystemPkgName: "mypkg", SystemPkgType: "rpm"}
	err := p.VerifyVersion("v1.0.0")
	if err != nil {
		t.Errorf("Expected nil for matching version with no file, got: %s", err)
	}
}

func TestPackage_VerifyVersion_system_package_mismatch(t *testing.T) {
	p := Package{Version: "v1.0.0", SystemPkgName: "mypkg", SystemPkgType: "rpm"}
	err := p.VerifyVersion("v2.0.0")
	if err == nil {
		t.Errorf("Expected error on version mismatch for system package, got nil")
	}
}

func TestPackage_VerifyVersion_binary_match(t *testing.T) {
	// Create a temp file to simulate an installed binary
	f, err := os.CreateTemp("", "grm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("binary content")
	f.Close()

	hash, _ := tomd5(f.Name())
	p := Package{Version: "v1.0.0", Filename: f.Name(), MD5: hash}
	if err := p.VerifyVersion("v1.0.0"); err != nil {
		t.Errorf("Expected nil for matching version and hash, got: %s", err)
	}
}

func TestPackage_VerifyVersion_binary_hash_mismatch(t *testing.T) {
	f, err := os.CreateTemp("", "grm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("binary content")
	f.Close()

	p := Package{Version: "v1.0.0", Filename: f.Name(), MD5: "wronghash"}
	if err := p.VerifyVersion("v1.0.0"); err == nil {
		t.Errorf("Expected error on hash mismatch, got nil")
	}
}

func TestUtils_CreatePackage_okVersionFilter(t *testing.T) {
	p, err := CreatePackage("jsnjack/kazy-go~=v146")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "kazy-go" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
	if p.VersionFilter != "v146" {
		t.Errorf("Expected v146, got %s", p.VersionFilter)
	}
	if p.Version != "" {
		t.Errorf("Expected empty Version, got %s", p.Version)
	}
}

func TestUtils_CreatePackage_okVersionFilterGlob(t *testing.T) {
	p, err := CreatePackage("jsnjack/kazy-go~=v146*")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.VersionFilter != "v146*" {
		t.Errorf("Expected v146*, got %s", p.VersionFilter)
	}
}

func TestUtils_CreatePackage_aliasWithVersionFilter(t *testing.T) {
	p, err := CreatePackage("grm~=v0.5")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "grm" {
		t.Errorf("Expected grm, got %s", p.Repo)
	}
	if p.VersionFilter != "v0.5" {
		t.Errorf("Expected v0.5, got %s", p.VersionFilter)
	}
}

func TestUtils_CreatePackage_conflictOperators(t *testing.T) {
	_, err := CreatePackage("jsnjack/kazy-go==v1~=v2")
	if err == nil {
		t.Errorf("Expected error for mixed operators, got nil")
		return
	}
	if err.Error() != "cannot use both == and ~= operators" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_alias_with_version(t *testing.T) {
	p, err := CreatePackage("grm==v0.50")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if p.Owner != "jsnjack" {
		t.Errorf("Expected jsnjack, got %s", p.Owner)
	}
	if p.Repo != "grm" {
		t.Errorf("Expected kazy-go, got %s", p.Repo)
	}
	if p.Version != "v0.50" {
		t.Errorf("Expected v0.50, got %s", p.Version)
	}
}
