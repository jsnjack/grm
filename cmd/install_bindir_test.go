package cmd

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
)

// rawReleaseServer serves the latest release of a repository with one
// raw-binary asset and counts its downloads. The test can change the release
// between commands.
type rawReleaseServer struct {
	tag, asset, digest string // an empty digest leaves the field out
	payload            []byte
	downloads          int
}

// rawRelease starts a rawReleaseServer with a new empty config file.
func rawRelease(t *testing.T, tag, asset string, payload []byte, digest string) *rawReleaseServer {
	t.Helper()
	server := &rawReleaseServer{tag: tag, asset: asset, digest: digest, payload: payload}
	withGitHubTransport(t, func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, "/releases/assets/1") {
			server.downloads++
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(server.payload)), Request: request}, nil
		}
		field := ""
		if server.digest != "" {
			field = fmt.Sprintf(`,"digest":%q`, server.digest)
		}
		release := fmt.Sprintf(`{"tag_name":%q,"assets":[{"id":1,"name":%q%s}]}`, server.tag, server.asset, field)
		return httpResponse(request, http.StatusOK, []byte(release)), nil
	})
	return server
}

func newRelease(tag, asset, digest string) *github.RepositoryRelease {
	return &github.RepositoryRelease{TagName: tag, Assets: []*github.ReleaseAsset{{Name: &asset, Digest: &digest}}}
}

func sha256Digest(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

// withInstallFlag sets a string flag of the install command for one test.
func withInstallFlag(t *testing.T, name, value string) {
	t.Helper()
	flag := installCmd.Flags().Lookup(name)
	t.Cleanup(func() {
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Error(err)
		}
		flag.Changed = false
	})
	if err := installCmd.Flags().Set(name, value); err != nil {
		t.Fatal(err)
	}
}

func installedPackage(t *testing.T, name string) Package {
	t.Helper()
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	return config.Packages[name]
}

func componentFiles(p Package) []string {
	var files []string
	for _, c := range p.Components {
		files = append(files, c.Filename)
	}
	return files
}

func setBinDirSetting(t *testing.T, dir string) {
	t.Helper()
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.PutSetting("bin_dir", dir); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, filename string, want bool) {
	t.Helper()
	_, err := os.Stat(filename)
	if exists := err == nil; exists != want {
		t.Errorf("%s exists: %v, want %v (%v)", filename, exists, want, err)
	}
}

func TestInstallBinDirIsSavedPerPackage(t *testing.T) {
	withYes(t, true)
	server := rawRelease(t, "v1.0.0", "tool", fixtureBinary, "")
	global := t.TempDir()
	setBinDirSetting(t, global)
	custom := filepath.Join(t.TempDir(), "home", ".local", "bin")
	install := func(flags map[string]string) (string, error) {
		t.Helper()
		var output string
		var err error
		t.Run(fmt.Sprint(flags), func(t *testing.T) {
			for name, value := range flags {
				withInstallFlag(t, name, value)
			}
			output = captureStdout(t, func() { err = installCmd.RunE(installCmd, []string{"owner/tool"}) })
		})
		return output, err
	}

	if _, err := install(map[string]string{"bin-dir": custom}); err != nil {
		t.Fatal(err)
	}
	p := installedPackage(t, "owner/tool")
	if p.BinDir != custom || !reflect.DeepEqual(componentFiles(p), []string{filepath.Join(custom, "tool")}) || server.downloads != 1 {
		t.Fatalf("first install saved bin dir %q and files %v after %d downloads", p.BinDir, componentFiles(p), server.downloads)
	}
	assertExists(t, filepath.Join(custom, "tool"), true)
	assertExists(t, filepath.Join(global, "tool"), false)

	// Repeating the directory, with or without a trailing slash, or leaving
	// the flag out changes nothing
	for _, flags := range []map[string]string{{"bin-dir": custom}, {"bin-dir": custom + "/"}, {}} {
		if _, err := install(flags); err != nil {
			t.Fatal(err)
		}
	}
	if server.downloads != 1 {
		t.Errorf("repeated installs downloaded %d times, want 1", server.downloads)
	}

	// An empty value follows bin_dir again and moves the binary there
	output, err := install(map[string]string{"bin-dir": ""})
	if err != nil {
		t.Fatal(err)
	}
	p = installedPackage(t, "owner/tool")
	if p.BinDir != "" || !reflect.DeepEqual(componentFiles(p), []string{filepath.Join(global, "tool")}) || server.downloads != 2 {
		t.Fatalf("reset saved bin dir %q and files %v after %d downloads", p.BinDir, componentFiles(p), server.downloads)
	}
	assertExists(t, filepath.Join(custom, "tool"), false)
	assertExists(t, filepath.Join(global, "tool"), true)
	if !strings.Contains(output, "moved to "+global) {
		t.Errorf("plan does not say the binary moved to %s:\n%s", global, output)
	}

	// update keeps the saved directory
	if _, err := install(map[string]string{"bin-dir": custom}); err != nil {
		t.Fatal(err)
	}
	server.tag, server.downloads = "v2.0.0", 0
	captureStdout(t, func() { err = updateCmd.RunE(updateCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	p = installedPackage(t, "owner/tool")
	if p.Version != "v2.0.0" || !reflect.DeepEqual(componentFiles(p), []string{filepath.Join(custom, "tool")}) || server.downloads != 1 {
		t.Errorf("update installed %s into %v after %d downloads", p.Version, componentFiles(p), server.downloads)
	}
}

func TestInstallBinDirMustBeAbsolute(t *testing.T) {
	withInstallFlag(t, "bin-dir", "relative/bin")
	if err := installCmd.Args(installCmd, []string{"owner/tool"}); err == nil || !strings.Contains(err.Error(), "--bin-dir must be an absolute path") {
		t.Errorf("relative --bin-dir returned %v", err)
	}
}

func TestInstallRepeatedRenameChangesNothing(t *testing.T) {
	withYes(t, true)
	server := rawRelease(t, "v1.0.0", "tool", fixtureBinary, "")
	dir := t.TempDir()
	setBinDirSetting(t, dir)
	for _, c := range []struct {
		rename        string
		wantDownloads int
	}{{"renamed", 1}, {"renamed", 1}, {"other", 2}} {
		t.Run(c.rename, func(t *testing.T) {
			withInstallFlag(t, "rename", c.rename)
			if err := installCmd.RunE(installCmd, []string{"owner/tool"}); err != nil {
				t.Fatal(err)
			}
			if server.downloads != c.wantDownloads {
				t.Errorf("downloaded %d times, want %d", server.downloads, c.wantDownloads)
			}
			p := installedPackage(t, "owner/tool")
			if want := []string{filepath.Join(dir, c.rename)}; !reflect.DeepEqual(componentFiles(p), want) {
				t.Errorf("installed %v, want %v", componentFiles(p), want)
			}
		})
	}
	assertExists(t, filepath.Join(dir, "renamed"), false)
}

func TestBinaryCounts(t *testing.T) {
	got := binaryCounts(map[string]int{"/usr/local/bin": 3, "/home/client/.local/bin": 1, "/opt/bin": 1})
	want := []string{"3 binaries in /usr/local/bin", "1 binary in /home/client/.local/bin", "1 binary in /opt/bin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("binaryCounts() = %v, want %v", got, want)
	}
}

// selfUpdated records owner/tool at v1.0.0 in dir while the file on disk
// already holds newer, as wakeci leaves it after updating itself.
func selfUpdated(t *testing.T, dir string, newer []byte) Package {
	t.Helper()
	filename := filepath.Join(dir, "tool")
	if err := os.WriteFile(filename, newer, 0755); err != nil {
		t.Fatal(err)
	}
	p := Package{
		Owner: "owner", Repo: "tool", Version: "v1.0.0",
		Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "tool"}}}},
		Components: []InstalledComponent{{Asset: "tool", Source: "tool", Version: "v1.0.0", Filename: filename, MD5: fmt.Sprintf("%x", md5.Sum(fixtureBinary))}},
	}
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.PutPackage(&p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSelfUpdatedBinaryIsRecordedWithoutDownload(t *testing.T) {
	newer := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	commands := map[string]func() error{
		"install": func() error { return installCmd.RunE(installCmd, []string{"owner/tool"}) },
		"update":  func() error { return updateCmd.RunE(updateCmd, nil) },
	}
	cases := []struct {
		name, digest  string
		wantDownloads int
	}{
		{name: "matching digest", digest: sha256Digest(newer)},
		{name: "different digest", digest: sha256Digest(fixtureBinary), wantDownloads: 1},
		{name: "no digest", wantDownloads: 1},
	}
	for command, run := range commands {
		for _, c := range cases {
			t.Run(command+" "+c.name, func(t *testing.T) {
				withYes(t, true)
				server := rawRelease(t, "v2.0.0", "tool", newer, c.digest)
				dir := t.TempDir()
				setBinDirSetting(t, dir)
				selfUpdated(t, dir, newer)
				var err error
				output := captureStdout(t, func() { err = run() })
				if err != nil {
					t.Fatal(err)
				}
				p := installedPackage(t, "owner/tool")
				if server.downloads != c.wantDownloads || p.Version != "v2.0.0" || p.Components[0].Version != "v2.0.0" || p.Components[0].MD5 != fmt.Sprintf("%x", md5.Sum(newer)) {
					t.Errorf("after %d downloads recorded %+v, want %d downloads and v2.0.0", server.downloads, p, c.wantDownloads)
				}
				if err := p.VerifyVersion("v2.0.0"); err != nil {
					t.Error(err)
				}
				if c.wantDownloads == 0 && !strings.Contains(output, "installed binaries match the release") {
					t.Errorf("output does not explain the recorded update:\n%s", output)
				}
			})
		}
	}
}

func TestReleaseOnDisk(t *testing.T) {
	newer := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	dir := t.TempDir()
	for name, mode := range map[string]os.FileMode{"tool": 0755, "tool_v1.0.0_linux": 0755, "plain": 0644} {
		if err := os.WriteFile(filepath.Join(dir, name), newer, mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(dir, "tool"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		pkg     Package
		asset   string
		binDir  string
		want    bool
		wantRec InstalledComponent
	}{
		{
			name:  "renamed versioned asset",
			asset: "tool_v2.0.0_linux",
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool_*_linux", Binaries: []BinarySelection{{Path: "tool_*_linux", Rename: "tool"}}}},
				Components: []InstalledComponent{{Asset: "tool_v1.0.0_linux", Source: "tool_v1.0.0_linux", Version: "v1.0.0", Filename: filepath.Join(dir, "tool")}}},
			want:    true,
			wantRec: InstalledComponent{Asset: "tool_v2.0.0_linux", Source: "tool_v2.0.0_linux", Version: "v2.0.0", Filename: filepath.Join(dir, "tool")},
		},
		{
			name:  "versioned name would change",
			asset: "tool_v2.0.0_linux",
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool_*_linux", Binaries: []BinarySelection{{Path: "tool_*_linux"}}}},
				Components: []InstalledComponent{{Asset: "tool_v1.0.0_linux", Source: "tool_v1.0.0_linux", Version: "v1.0.0", Filename: filepath.Join(dir, "tool_v1.0.0_linux")}}},
		},
		{
			name:  "archive member",
			asset: "tool",
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "bin/tool"}}}},
				Components: []InstalledComponent{{Asset: "tool", Source: "bin/tool", Version: "v1.0.0", Filename: filepath.Join(dir, "tool")}}},
		},
		{
			name:   "pending move to another directory",
			asset:  "tool",
			binDir: t.TempDir(),
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "tool"}}}},
				Components: []InstalledComponent{{Asset: "tool", Source: "tool", Version: "v1.0.0", Filename: filepath.Join(dir, "tool")}}},
		},
		{
			name:  "not executable",
			asset: "tool",
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "tool", Rename: "plain"}}}},
				Components: []InstalledComponent{{Asset: "tool", Source: "tool", Version: "v1.0.0", Filename: filepath.Join(dir, "plain")}}},
		},
		{
			name:  "symlink",
			asset: "tool",
			pkg: Package{Version: "v1.0.0",
				Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "tool", Rename: "link"}}}},
				Components: []InstalledComponent{{Asset: "tool", Source: "tool", Version: "v1.0.0", Filename: filepath.Join(dir, "link")}}},
		},
		{
			name:  "incomplete",
			asset: "tool",
			pkg: Package{Version: "v1.0.0", Incomplete: true,
				Selections: []AssetSelection{{Pattern: "tool", Binaries: []BinarySelection{{Path: "tool"}}}},
				Components: []InstalledComponent{{Asset: "tool", Source: "tool", Version: "v1.0.0", Filename: filepath.Join(dir, "tool")}}},
		},
		{
			name:  "legacy record",
			asset: "tool",
			pkg:   Package{Version: "v1.0.0", Filename: filepath.Join(dir, "tool")},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			binDir := dir
			if c.binDir != "" {
				binDir = c.binDir
			}
			got, ok := releaseOnDisk(c.pkg, newRelease("v2.0.0", c.asset, sha256Digest(newer)), binDir)
			if ok != c.want {
				t.Fatalf("releaseOnDisk() = %v, want %v", ok, c.want)
			}
			if ok && (got.Version != "v2.0.0" || !reflect.DeepEqual(got.Components, []InstalledComponent{c.wantRec})) {
				t.Errorf("recorded %s %+v, want v2.0.0 %+v", got.Version, got.Components, c.wantRec)
			}
		})
	}
}
