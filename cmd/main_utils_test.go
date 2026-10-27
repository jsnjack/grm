package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/schollz/progressbar/v3"
)

// withGitHubTransport routes all HTTP requests to handler and isolates the
// token sources that CreateClient reads.
func withGitHubTransport(t *testing.T, handler func(*http.Request) (*http.Response, error)) {
	t.Helper()
	oldConfig, oldToken, oldTransport := ConfigFile, rootToken, http.DefaultTransport
	t.Cleanup(func() { ConfigFile, rootToken, http.DefaultTransport = oldConfig, oldToken, oldTransport })
	rootToken = ""
	t.Setenv("GITHUB_TOKEN", "")
	ConfigFile = filepath.Join(t.TempDir(), "grm.yaml")
	http.DefaultTransport = updateTransport(handler)
}

func httpResponse(request *http.Request, status int, body []byte) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(bytes.NewReader(body)), Request: request}
}

func TestGenerateRandomString(t *testing.T) {
	for _, n := range []int{0, 1, 6, 20} {
		t.Run(fmt.Sprintf("length %d", n), func(t *testing.T) {
			got := generateRandomString(n)
			if len(got) != n {
				t.Errorf("expected length %d, got %d (%q)", n, len(got), got)
			}
			for _, r := range got {
				if !strings.ContainsRune(letterBytes, r) {
					t.Errorf("unexpected character %q in %q", r, got)
				}
			}
		})
	}
}

func TestPosString(t *testing.T) {
	cases := []struct {
		name    string
		slice   []string
		element string
		want    int
	}{
		{"found first", []string{"a", "b", "c"}, "a", 0},
		{"found middle", []string{"a", "b", "c"}, "b", 1},
		{"not found", []string{"a", "b", "c"}, "z", -1},
		{"empty slice", []string{}, "a", -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := posString(c.slice, c.element); got != c.want {
				t.Errorf("posString(%v, %q) = %d, want %d", c.slice, c.element, got, c.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	cases := []struct {
		name    string
		slice   []string
		element string
		want    bool
	}{
		{"present", []string{"y", "yes"}, "yes", true},
		{"absent", []string{"y", "yes"}, "no", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containsString(c.slice, c.element); got != c.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", c.slice, c.element, got, c.want)
			}
		})
	}
}

func TestToMD5(t *testing.T) {
	t.Run("known content", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
			t.Fatalf("setup: %s", err)
		}
		got, err := tomd5(path)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		const want = "5d41402abc4b2a76b9719d911017c592" // md5("hello")
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := tomd5(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Errorf("expected error for missing file, got nil")
		}
	})
}

func TestGetTmpDir(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"grm tmp path", DefaultTmpDirPattern + "abcdef/somefile", "/tmp/grm.abcdef/"},
		{"unrelated path", "/usr/local/bin/tool", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getTmpDir(c.path); got != c.want {
				t.Errorf("getTmpDir(%q) = %q, want %q", c.path, got, c.want)
			}
		})
	}
}

func TestCreateClientTokenSources(t *testing.T) {
	tests := []struct {
		name, flag, config, env, want string
	}{
		{"flag", "flag-token", "config-token", "env-token", "Bearer flag-token"},
		{"config", "", "config-token", "env-token", "Bearer config-token"},
		{"env", "", "", "env-token", "Bearer env-token"},
		{"anonymous", "", "", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got string
			withGitHubTransport(t, func(request *http.Request) (*http.Response, error) {
				got = request.Header.Get("Authorization")
				return httpResponse(request, http.StatusOK, []byte(`{"tag_name":"v1"}`)), nil
			})
			rootToken = test.flag
			t.Setenv("GITHUB_TOKEN", test.env)
			if test.config != "" {
				config, err := ReadConfig(ConfigFile)
				if err != nil {
					t.Fatal(err)
				}
				config.Settings["token"] = test.config
				if err := config.save(); err != nil {
					t.Fatal(err)
				}
			}
			client, err := CreateClient()
			if err != nil {
				t.Fatal(err)
			}
			release, _, err := client.Repositories.GetLatestRelease(context.Background(), "owner", "repo")
			if err != nil {
				t.Fatal(err)
			}
			if release.GetTagName() != "v1" || got != test.want {
				t.Fatalf("got tag %q and Authorization %q, want v1 and %q", release.GetTagName(), got, test.want)
			}
		})
	}
}

func TestDownloadFileDoesNotSendTokenToStorage(t *testing.T) {
	var apiAuth string
	storageAuth := []string{"not requested"}
	withGitHubTransport(t, func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "api.github.com":
			apiAuth = request.Header.Get("Authorization")
			response := httpResponse(request, http.StatusFound, nil)
			response.Header.Set("Location", "https://storage.example.com/tool")
			return response, nil
		case "storage.example.com":
			storageAuth = request.Header.Values("Authorization")
			return httpResponse(request, http.StatusOK, fixtureBinary), nil
		}
		return httpResponse(request, http.StatusNotFound, nil), nil
	})
	rootToken = "secret"
	asset := &github.ReleaseAsset{ID: new(int64(1)), Name: new("tool"), Size: new(len(fixtureBinary))}
	destination, err := downloadFile(asset, &Package{Owner: "owner", Repo: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(destination)) })
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, fixtureBinary) {
		t.Fatalf("downloaded %v, want %v", data, fixtureBinary)
	}
	if apiAuth != "Bearer secret" {
		t.Fatalf("API request Authorization = %q, want the token", apiAuth)
	}
	if len(storageAuth) != 0 {
		t.Fatalf("storage request Authorization = %q, want none", storageAuth)
	}
}

func TestProgressReader(t *testing.T) {
	data := bytes.Repeat([]byte("grm"), 4096)
	bar := progressbar.NewOptions(len(data), progressbar.OptionShowBytes(true), progressbar.OptionSetWriter(io.Discard))
	got, err := io.ReadAll(&ProgressReader{r: bytes.NewReader(data), bar: bar})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("read %d bytes, want %d unchanged bytes", len(got), len(data))
	}
	if state := bar.State(); state.CurrentNum != int64(len(data)) || state.CurrentPercent != 1 {
		t.Fatalf("bar at %d (%.2f), want %d (1.00)", state.CurrentNum, state.CurrentPercent, len(data))
	}
}
