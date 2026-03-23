package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TestPair holds paths for a matched .http/.assert file pair.
type TestPair struct {
	Name       string // relative path stem, e.g. "users/create"
	HTTPFile   string // absolute path to .http file
	AssertFile string // absolute path to .assert file (may be "")
}

// Discover walks root, finds all *.http files, checks for a sibling *.assert
// file, applies optional filter (stem name or path prefix), and returns a
// sorted slice.
func Discover(root, filter string) ([]TestPair, error) {
	var pairs []TestPair

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden dirs and common vendor directories
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".http") {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		base := filepath.Base(absPath)
		stem := strings.TrimSuffix(base, ".http")

		// Build a relative name from root for display and filtering
		relPath, err := filepath.Rel(root, absPath)
		if err != nil {
			relPath = absPath
		}
		// Normalize to forward slashes and strip .http extension
		relPath = filepath.ToSlash(relPath)
		relName := strings.TrimSuffix(relPath, ".http")

		// Apply filter: match stem name exactly, or path prefix.
		// Trailing slashes are stripped so "tests/" and "tests" both work.
		if filter != "" {
			filterNorm := strings.TrimRight(filepath.ToSlash(filter), "/")
			if stem != filterNorm && !strings.HasPrefix(relName+"/", filterNorm+"/") && relName != filterNorm {
				return nil
			}
		}

		// Check for sibling .assert file
		assertPath := strings.TrimSuffix(absPath, ".http") + ".assert"
		if _, statErr := os.Stat(assertPath); statErr != nil {
			assertPath = ""
		}

		pairs = append(pairs, TestPair{
			Name:       relName,
			HTTPFile:   absPath,
			AssertFile: assertPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Name < pairs[j].Name
	})

	return pairs, nil
}
