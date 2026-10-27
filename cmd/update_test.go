package cmd

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUpdateReportsReleaseCheckFailures(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "failed check", true: "failed and current checks"}[mixed], func(t *testing.T) {
			oldConfig, oldToken := ConfigFile, rootToken
			oldTransport := http.DefaultTransport
			t.Cleanup(func() {
				ConfigFile, rootToken = oldConfig, oldToken
				http.DefaultTransport = oldTransport
			})
			rootToken = ""
			t.Setenv("GITHUB_TOKEN", "")
			ConfigFile = filepath.Join(t.TempDir(), "grm.yaml")
			config, err := ReadConfig(ConfigFile)
			if err != nil {
				t.Fatal(err)
			}
			config.Packages["owner/broken"] = Package{Owner: "owner", Repo: "broken", Version: "v1"}
			if mixed {
				config.Packages["owner/current"] = Package{Owner: "owner", Repo: "current", Version: "v1"}
			}
			if err := config.save(); err != nil {
				t.Fatal(err)
			}
			http.DefaultTransport = updateTransport(func(request *http.Request) (*http.Response, error) {
				status, body := http.StatusServiceUnavailable, `{"message":"release lookup failed"}`
				if request.URL.Path == "/repos/owner/current/releases/latest" {
					status, body = http.StatusOK, `{"tag_name":"v1"}`
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
			})
			output, err := os.CreateTemp(t.TempDir(), "output")
			if err != nil {
				t.Fatal(err)
			}
			defer logClose(output)
			oldOutput := os.Stdout
			os.Stdout = output
			t.Cleanup(func() { os.Stdout = oldOutput })
			err = updateCmd.RunE(updateCmd, nil)
			if err == nil || !strings.Contains(err.Error(), "owner/broken") {
				t.Errorf("expected failed repository in returned error, got %v", err)
			}
			data, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "All packages are up to date") {
				t.Errorf("failed lookup was reported as current: %s", data)
			}
		})
	}
}
