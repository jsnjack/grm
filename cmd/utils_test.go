package cmd

import "testing"

func TestUtils_CreatePackage_empty(t *testing.T) {
	_, err := CreatePackage("")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "Invalid package: expected <owner>/<repo>==<version>, got " {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_oneEl(t *testing.T) {
	_, err := CreatePackage("jsnjack")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "Invalid package: expected <owner>/<repo>==<version>, got jsnjack" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_oneSlash(t *testing.T) {
	_, err := CreatePackage("/")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "Got empty <owner> from /" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestUtils_CreatePackage_onlyOwner(t *testing.T) {
	_, err := CreatePackage("jsnjack/")
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if err.Error() != "Got empty <repo> from jsnjack/" {
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
