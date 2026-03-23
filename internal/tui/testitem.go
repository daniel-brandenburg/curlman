package tui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/danielbrandenburg/curlman/internal/asserter"
	"github.com/danielbrandenburg/curlman/internal/discovery"
	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
)

// TestStatus represents the current state of a test item.
type TestStatus int

const (
	StatusPending TestStatus = iota
	StatusRunning
	StatusPass
	StatusFail
	StatusSkip
	StatusError
)

// TestItem holds all state for a single test (one request-assertion pair).
type TestItem struct {
	// Identity
	Label    string // display label, e.g. "users/create"
	Pair     discovery.TestPair
	DirGroup string // first path segment, e.g. "users", "" for root

	// Runtime state
	Status     TestStatus
	LastRun    time.Time // zero if never run
	Request    *parser.Request
	Response   *runner.Response
	Assertions []parser.Assertion
	Results    []asserter.Result
	RunErr     error
}

// TreeNode is a node in the left-panel tree — either a directory or a test.
type TreeNode struct {
	IsDir    bool
	Label    string // display name (last path segment only)
	DirGroup string // for dir nodes: the dir's own full path; for items: the parent dir full path
	Depth    int    // indentation depth (0 = root-level)
	ItemIdx  int    // index into items; -1 for dir nodes
	Expanded bool   // dir nodes only
}

// newTestItems converts discovered TestPairs into a flat TestItem slice.
func newTestItems(pairs []discovery.TestPair) []TestItem {
	items := make([]TestItem, 0, len(pairs))
	for _, pair := range pairs {
		items = append(items, TestItem{
			Label:    pair.Name,
			Pair:     pair,
			DirGroup: dirGroup(pair.Name),
			Status:   StatusPending,
		})
	}
	return items
}

// dirGroup returns the full parent directory path for grouping.
// "users/create" → "users", "auth/users/create" → "auth/users", "login" → "".
func dirGroup(name string) string {
	name = filepath.ToSlash(name)
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		return name[:idx]
	}
	return ""
}

// parentDir returns the parent directory of a path, or "" for top-level.
// "auth/users" → "auth", "auth" → "".
func parentDir(path string) string {
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[:idx]
	}
	return ""
}

// isAncestorChainExpanded returns true if dirPath and all its ancestors are expanded.
func isAncestorChainExpanded(dirPath string, expandedMap map[string]bool) bool {
	for dirPath != "" {
		if !expandedMap[dirPath] {
			return false
		}
		dirPath = parentDir(dirPath)
	}
	return true
}

// buildTree constructs the tree node slice from items.
// Directory nodes are inserted immediately before the first child encountered,
// with intermediate parent dirs created as needed for nested paths.
func buildTree(items []TestItem) []TreeNode {
	var nodes []TreeNode
	seenDirs := map[string]bool{}

	// ensureDir creates all intermediate dir nodes for dirPath if not yet seen.
	ensureDir := func(dirPath string) {
		parts := strings.Split(dirPath, "/")
		for i := range parts {
			path := strings.Join(parts[:i+1], "/")
			if seenDirs[path] {
				continue
			}
			seenDirs[path] = true
			nodes = append(nodes, TreeNode{
				IsDir:    true,
				Label:    parts[i],
				DirGroup: path,
				Depth:    i,
				ItemIdx:  -1,
				Expanded: true,
			})
		}
	}

	for i, item := range items {
		if item.DirGroup != "" {
			ensureDir(item.DirGroup)
		}
		depth := 0
		if item.DirGroup != "" {
			depth = strings.Count(item.DirGroup, "/") + 1
		}
		nodes = append(nodes, TreeNode{
			IsDir:    false,
			Label:    strings.TrimPrefix(filepath.ToSlash(item.Label), item.DirGroup+"/"),
			DirGroup: item.DirGroup,
			Depth:    depth,
			ItemIdx:  i,
		})
	}
	return nodes
}

// visibleNodes returns indices into tree of nodes currently visible.
// Uses a two-pass approach: first build expandedMap, then check ancestry.
func visibleNodes(tree []TreeNode) []int {
	// First pass: record expanded state for every dir node.
	expandedMap := map[string]bool{}
	for _, node := range tree {
		if node.IsDir {
			expandedMap[node.DirGroup] = node.Expanded
		}
	}

	// Second pass: a node is visible if all its ancestor dirs are expanded.
	var visible []int
	for i, node := range tree {
		if node.IsDir {
			parent := parentDir(node.DirGroup)
			if parent == "" || isAncestorChainExpanded(parent, expandedMap) {
				visible = append(visible, i)
			}
		} else {
			if node.DirGroup == "" || isAncestorChainExpanded(node.DirGroup, expandedMap) {
				visible = append(visible, i)
			}
		}
	}
	return visible
}
