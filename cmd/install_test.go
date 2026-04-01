package cmd

import (
	"os/exec"
	"testing"
)

func TestMatchVersionFilter_prefix(t *testing.T) {
	cases := []struct {
		filter  string
		tag     string
		want    bool
	}{
		{"v146", "v146.0.7680.72", true},
		{"v146", "v146", true},
		{"v146", "v145.9.0", false},
		{"v146", "v1460.0", true}, // prefix match — v1460 starts with v146
		{"v146.", "v146.0.1", true},
		{"v146.", "v1460.0", false},
	}
	for _, c := range cases {
		got, err := matchVersionFilter(c.filter, c.tag)
		if err != nil {
			t.Errorf("filter=%q tag=%q: unexpected error: %s", c.filter, c.tag, err)
			continue
		}
		if got != c.want {
			t.Errorf("filter=%q tag=%q: got %v, want %v", c.filter, c.tag, got, c.want)
		}
	}
}

func TestMatchVersionFilter_glob(t *testing.T) {
	cases := []struct {
		filter  string
		tag     string
		want    bool
		wantErr bool
	}{
		{"v146*", "v146.0.7680.72", true, false},
		{"v146*", "v145.0", false, false},
		{"v[0-9]*-stable", "v12-stable", true, false},
		{"v[0-9]*-stable", "v12-beta", false, false},
		{"*-chromium*", "v100-chromium-linux", true, false},
		{"*-chromium*", "v100-firefox-linux", false, false},
		{"v1[45][0-9]", "v146", true, false},
		{"v1[45][0-9]", "v160", false, false},
		{"[invalid", "v1", false, true}, // malformed glob
	}
	for _, c := range cases {
		got, err := matchVersionFilter(c.filter, c.tag)
		if c.wantErr {
			if err == nil {
				t.Errorf("filter=%q tag=%q: expected error, got nil", c.filter, c.tag)
			}
			continue
		}
		if err != nil {
			t.Errorf("filter=%q tag=%q: unexpected error: %s", c.filter, c.tag, err)
			continue
		}
		if got != c.want {
			t.Errorf("filter=%q tag=%q: got %v, want %v", c.filter, c.tag, got, c.want)
		}
	}
}

func TestInstall_filterList_empty(t *testing.T) {
	input := []string{"a", "b"}
	output := preferToContain(input, "")
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
	output := preferToContain(input, "extended")
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
		"hugo_0.80.0_Linux-64bit.tar.gz",
		"hugo_0.80.0_Linux-ARM64.tar.gz",
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterSuitableAssets(input, nil)
	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
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
		"hugo_extended_0.80.0_Linux-64bit.tar.gz",
	}
	output := filterSuitableAssets(input, []string{"extended"})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
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
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>", item)
		}
	}
}

func TestInstall_filterSuitableAssets_no_arm(t *testing.T) {
	input := []string{
		"checksums.txt",
		"go-mod-upgrade_0.9.0_Darwin_arm64.tar.gz",
		"go-mod-upgrade_0.9.0_Darwin_x86_64.tar.gz",
		"go-mod-upgrade_0.9.0_Linux_arm64.tar.gz",
		"go-mod-upgrade_0.9.0_Linux_i386.tar.gz",
		"go-mod-upgrade_0.9.0_Linux_x86_64.tar.gz",
		"go-mod-upgrade_0.9.0_Windows_arm64.tar.gz",
		"go-mod-upgrade_0.9.0_Windows_i386.tar.gz",
		"go-mod-upgrade_0.9.0_Windows_x86_64.tar.gz",
	}
	expected := []string{
		"go-mod-upgrade_0.9.0_Linux_x86_64.tar.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>", item)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_out_system_packages(t *testing.T) {
	input := []string{
		"k6-v0.41.0-linux-amd64.deb",
		"k6-v0.41.0-linux-amd64.rpm",
		"k6-v0.41.0-linux-amd64.tar.gz",
	}
	expected := []string{"k6-v0.41.0-linux-amd64.tar.gz"}
	if _, err := exec.LookPath("rpm"); err == nil {
		expected = append(expected, "k6-v0.41.0-linux-amd64.rpm")
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		expected = append(expected, "k6-v0.41.0-linux-amd64.deb")
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_out_asc(t *testing.T) {
	input := []string{
		"geckodriver-v0.32.0-linux64.tar.gz",
		"geckodriver-v0.32.0-linux64.tar.gz.asc",
	}
	expected := []string{
		"geckodriver-v0.32.0-linux64.tar.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_out_sha256sum(t *testing.T) {
	input := []string{
		"zellij-x86_64-apple-darwin.sha256sum",
		"zellij-x86_64-apple-darwin.tar.gz",
	}
	expected := []string{
		"zellij-x86_64-apple-darwin.tar.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_out_md5(t *testing.T) {
	input := []string{
		"cnotes-sftp-client-0.0.4-linux-amd64.tar.gz",
		"cnotes-sftp-client-0.0.4-linux-amd64.tar.gz.md5",
	}
	expected := []string{
		"cnotes-sftp-client-0.0.4-linux-amd64.tar.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_out_sha(t *testing.T) {
	input := []string{
		"micro-2.0.14-linux64-static.tar.gz",
		"micro-2.0.14-linux64-static.tar.gz.sha",
	}
	expected := []string{
		"micro-2.0.14-linux64-static.tar.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_filterSuitableAssets_filter_x86(t *testing.T) {
	input := []string{
		"tree-sitter-linux-arm64.gz",
		"tree-sitter-linux-powerpc64.gz",
		"tree-sitter-linux-x64.gz",
	}
	expected := []string{
		"tree-sitter-linux-x64.gz",
	}
	output := filterSuitableAssets(input, []string{})

	if len(output) != len(expected) {
		t.Errorf("Unexpected amount of items in <output>: got %d want %d", len(output), len(expected))
		return
	}

	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("Expected %s to be in <output>, got %s", item, output)
		}
	}
}

func TestInstall_hardExcludeExtension_removes_match(t *testing.T) {
	input := []string{"file.AppImage", "file.tar.gz"}
	output := hardExcludeExtension(input, ".AppImage")
	if len(output) != 1 || output[0] != "file.tar.gz" {
		t.Errorf("Expected only file.tar.gz, got %v", output)
	}
}

func TestInstall_hardExcludeExtension_empty_result(t *testing.T) {
	input := []string{"a.AppImage", "b.AppImage"}
	output := hardExcludeExtension(input, ".AppImage")
	if len(output) != 0 {
		t.Errorf("Expected empty result, got %v", output)
	}
}

func TestInstall_hardExcludeExtension_case_insensitive(t *testing.T) {
	input := []string{"file.APPIMAGE", "file.tar.gz"}
	output := hardExcludeExtension(input, ".AppImage")
	if len(output) != 1 || output[0] != "file.tar.gz" {
		t.Errorf("Expected only file.tar.gz, got %v", output)
	}
}

func TestInstall_filterSuitableAssets_appimage_always_excluded(t *testing.T) {
	// AppImage should be excluded even when it's the only option
	input := []string{
		"app-x86_64.AppImage",
		"app-x86_64.AppImage.zsync",
	}
	output := filterSuitableAssets(input, []string{})
	if len(output) != 0 {
		t.Errorf("Expected AppImage to always be excluded, got %v", output)
	}
}

func TestInstall_filterSuitableAssets_appimage_excluded_with_alternatives(t *testing.T) {
	input := []string{
		"app-linux-x86_64.AppImage",
		"app-linux-x86_64.AppImage.zsync",
		"app-linux-x86_64.tar.gz",
	}
	expected := "app-linux-x86_64.tar.gz"
	output := filterSuitableAssets(input, []string{})
	if len(output) != 1 || output[0] != expected {
		t.Errorf("Expected only %s, got %v", expected, output)
	}
}

func TestInstall_filterSuitableAssets_zsync_always_excluded(t *testing.T) {
	input := []string{
		"app-linux-x86_64.tar.gz",
		"app-linux-x86_64.tar.gz.zsync",
	}
	expected := "app-linux-x86_64.tar.gz"
	output := filterSuitableAssets(input, []string{})
	if len(output) != 1 || output[0] != expected {
		t.Errorf("Expected only %s, got %v", expected, output)
	}
}

// TestInstall_filterSuitableAssets_darwin_arm64_aarch64 checks that assets
// labelled with the "aarch64" alias are preferred on darwin/arm64.
func TestInstall_filterSuitableAssets_darwin_arm64_aarch64(t *testing.T) {
	// Typical release layout used by tools like bat, fd, ripgrep
	input := []string{
		"tool-aarch64-apple-darwin.tar.gz",
		"tool-x86_64-apple-darwin.tar.gz",
		"tool-x86_64-unknown-linux-gnu.tar.gz",
		"checksums.txt",
	}
	expected := []string{"tool-aarch64-apple-darwin.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "darwin", "arm64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_darwin_arm64_explicit checks assets
// labelled "darwin-arm64" (without the aarch64 alias).
func TestInstall_filterSuitableAssets_darwin_arm64_explicit(t *testing.T) {
	input := []string{
		"tool-darwin-arm64.tar.gz",
		"tool-darwin-amd64.tar.gz",
		"tool-linux-amd64.tar.gz",
	}
	expected := []string{"tool-darwin-arm64.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "darwin", "arm64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_darwin_arm64_universal checks that a
// macOS universal binary is preferred on darwin/arm64 when no arm64/aarch64
// specific asset is available.
func TestInstall_filterSuitableAssets_darwin_arm64_universal(t *testing.T) {
	input := []string{
		"tool-darwin-amd64.tar.gz",
		"tool-darwin-universal.tar.gz",
		"tool-linux-amd64.tar.gz",
	}
	expected := []string{"tool-darwin-universal.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "darwin", "arm64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_darwin_arm64_prefers_native_over_universal
// checks that a native arm64 asset is selected over a universal binary when both exist.
func TestInstall_filterSuitableAssets_darwin_arm64_prefers_native_over_universal(t *testing.T) {
	input := []string{
		"tool-darwin-arm64.tar.gz",
		"tool-darwin-universal.tar.gz",
		"tool-darwin-amd64.tar.gz",
		"tool-linux-amd64.tar.gz",
	}
	expected := []string{"tool-darwin-arm64.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "darwin", "arm64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_darwin_amd64 checks that Intel darwin assets
// are still selected correctly after the refactor.
func TestInstall_filterSuitableAssets_darwin_amd64(t *testing.T) {
	input := []string{
		"tool-aarch64-apple-darwin.tar.gz",
		"tool-x86_64-apple-darwin.tar.gz",
		"tool-x86_64-unknown-linux-gnu.tar.gz",
	}
	expected := []string{"tool-x86_64-apple-darwin.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "darwin", "amd64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_linux_arm64_aarch64 checks aarch64 alias on Linux.
func TestInstall_filterSuitableAssets_linux_arm64_aarch64(t *testing.T) {
	input := []string{
		"tool-linux-aarch64.tar.gz",
		"tool-linux-x86_64.tar.gz",
		"tool-darwin-arm64.tar.gz",
	}
	expected := []string{"tool-linux-aarch64.tar.gz"}
	output := filterSuitableAssetsForPlatform(input, nil, "linux", "arm64")
	if len(output) != len(expected) {
		t.Errorf("got %v, want %v", output, expected)
		return
	}
	for _, item := range expected {
		if !stringInSlice(item, output) {
			t.Errorf("expected %s to be in output %v", item, output)
		}
	}
}

// TestInstall_filterSuitableAssets_linux_amd64_rpm_vs_deb checks that on a
// linux/amd64 system the x86_64.rpm is not shadowed by the amd64.deb when
// both naming conventions are present.
func TestInstall_filterSuitableAssets_linux_amd64_rpm_vs_deb(t *testing.T) {
	input := []string{
		"CHECKSUMS.sha256.asc",
		"RELEASES",
		"youtube-music-desktop-app-2.0.11-1.arm64.rpm",
		"youtube-music-desktop-app-2.0.11-1.x86_64.rpm",
		"youtube-music-desktop-app_2.0.11_amd64.deb",
		"youtube-music-desktop-app_2.0.11_arm64.deb",
		"YouTube.Music.Desktop.App-2.0.11.Setup.exe",
		"YouTube.Music.Desktop.App-darwin-arm64-2.0.11.zip",
		"YouTube.Music.Desktop.App-darwin-x64-2.0.11.zip",
		"youtube_music_desktop_app-2.0.11-full.nupkg",
	}
	output := filterSuitableAssetsForPlatform(input, nil, "linux", "amd64")
	// The x86_64 rpm must always survive arch filtering
	if !stringInSlice("youtube-music-desktop-app-2.0.11-1.x86_64.rpm", output) {
		t.Errorf("expected x86_64.rpm in output %v", output)
	}
	// The amd64 deb should survive only if dpkg is available
	if _, err := exec.LookPath("dpkg"); err == nil {
		if !stringInSlice("youtube-music-desktop-app_2.0.11_amd64.deb", output) {
			t.Errorf("dpkg available: expected amd64.deb in output %v", output)
		}
	} else {
		if stringInSlice("youtube-music-desktop-app_2.0.11_amd64.deb", output) {
			t.Errorf("dpkg not available: did not expect amd64.deb in output %v", output)
		}
	}
	// arm64 variants should be excluded
	for _, reject := range []string{
		"youtube-music-desktop-app-2.0.11-1.arm64.rpm",
		"youtube-music-desktop-app_2.0.11_arm64.deb",
	} {
		if stringInSlice(reject, output) {
			t.Errorf("did not expect %s in output %v", reject, output)
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
