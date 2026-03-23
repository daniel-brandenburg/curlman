package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file at dir/relPath with the given content.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestDiscover_SinglePairWithAssert(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "check.http", "GET http://example.com\n")
	writeFile(t, root, "check.assert", "STATUS 200\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Name != "check" {
		t.Errorf("unexpected name: %q", pairs[0].Name)
	}
	if pairs[0].AssertFile == "" {
		t.Error("expected AssertFile to be set")
	}
}

func TestDiscover_HTTPWithoutAssert(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "noassert.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].AssertFile != "" {
		t.Errorf("expected empty AssertFile, got %q", pairs[0].AssertFile)
	}
}

func TestDiscover_NestedDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "sub/test.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Name != "sub/test" {
		t.Errorf("expected sub/test, got %q", pairs[0].Name)
	}
}

func TestDiscover_HiddenDirectorySkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".hidden/test.http", "GET http://example.com\n")
	writeFile(t, root, "visible.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range pairs {
		if p.Name == ".hidden/test" {
			t.Error("hidden directory should be skipped")
		}
	}
	if len(pairs) != 1 {
		t.Errorf("expected 1 pair (visible only), got %d", len(pairs))
	}
}

func TestDiscover_VendorDirectorySkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "vendor/test.http", "GET http://example.com\n")
	writeFile(t, root, "real.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 || pairs[0].Name != "real" {
		t.Errorf("vendor dir should be skipped, got: %v", pairs)
	}
}

func TestDiscover_NodeModulesSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "node_modules/test.http", "GET http://example.com\n")
	writeFile(t, root, "app.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 || pairs[0].Name != "app" {
		t.Errorf("node_modules should be skipped, got: %v", pairs)
	}
}

func TestDiscover_FilterByStem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "health/check.http", "GET http://example.com\n")
	writeFile(t, root, "users/list.http", "GET http://example.com\n")

	pairs, err := Discover(root, "check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 || pairs[0].Name != "health/check" {
		t.Errorf("expected only check, got: %v", pairs)
	}
}

func TestDiscover_FilterByPrefix(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/list.http", "GET http://example.com\n")
	writeFile(t, root, "users/create.http", "POST http://example.com\n")
	writeFile(t, root, "health/check.http", "GET http://example.com\n")

	pairs, err := Discover(root, "users/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs for users/ prefix, got %d: %v", len(pairs), pairs)
	}
	for _, p := range pairs {
		if filepath.ToSlash(p.Name)[:6] != "users/" {
			t.Errorf("non-users pair returned: %q", p.Name)
		}
	}
}

func TestDiscover_EmptyRootNoPairs(t *testing.T) {
	root := t.TempDir()
	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs, got %d", len(pairs))
	}
}

func TestDiscover_FilterMatchesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "health/check.http", "GET http://example.com\n")

	pairs, err := Discover(root, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for non-matching filter, got %d", len(pairs))
	}
}

func TestDiscover_SortedOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "z.http", "GET http://example.com\n")
	writeFile(t, root, "a.http", "GET http://example.com\n")
	writeFile(t, root, "m.http", "GET http://example.com\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}
	if pairs[0].Name != "a" || pairs[1].Name != "m" || pairs[2].Name != "z" {
		t.Errorf("not sorted: %v", pairs)
	}
}

func TestDiscover_DeepNesting(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a/b/c/deep.http", "GET http://example.com\n")
	writeFile(t, root, "a/b/c/deep.assert", "STATUS 200\n")

	pairs, err := Discover(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Name != "a/b/c/deep" {
		t.Errorf("unexpected name: %q", pairs[0].Name)
	}
	if pairs[0].AssertFile == "" {
		t.Error("expected assert file to be found")
	}
}
