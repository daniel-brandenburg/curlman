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
	Label    string // directory name or test base name
	DirGroup string // the directory path this node belongs to (or is)
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

// dirGroup returns the first path segment for directory grouping.
// "users/create" → "users", "login" → "".
func dirGroup(name string) string {
	name = filepath.ToSlash(name)
	if idx := strings.Index(name, "/"); idx != -1 {
		return name[:idx]
	}
	return ""
}

// buildTree constructs the tree node slice from items.
// Directory nodes are inserted immediately before their children.
func buildTree(items []TestItem) []TreeNode {
	var nodes []TreeNode
	seenGroups := map[string]bool{}

	for i, item := range items {
		if item.DirGroup == "" {
			nodes = append(nodes, TreeNode{
				IsDir:   false,
				Label:   item.Label,
				ItemIdx: i,
			})
		} else {
			if !seenGroups[item.DirGroup] {
				seenGroups[item.DirGroup] = true
				nodes = append(nodes, TreeNode{
					IsDir:    true,
					Label:    item.DirGroup,
					DirGroup: item.DirGroup,
					ItemIdx:  -1,
					Expanded: true,
				})
			}
			nodes = append(nodes, TreeNode{
				IsDir:    false,
				Label:    strings.TrimPrefix(item.Label, item.DirGroup+"/"),
				DirGroup: item.DirGroup,
				ItemIdx:  i,
			})
		}
	}
	return nodes
}

// visibleNodes returns indices into tree of nodes currently visible
// (i.e. all dir nodes, root-level tests, and children of expanded dirs).
func visibleNodes(tree []TreeNode) []int {
	var visible []int
	var currentDir string
	var dirExpanded bool

	for i, node := range tree {
		if node.IsDir {
			currentDir = node.DirGroup
			dirExpanded = node.Expanded
			visible = append(visible, i)
		} else if node.DirGroup == "" {
			visible = append(visible, i)
		} else if node.DirGroup == currentDir && dirExpanded {
			visible = append(visible, i)
		}
	}
	return visible
}
