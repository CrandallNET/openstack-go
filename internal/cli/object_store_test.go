package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestContainerSaveDownloadsAllObjects(t *testing.T) {
	objects := map[string]string{
		"root.txt":      "root content",
		"nested/one.md": "nested content",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/test-container":
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("marker") != "" {
				_, _ = w.Write([]byte("[]"))
				return
			}
			if err := json.NewEncoder(w).Encode([]map[string]any{
				{"name": "root.txt", "bytes": len(objects["root.txt"]), "content_type": "text/plain"},
				{"name": "nested/one.md", "bytes": len(objects["nested/one.md"]), "content_type": "text/markdown"},
			}); err != nil {
				t.Fatalf("encode object list: %v", err)
			}
		case r.Method == http.MethodGet:
			escapedName := r.URL.EscapedPath()[len("/test-container/"):]
			objectName, err := url.PathUnescape(escapedName)
			if err != nil {
				t.Fatalf("unescape object name: %v", err)
			}
			content, ok := objects[objectName]
			if !ok {
				t.Fatalf("unexpected object download %q", objectName)
			}
			_, _ = w.Write([]byte(content))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	workdir := t.TempDir()
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir test workdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	if err := containerSave(context.Background(), &bytes.Buffer{}, testObjectStorageClient(server.URL), []string{"test-container"}); err != nil {
		t.Fatalf("container save: %v", err)
	}
	for name, want := range objects {
		got, err := os.ReadFile(filepath.Join(workdir, name))
		if err != nil {
			t.Fatalf("read saved object %s: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("saved object %s = %q, want %q", name, string(got), want)
		}
	}
}

func TestObjectStoreAccountSetUnsetHeaders(t *testing.T) {
	var setSeen bool
	var unsetSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		switch {
		case r.Header.Get("X-Account-Meta-Color") == "blue":
			setSeen = true
		case r.Header.Get("X-Remove-Account-Meta-Color") == "x":
			unsetSeen = true
		default:
			t.Fatalf("unexpected account metadata headers: %#v", r.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := testObjectStorageClient(server.URL)
	if err := objectStoreAccountSet(context.Background(), &Options{CommandFlagList: map[string][]string{"property": {"Color=blue"}}}, client); err != nil {
		t.Fatalf("account set: %v", err)
	}
	if err := objectStoreAccountUnset(context.Background(), &Options{CommandFlagList: map[string][]string{"property": {"Color"}}}, client); err != nil {
		t.Fatalf("account unset: %v", err)
	}
	if !setSeen || !unsetSeen {
		t.Fatalf("missing account metadata request, set=%t unset=%t", setSeen, unsetSeen)
	}
}

func testObjectStorageClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/",
		ResourceBase:   baseURL + "/",
		Type:           "object-store",
	}
}
