package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContaining(t *testing.T) {
	root := t.TempDir()

	for _, dir := range []string{"api", "api-v2", "web/apps/site"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// A project reached through a symlinked root still matches real paths.
	link := filepath.Join(t.TempDir(), "projects")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}

	projects := []Project{
		{Name: "api", Path: filepath.Join(root, "api")},
		{Name: "api-v2", Path: filepath.Join(root, "api-v2")},
		{Name: "web", Path: filepath.Join(link, "web")},
	}

	tests := []struct {
		path string
		want string
	}{
		{filepath.Join(root, "api"), "api"},
		{filepath.Join(root, "api", "cmd"), "api"},
		{filepath.Join(root, "api-v2"), "api-v2"},
		{filepath.Join(root, "web", "apps", "site"), "web"},
		{root, ""},
		{"/", ""},
	}

	for _, test := range tests {
		got, ok := Containing(projects, test.path)

		if test.want == "" {
			if ok {
				t.Errorf("Containing(%q) = %q, want no match", test.path, got.Name)
			}
			continue
		}

		if !ok || got.Name != test.want {
			t.Errorf("Containing(%q) = %q, %v; want %q", test.path, got.Name, ok, test.want)
		}
	}
}
