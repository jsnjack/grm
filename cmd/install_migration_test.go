package cmd

import (
	"archive/tar"
	"bytes"
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

func TestInstallReleaseMigratesLegacyConfig(t *testing.T) {
	cases := []struct {
		name, filename, rename, packageType string
		raw, explicit                       bool
		missingType, changedPackage         bool
		missingBinary                       bool
		renameTo                            string
	}{
		{name: "archive keeps one binary", filename: "ya"},
		{name: "renamed archive keeps installed name", filename: "my-ya", rename: "my-ya"},
		{name: "self update keeps raw binary", filename: "grm", raw: true},
		{name: "explicit rename keeps original binary", filename: "ya", renameTo: "my-ya"},
		{name: "DEB stays DEB", packageType: "deb"},
		{name: "RPM stays RPM", packageType: "rpm"},
		{name: "explicit selection expands renamed legacy install", filename: "my-ya", rename: "my-ya", explicit: true},
		{name: "missing binary leaves old install intact", filename: "ya", missingBinary: true},
		{name: "missing DEB leaves old install intact", packageType: "deb", missingType: true},
		{name: "different RPM leaves old install intact", packageType: "rpm", changedPackage: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withYes(t, true)
			oldConfig, oldToken, oldTransport := ConfigFile, rootToken, http.DefaultTransport
			t.Cleanup(func() {
				ConfigFile, rootToken, http.DefaultTransport = oldConfig, oldToken, oldTransport
			})
			rootToken = ""
			t.Setenv("GITHUB_TOKEN", "")
			dir := t.TempDir()
			ConfigFile = filepath.Join(dir, "grm.yaml")
			filename, packageName := "", ""
			if tc.filename != "" {
				filename = filepath.Join(DefaultBinDir, tc.filename)
			}
			if tc.packageType != "" {
				packageName = "tools-gui"
			}
			legacy := fmt.Sprintf(`packages:
  owner/tools:
    owner: owner
    repo: tools
    version: v1.0.0
    filename: %q
    md5: old-hash
    renamebinaryto: %q
    systempkgname: %q
    systempkgtype: %q
    filter: [linux]
    locked: false
    versionfilter: v
  other/locked:
    owner: other
    repo: locked
    version: v1.0.0
    locked: true
settings:
  token: test-token
`, filename, tc.rename, packageName, tc.packageType)
			if err := os.WriteFile(ConfigFile, []byte(legacy), 0600); err != nil {
				t.Fatal(err)
			}
			config, err := ReadConfig(ConfigFile)
			if err != nil {
				t.Fatal(err)
			}
			originalOther := config.Packages["other/locked"]
			data, err := os.ReadFile(ConfigFile)
			if err != nil || string(data) != legacy {
				t.Fatalf("reading rewrote the old config: %v", err)
			}
			fakeProgram(t, dir, "sudo", `exec "$@"`)
			for _, program := range []string{"cp", "chmod", "mv", "rm", "apt", "dpkg", "dnf"} {
				fakeProgram(t, dir, program, "exit 0")
			}
			fakeProgram(t, dir, "rpm", `case "$1" in
 -qp) echo tools-gui;;
 -qpl) echo /usr/bin/tools-gui;;
 -qf) exit 1;;
 *) exit 0;;
esac`)
			if tc.changedPackage {
				fakeProgram(t, dir, "rpm", `case "$1" in
 -qp) echo another-gui;;
 -qpl) echo /usr/bin/another-gui;;
 *) exit 1;;
esac`)
			}
			var contents bytes.Buffer
			writer := tar.NewWriter(&contents)
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			tarFile := filepath.Join(dir, "contents.tar")
			if err := os.WriteFile(tarFile, contents.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GRM_TEST_CONTENTS", tarFile)
			fakeProgram(t, dir, "dpkg-deb", `case "$1" in
 --field) echo tools-gui;;
 --fsys-tarfile) exec /bin/cat "$GRM_TEST_CONTENTS";;
 *) exit 1;;
esac`)
			t.Setenv("PATH", dir)
			for _, tag := range []string{"v2.0.0", "v3.0.0"} {
				files := map[string][]byte{
					"tools-" + tag + "/ya":   fixtureBinary,
					"tools-" + tag + "/yazi": fixtureBinary,
				}
				if tc.missingBinary {
					delete(files, "tools-"+tag+"/ya")
				}
				if tc.rename == "" {
					files["tools-"+tag+"/a-new-helper"] = fixtureBinary
				}
				archive := releaseArchive(t, files)
				payload, err := os.ReadFile(archive)
				if err != nil {
					t.Fatal(err)
				}
				binaryName := "tools_" + tag + "_linux_amd64.zip"
				if tc.raw {
					binaryName, payload = "grm", fixtureBinary
				}
				release := &github.RepositoryRelease{TagName: tag, Assets: []*github.ReleaseAsset{
					{ID: new(int64(1)), Name: new(binaryName)},
					{ID: new(int64(2)), Name: new("tools-gui-" + tag + ".x86_64.rpm")},
					{ID: new(int64(3)), Name: new("tools-gui_" + tag + "_amd64.deb")},
				}}
				if tc.missingType {
					release.Assets = release.Assets[:2]
				}
				http.DefaultTransport = updateTransport(func(request *http.Request) (*http.Response, error) {
					body := fixtureBinary
					if strings.HasSuffix(request.URL.Path, "/assets/1") {
						body = payload
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
				})
				pkg := config.Packages["owner/tools"]
				options := installOptions{}
				if tc.explicit && tag == "v2.0.0" {
					options.assets = []string{"tools_*_linux_amd64.zip", "tools-gui-*.rpm"}
					options.binaries, err = parseBinaryRequests([]string{"ya", "yazi"})
					if err != nil {
						t.Fatal(err)
					}
				}
				if tag == "v2.0.0" {
					options.rename = tc.renameTo
				}
				err = installRelease(release, &pkg, options)
				if tc.missingType || tc.changedPackage || tc.missingBinary {
					if err == nil {
						t.Fatal("expected missing legacy selection to fail")
					}
					data, readErr := os.ReadFile(ConfigFile)
					if readErr != nil || string(data) != legacy {
						t.Fatalf("failed preparation changed the original config: %v", readErr)
					}
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				config, err = ReadConfig(ConfigFile)
				if err != nil {
					t.Fatal(err)
				}
				saved := config.Packages["owner/tools"]
				wantCount := 1
				if tc.explicit {
					wantCount = 3
				}
				if saved.Version != tag || saved.Incomplete || len(saved.Components) != wantCount {
					t.Fatalf("migration changed the installed selection: %+v", saved)
				}
				if !tc.explicit {
					component := saved.Components[0]
					wantFilename := filename
					if tc.renameTo != "" {
						wantFilename = filepath.Join(DefaultBinDir, tc.renameTo)
					}
					if component.Filename != wantFilename || component.SystemPkgName != packageName || component.SystemPkgType != tc.packageType {
						t.Errorf("migration changed the existing installation: %+v", component)
					}
					if !tc.raw && tc.filename != "" && filepath.Base(component.Source) != "ya" {
						t.Errorf("migration selected a different binary: %+v", component)
					}
				}
				if saved.Filename != "" || saved.SystemPkgName != "" || saved.RenameBinaryTo != "" || len(saved.Selections) == 0 {
					t.Errorf("legacy fields were not migrated: %+v", saved)
				}
				other := config.Packages["other/locked"]
				if !reflect.DeepEqual(saved.Filter, []string{"linux"}) || saved.VersionFilter != "v" || config.Settings["token"] != "test-token" || !other.Locked || other.Version != originalOther.Version || other.GetFullName() != originalOther.GetFullName() {
					t.Error("migration changed filters, settings, or an unrelated locked package")
				}
			}
		})
	}
}

func TestRemoveLegacyInstallation(t *testing.T) {
	for _, kind := range []string{"binary", "deb", "rpm"} {
		t.Run(kind, func(t *testing.T) {
			config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			pkg := Package{Owner: "owner", Repo: "tools", Version: "v1.0.0"}
			want := InstalledComponent{Version: pkg.Version}
			if kind == "binary" {
				pkg.Filename, pkg.MD5 = "/example/bin/tool", "old-hash"
				want.Filename, want.MD5 = pkg.Filename, pkg.MD5
			} else {
				pkg.SystemPkgName, pkg.SystemPkgType = "tools-gui", kind
				want.SystemPkgName, want.SystemPkgType = pkg.SystemPkgName, pkg.SystemPkgType
			}
			config.Packages[pkg.GetFullName()] = pkg
			config.Settings["token"] = "test-token"
			if err := config.save(); err != nil {
				t.Fatal(err)
			}
			config, err = ReadConfig(config.path)
			if err != nil {
				t.Fatal(err)
			}
			pkg = config.Packages[pkg.GetFullName()]
			var removed []InstalledComponent
			if err := removeInstalledPackage(config, &pkg, func(component InstalledComponent) error {
				removed = append(removed, component)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			config, err = ReadConfig(config.path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(removed, []InstalledComponent{want}) || len(config.Packages) != 0 || config.Settings["token"] != "test-token" {
				t.Fatalf("legacy removal changed another component or lost settings: removed=%+v, config=%+v", removed, config)
			}
		})
	}
}
