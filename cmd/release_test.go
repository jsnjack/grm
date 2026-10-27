package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReleaseCreatesReleaseAndUploadsAssets(t *testing.T) {
	content := []byte(`{"ok":true}`)
	asset := filepath.Join(t.TempDir(), "notes.json")
	if err := os.WriteFile(asset, content, 0644); err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	var uploaded []byte
	var uploadName, mediaType string
	withGitHubTransport(t, func(request *http.Request) (*http.Response, error) {
		var body []byte
		if request.Body != nil {
			var err error
			if body, err = io.ReadAll(request.Body); err != nil {
				return nil, err
			}
		}
		switch request.Method + " " + request.URL.Host + request.URL.Path {
		case "GET api.github.com/repos/owner/repo/releases/tags/v1.2.3":
			return httpResponse(request, http.StatusNotFound, []byte(`{"message":"Not Found"}`)), nil
		case "POST api.github.com/repos/owner/repo/releases":
			if err := json.Unmarshal(body, &created); err != nil {
				return nil, err
			}
			return httpResponse(request, http.StatusCreated, []byte(`{"id":7,"tag_name":"v1.2.3"}`)), nil
		case "POST uploads.github.com/repos/owner/repo/releases/7/assets":
			uploaded, uploadName, mediaType = body, request.URL.Query().Get("name"), request.Header.Get("Content-Type")
			return httpResponse(request, http.StatusCreated, []byte(`{"id":1}`)), nil
		}
		t.Errorf("unexpected request %s %s", request.Method, request.URL)
		return httpResponse(request, http.StatusInternalServerError, nil), nil
	})
	oldFiles, oldTag := releaseFilename, releaseTag
	t.Cleanup(func() { releaseFilename, releaseTag = oldFiles, oldTag })
	releaseFilename, releaseTag = []string{asset}, "v1.2.3"

	if err := releaseCmd.RunE(releaseCmd, []string{"owner/repo"}); err != nil {
		t.Fatal(err)
	}
	if want := map[string]any{"tag_name": "v1.2.3"}; !reflect.DeepEqual(created, want) {
		t.Fatalf("created release %v, want %v", created, want)
	}
	if string(uploaded) != string(content) || uploadName != "notes.json" || mediaType != "application/json" {
		t.Fatalf("uploaded %q as %q with type %q", uploaded, uploadName, mediaType)
	}
}
