package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielbrandenburg/curlman/internal/asserter"
	"github.com/danielbrandenburg/curlman/internal/discovery"
	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
)

// testResultMsg is sent when a single test finishes executing.
type testResultMsg struct {
	index      int
	status     TestStatus
	lastRun    time.Time
	request    *parser.Request
	response   *runner.Response
	assertions []parser.Assertion
	results    []asserter.Result
	err        error
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	// Configuration
	envVars map[string]string

	// Test data
	items []TestItem

	// Tree state
	tree        []TreeNode
	visibleTree []int // indices into tree of currently visible nodes
	treeCursor  int   // index into visibleTree

	// Panel state
	activePanel int // 0 = left, 1 = right
	viewport    viewport.Model
	leftWidth   int // content width of left panel

	// Run state
	running int
	hasRun  bool // true once at least one test has been dispatched

	// Spinner
	spinner spinner.Model

	// Terminal dimensions
	width  int
	height int

	// Key bindings
	keys keyMap
}

// New creates a new TUI Model. Tests are NOT run automatically.
func New(pairs []discovery.TestPair, vars map[string]string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorRunning)

	items := newTestItems(pairs)
	tree := buildTree(items)
	visible := visibleNodes(tree)

	return Model{
		envVars:     vars,
		items:       items,
		tree:        tree,
		visibleTree: visible,
		spinner:     sp,
		keys:        defaultKeyMap(),
	}
}

// Init does nothing — tests start only on user action.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.recalcLayout()
		m.viewport.SetContent(renderDetail(m))
		return m, nil

	case testResultMsg:
		if msg.index >= 0 && msg.index < len(m.items) {
			it := &m.items[msg.index]
			it.Status = msg.status
			it.LastRun = msg.lastRun
			it.RunErr = msg.err
			if msg.request != nil {
				it.Request = msg.request
			}
			if msg.response != nil {
				it.Response = msg.response
			}
			if msg.assertions != nil {
				it.Assertions = msg.assertions
			}
			if msg.results != nil {
				it.Results = msg.results
			}
			m.running--
			if m.running < 0 {
				m.running = 0
			}
			// Refresh detail panel if the result is for the selected test
			if msg.index == m.selectedItemIdx() {
				m.viewport.SetContent(renderDetail(m))
			}
		}
		return m, nil

	case spinner.TickMsg:
		if m.running > 0 {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Always handle quit/tab regardless of active panel
		switch {
		case keyMatches(msg, m.keys.Quit):
			return m, tea.Quit

		case keyMatches(msg, m.keys.Tab):
			m.activePanel = 1 - m.activePanel
			return m, nil

		case keyMatches(msg, m.keys.RunAll):
			return m.runAll()
		}

		// Left-panel navigation
		if m.activePanel == 0 {
			switch {
			case keyMatches(msg, m.keys.Up):
				if m.treeCursor > 0 {
					m.treeCursor--
					m.viewport.SetContent(renderDetail(m))
					m.viewport.GotoTop()
				}
				return m, nil

			case keyMatches(msg, m.keys.Down):
				if m.treeCursor < len(m.visibleTree)-1 {
					m.treeCursor++
					m.viewport.SetContent(renderDetail(m))
					m.viewport.GotoTop()
				}
				return m, nil

			case keyMatches(msg, m.keys.Enter):
				node := m.selectedNode()
				if node == nil {
					return m, nil
				}
				if node.IsDir {
					return m.toggleDir(), nil
				}
				return m.runItem(node.ItemIdx)

			case keyMatches(msg, m.keys.Run):
				node := m.selectedNode()
				if node == nil {
					return m, nil
				}
				if node.IsDir {
					return m.runDir(node.DirGroup)
				}
				return m.runItem(node.ItemIdx)
			}
			return m, nil
		}

		// Right-panel scroll
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// Pass mouse/other events to viewport when right panel is active
	if m.activePanel == 1 {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	if m.width < 40 {
		return "Terminal too narrow (min 40 cols)"
	}

	contentH := m.contentHeight()
	innerH := contentH - 2 // subtract top+bottom border

	leftStyle := panelStyle
	if m.activePanel == 0 {
		leftStyle = activePanelStyle
	}
	leftPanel := leftStyle.
		Width(m.leftWidth).
		Height(innerH).
		Render(renderList(m, innerH))

	rightStyle := panelStyle
	if m.activePanel == 1 {
		rightStyle = activePanelStyle
	}
	m.viewport.SetContent(renderDetail(m))
	rightPanel := rightStyle.
		Width(m.width - m.leftWidth - 4).
		Height(innerH).
		Render(m.viewport.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	return lipgloss.JoinVertical(lipgloss.Left, renderStatusBar(m), content, renderKeyBar(m))
}

// --- Tree helpers ---

// selectedNode returns the TreeNode under the cursor, or nil if none.
func (m Model) selectedNode() *TreeNode {
	if len(m.visibleTree) == 0 || m.treeCursor >= len(m.visibleTree) {
		return nil
	}
	n := &m.tree[m.visibleTree[m.treeCursor]]
	return n
}

// selectedItemIdx returns the item index of the selected tree node, or -1 if
// the cursor is on a dir node or there are no items.
func (m Model) selectedItemIdx() int {
	n := m.selectedNode()
	if n == nil || n.IsDir {
		return -1
	}
	return n.ItemIdx
}

// toggleDir flips the expanded state of the currently selected dir node.
func (m Model) toggleDir() Model {
	n := m.selectedNode()
	if n == nil || !n.IsDir {
		return m
	}
	// Find the node in tree and flip it
	for i := range m.tree {
		if m.tree[i].IsDir && m.tree[i].DirGroup == n.DirGroup {
			m.tree[i].Expanded = !m.tree[i].Expanded
			break
		}
	}
	m.visibleTree = visibleNodes(m.tree)
	// Clamp cursor
	if m.treeCursor >= len(m.visibleTree) {
		m.treeCursor = len(m.visibleTree) - 1
	}
	if m.treeCursor < 0 {
		m.treeCursor = 0
	}
	return m
}

// --- Run actions ---

func (m Model) resetItem(idx int) {
	m.items[idx].Status = StatusPending
	m.items[idx].Request = nil
	m.items[idx].Response = nil
	m.items[idx].Assertions = nil
	m.items[idx].Results = nil
	m.items[idx].RunErr = nil
}

// runItem runs a single test by item index.
func (m Model) runItem(idx int) (Model, tea.Cmd) {
	m.resetItem(idx)
	m.items[idx].Status = StatusRunning
	m.running++
	m.hasRun = true
	m.viewport.SetContent(renderDetail(m))
	return m, tea.Batch(m.spinner.Tick, runTestCmd(idx, m.items[idx].Pair, m.envVars))
}

// runDir runs all tests whose DirGroup matches group.
func (m Model) runDir(group string) (Model, tea.Cmd) {
	cmds := []tea.Cmd{m.spinner.Tick}
	for i := range m.items {
		if m.items[i].DirGroup == group {
			m.resetItem(i)
			m.items[i].Status = StatusRunning
			m.running++
			cmds = append(cmds, runTestCmd(i, m.items[i].Pair, m.envVars))
		}
	}
	m.hasRun = true
	m.viewport.SetContent(renderDetail(m))
	return m, tea.Batch(cmds...)
}

// runAll runs every test.
func (m Model) runAll() (Model, tea.Cmd) {
	cmds := []tea.Cmd{m.spinner.Tick}
	for i := range m.items {
		m.resetItem(i)
		m.items[i].Status = StatusRunning
		m.running++
		cmds = append(cmds, runTestCmd(i, m.items[i].Pair, m.envVars))
	}
	m.hasRun = true
	m.viewport.SetContent(renderDetail(m))
	return m, tea.Batch(cmds...)
}

// --- Layout helpers ---

func (m Model) recalcLayout() Model {
	leftW := m.width * 35 / 100
	if leftW < 25 {
		leftW = 25
	}
	if leftW > 45 {
		leftW = 45
	}
	m.leftWidth = leftW

	vpWidth := m.width - leftW - 6
	vpHeight := m.contentHeight() - 2
	if vpWidth < 10 {
		vpWidth = 10
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = viewport.New(vpWidth, vpHeight)
	return m
}

func (m Model) contentHeight() int {
	h := m.height - 2
	if h < 3 {
		h = 3
	}
	return h
}

// keyMatches reports whether msg matches a key binding.
func keyMatches(msg tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if k == msg.String() {
			return true
		}
	}
	return false
}
