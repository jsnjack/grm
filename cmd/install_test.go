package cmd

import (
	"testing"
)

func TestInstall_filterList_empty(t *testing.T) {
	input := []string{"a", "b"}
	output := filterList(input, "", false)
	if len(input) != len(output) {
		t.Errorf("Expected nothing to be filtered")
		return
	}
}

func TestInstall_filterList_filter(t *testing.T) {
	input := []string{
		"hugo_0.80.0_Linux-64bit.deb",
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM64.deb",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterList(input, "extended", false)
	if len(output) != 2 {
		t.Errorf("Expected 2 values in output, got %d (%s)", len(output), output)
		return
	}
}

func TestInstall_filterSuitableAssets_empty_filter(t *testing.T) {
	input := []string{
		"hugo_0.80.0_checksums.txt",
		"hugo_0.80.0_DragonFlyBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-32bit.tar.gz",
		"hugo_0.80.0_FreeBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-ARM.tar.gz",
		"hugo_0.80.0_Linux-32bit.deb",
		"hugo_0.80.0_Linux-32bit.tar.gz",
		"hugo_0.80.0_Linux-64bit.deb",
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM.deb",
		"hugo_0.80.0_Linux-ARM.tar.gz",
		"hugo_0.80.0_Linux-ARM64.deb",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_0.80.0_macOS-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-32bit.tar.gz",
		"hugo_0.80.0_NetBSD-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-ARM.tar.gz",
		"hugo_0.80.0_OpenBSD-32bit.tar.gz",
		"hugo_0.80.0_OpenBSD-64bit.tar.gz",
		"hugo_0.80.0_OpenBSD-ARM.tar.gz",
		"hugo_0.80.0_Windows-32bit.zip",
		"hugo_0.80.0_Windows-64bit.zip",
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
		"hugo_extended_0.80.0_macOS-64bit.tar.gz",
		"hugo_extended_0.80.0_Windows-64bit.zip",
	}
	expected := []string{
		"hugo_0.80.0_Linux-64bit.deb",
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM64.deb",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterSuitableAssets(input, nil)
	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>")
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>", item)
		}
	}
}

func TestInstall_filterSuitableAssets_extended_filter(t *testing.T) {
	input := []string{
		"hugo_0.80.0_checksums.txt",
		"hugo_0.80.0_DragonFlyBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-32bit.tar.gz",
		"hugo_0.80.0_FreeBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-ARM.tar.gz",
		"hugo_0.80.0_Linux-32bit.deb",
		"hugo_0.80.0_Linux-32bit.tar.gz",
		"hugo_0.80.0_Linux-64bit.deb",
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM.deb",
		"hugo_0.80.0_Linux-ARM.tar.gz",
		"hugo_0.80.0_Linux-ARM64.deb",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_0.80.0_macOS-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-32bit.tar.gz",
		"hugo_0.80.0_NetBSD-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-ARM.tar.gz",
		"hugo_0.80.0_OpenBSD-32bit.tar.gz",
		"hugo_0.80.0_OpenBSD-64bit.tar.gz",
		"hugo_0.80.0_OpenBSD-ARM.tar.gz",
		"hugo_0.80.0_Windows-32bit.zip",
		"hugo_0.80.0_Windows-64bit.zip",
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
		"hugo_extended_0.80.0_macOS-64bit.tar.gz",
		"hugo_extended_0.80.0_Windows-64bit.zip",
	}
	expected := []string{
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterSuitableAssets(input, []string{"extended"})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>")
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>", item)
		}
	}
}

func TestInstall_filterSuitableAssets_extended_tar_filter(t *testing.T) {
	input := []string{
		"hugo_0.80.0_checksums.txt",
		"hugo_0.80.0_DragonFlyBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-32bit.tar.gz",
		"hugo_0.80.0_FreeBSD-64bit.tar.gz",
		"hugo_0.80.0_FreeBSD-ARM.tar.gz",
		"hugo_0.80.0_Linux-32bit.deb",
		"hugo_0.80.0_Linux-32bit.tar.gz",
		"hugo_0.80.0_Linux-64bit.deb",
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM.deb",
		"hugo_0.80.0_Linux-ARM.tar.gz",
		"hugo_0.80.0_Linux-ARM64.deb",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_0.80.0_macOS-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-32bit.tar.gz",
		"hugo_0.80.0_NetBSD-64bit.tar.gz",
		"hugo_0.80.0_NetBSD-ARM.tar.gz",
		"hugo_0.80.0_OpenBSD-32bit.tar.gz",
		"hugo_0.80.0_OpenBSD-64bit.tar.gz",
		"hugo_0.80.0_OpenBSD-ARM.tar.gz",
		"hugo_0.80.0_Windows-32bit.zip",
		"hugo_0.80.0_Windows-64bit.zip",
		"hugo_extended_0.80.0_Linux-64bit.deb",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
		"hugo_extended_0.80.0_macOS-64bit.tar.gz",
		"hugo_extended_0.80.0_Windows-64bit.zip",
	}
	expected := []string{
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterSuitableAssets(input, []string{"extended", "tar"})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>")
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>", item)
		}
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
