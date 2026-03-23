package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/danielbrandenburg/curlman/internal/asserter"
	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

// statusIcon returns the colored icon for a given test status.
func statusIcon(status TestStatus, spinnerFrame string) string {
	switch status {
	case StatusPass:
		return passIconStyle.Render("✓")
	case StatusFail:
		return failIconStyle.Render("✗")
	case StatusSkip:
		return lipgloss.NewStyle().Foreground(colorSkip).Render("-")
	case StatusError:
		return failIconStyle.Render("!")
	case StatusRunning:
		return lipgloss.NewStyle().Foreground(colorRunning).Render(spinnerFrame)
	default:
		return mutedStyle.Render("○")
	}
}

// dirStatus returns the aggregate status of all items in a directory group.
func dirStatus(items []TestItem, group string) TestStatus {
	hasFail := false
	hasRunning := false
	hasPass := false
	allPending := true

	for _, item := range items {
		if item.DirGroup != group {
			continue
		}
		allPending = allPending && item.Status == StatusPending
		switch item.Status {
		case StatusFail, StatusError:
			hasFail = true
		case StatusRunning:
			hasRunning = true
		case StatusPass, StatusSkip:
			hasPass = true
		}
	}

	if hasFail {
		return StatusFail
	}
	if hasRunning {
		return StatusRunning
	}
	if hasPass && !allPending {
		return StatusPass
	}
	return StatusPending
}

// renderStatusBar renders the top status bar line.
func renderStatusBar(m Model) string {
	passed, failed, skipped := 0, 0, 0
	for _, item := range m.items {
		switch item.Status {
		case StatusPass:
			passed++
		case StatusFail, StatusError:
			failed++
		case StatusSkip:
			skipped++
		}
	}

	bg := lipgloss.Color("#1e293b")
	title := statusBarTitleStyle.Render("curlman")

	passStr := passIconStyle.Copy().Background(bg).Render(fmt.Sprintf("  ✓%d", passed))
	failStr := failIconStyle.Copy().Background(bg).Render(fmt.Sprintf("  ✗%d", failed))
	skipStr := lipgloss.NewStyle().Foreground(colorSkip).Background(bg).Render(fmt.Sprintf("  -%d", skipped))

	var runIndicator string
	if m.running > 0 {
		runIndicator = "  " + lipgloss.NewStyle().Foreground(colorRunning).Background(bg).Render(m.spinner.View()+" running…")
	} else if m.hasRun {
		runIndicator = "  " + mutedStyle.Copy().Background(bg).Render("done")
	} else {
		runIndicator = "  " + mutedStyle.Copy().Background(bg).Render("press R to run all")
	}

	right := statusBarStyle.Render(
		fmt.Sprintf(" %d tests", len(m.items)) + passStr + failStr + skipStr + runIndicator,
	)

	titleWidth := lipgloss.Width(title)
	rightWidth := lipgloss.Width(right)
	gap := m.width - titleWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		statusBarStyle.Render(strings.Repeat(" ", gap)),
		right,
	)
}

// renderKeyBar renders the bottom keybinding bar.
func renderKeyBar(m Model) string {
	node := m.selectedNode()

	var pairs []string
	if m.activePanel == 0 {
		pairs = append(pairs, keyStyle.Render("↑↓")+keyDescStyle.Render(" navigate"))
		if node != nil && node.IsDir {
			pairs = append(pairs, keyStyle.Render("enter")+keyDescStyle.Render(" toggle"))
			pairs = append(pairs, keyStyle.Render("r")+keyDescStyle.Render(" run folder"))
		} else {
			pairs = append(pairs, keyStyle.Render("enter/r")+keyDescStyle.Render(" run"))
		}
		pairs = append(pairs, keyStyle.Render("R")+keyDescStyle.Render(" run all"))
	} else {
		pairs = append(pairs, keyStyle.Render("↑↓")+keyDescStyle.Render(" scroll"))
	}
	pairs = append(pairs,
		keyStyle.Render("tab")+keyDescStyle.Render(" switch panel"),
		keyStyle.Render("q")+keyDescStyle.Render(" quit"),
	)

	sep := keyDescStyle.Render("  ·  ")
	return keyBarStyle.Width(m.width).Render(strings.Join(pairs, sep))
}

// renderList renders the left panel tree.
func renderList(m Model, innerHeight int) string {
	if len(m.visibleTree) == 0 {
		return mutedStyle.Render("No tests found.")
	}

	// Determine scroll window to keep cursor visible
	start := 0
	end := innerHeight
	cursor := m.treeCursor

	if end > len(m.visibleTree) {
		end = len(m.visibleTree)
	}
	if cursor < start {
		start = cursor
		end = start + innerHeight
		if end > len(m.visibleTree) {
			end = len(m.visibleTree)
		}
	}
	if cursor >= end {
		end = cursor + 1
		start = end - innerHeight
		if start < 0 {
			start = 0
		}
	}

	var sb strings.Builder
	total := end - start
	for i := start; i < end; i++ {
		nodeIdx := m.visibleTree[i]
		node := m.tree[nodeIdx]
		selected := i == m.treeCursor

		var line string
		if node.IsDir {
			arrow := "▼"
			if !node.Expanded {
				arrow = "▶"
			}
			ds := dirStatus(m.items, node.DirGroup)
			icon := statusIcon(ds, m.spinner.View())
			raw := fmt.Sprintf("%s %s %s/", arrow, icon, node.Label)
			if selected {
				line = selectedItemStyle.Width(m.leftWidth).Render(raw)
			} else {
				line = groupHeaderStyle.Render(raw)
			}
		} else {
			icon := statusIcon(m.items[node.ItemIdx].Status, m.spinner.View())
			indent := "  "
			if node.DirGroup != "" {
				indent = "    " // extra indent under dir
			}
			raw := fmt.Sprintf("%s%s %s", indent, icon, node.Label)
			if selected {
				line = selectedItemStyle.Width(m.leftWidth).Render(raw)
			} else {
				line = raw
			}
		}

		if i < end-1 || total > 0 {
			sb.WriteString(line + "\n")
		} else {
			sb.WriteString(line)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderDetail builds the right panel content for the selected tree node.
func renderDetail(m Model) string {
	node := m.selectedNode()
	if node == nil {
		return mutedStyle.Render("No selection.")
	}

	if node.IsDir {
		return renderDirDetail(m, node)
	}

	item := m.items[node.ItemIdx]
	var sb strings.Builder

	title := detailTitleStyle.Render(item.Label)
	if !item.LastRun.IsZero() {
		icon := statusIcon(item.Status, m.spinner.View())
		ts := mutedStyle.Render(item.LastRun.Format("15:04:05"))
		title += "  " + icon + "  " + ts
	}
	sb.WriteString(title + "\n")

	switch item.Status {
	case StatusPending:
		sb.WriteString("\n" + mutedStyle.Render("○  Not run yet.  Press enter or r to run."))
		return sb.String()
	case StatusRunning:
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(colorRunning).Render(m.spinner.View()+"  Running…"))
		return sb.String()
	}

	if item.RunErr != nil {
		sb.WriteString("\n" + errorBoxStyle.Render("ERROR\n"+item.RunErr.Error()) + "\n")
	}
	if item.Request != nil {
		sb.WriteString(renderReq(item.Request))
	}
	if item.Response != nil {
		sb.WriteString(renderResp(item.Response))
	}
	if item.Results != nil {
		sb.WriteString(renderAssertions(item.Assertions, item.Results))
	} else if item.Status == StatusSkip {
		sb.WriteString("\n" + sectionHeaderStyle.Render("ASSERTIONS") + "\n")
		sb.WriteString(mutedStyle.Render("  -  no assertions"))
	}

	return sb.String()
}

// renderDirDetail renders a summary for a directory node.
func renderDirDetail(m Model, node *TreeNode) string {
	var sb strings.Builder
	sb.WriteString(detailTitleStyle.Render(node.Label+"/") + "\n")

	pass, fail, skip, pending, running := 0, 0, 0, 0, 0
	for _, item := range m.items {
		if item.DirGroup != node.DirGroup {
			continue
		}
		switch item.Status {
		case StatusPass:
			pass++
		case StatusFail, StatusError:
			fail++
		case StatusSkip:
			skip++
		case StatusRunning:
			running++
		default:
			pending++
		}
	}

	total := pass + fail + skip + pending + running
	sb.WriteString("\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("%d tests", total)) + "\n")
	if pass > 0 {
		sb.WriteString(passIconStyle.Render(fmt.Sprintf("  ✓ %d passed", pass)) + "\n")
	}
	if fail > 0 {
		sb.WriteString(failIconStyle.Render(fmt.Sprintf("  ✗ %d failed", fail)) + "\n")
	}
	if skip > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorSkip).Render(fmt.Sprintf("  - %d skipped", skip)) + "\n")
	}
	if running > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorRunning).Render(fmt.Sprintf("  %s %d running…", m.spinner.View(), running)) + "\n")
	}
	if pending > 0 && !m.hasRun {
		sb.WriteString("\n" + mutedStyle.Render("Press r to run this folder, R to run all."))
	}

	return sb.String()
}

func renderReq(req *parser.Request) string {
	var sb strings.Builder
	sb.WriteString("\n" + sectionHeaderStyle.Render("REQUEST") + "\n")
	sb.WriteString(methodStyle.Render(req.Method) + " " + urlStyle.Render(req.URL) + "\n")

	keys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.WriteString(headerKeyStyle.Render(k+":") + " " + headerValStyle.Render(req.Headers[k]) + "\n")
	}
	if req.Body != "" {
		sb.WriteString("\n" + prettyBody(req.Body) + "\n")
	}
	return sb.String()
}

func renderResp(resp *runner.Response) string {
	var sb strings.Builder
	sb.WriteString("\n" + sectionHeaderStyle.Render("RESPONSE") + "\n")

	codeStr := fmt.Sprintf("%d", resp.StatusCode)
	var codeStyled string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		codeStyled = statusOKStyle.Render(codeStr)
	} else {
		codeStyled = statusErrStyle.Render(codeStr)
	}
	sb.WriteString(codeStyled + "  " + durationStyle.Render(fmt.Sprintf("%dms", resp.Duration.Milliseconds())) + "\n")

	for _, k := range []string{"content-type", "content-length", "location"} {
		if v, ok := resp.Headers[k]; ok {
			sb.WriteString(headerKeyStyle.Render(k+":") + " " + headerValStyle.Render(v) + "\n")
		}
	}
	if resp.Body != "" {
		sb.WriteString("\n" + prettyBody(resp.Body) + "\n")
	}
	return sb.String()
}

func renderAssertions(assertions []parser.Assertion, results []asserter.Result) string {
	var sb strings.Builder
	sb.WriteString("\n" + sectionHeaderStyle.Render("ASSERTIONS") + "\n")
	for i, r := range results {
		var label string
		if i < len(assertions) {
			label = assertionLabel(assertions[i])
		}
		if r.Passed {
			sb.WriteString(passIconStyle.Render("  ✓ ") + mutedStyle.Render(label) + "\n")
		} else {
			sb.WriteString(failIconStyle.Render("  ✗ ") + failMsgStyle.Render(r.Message) + "\n")
		}
	}
	return sb.String()
}

func assertionLabel(a parser.Assertion) string {
	switch a.Kind {
	case parser.AssertStatus:
		return fmt.Sprintf("STATUS %d", a.StatusCode)
	case parser.AssertHeader:
		return fmt.Sprintf("HEADER %s: %s", a.HeaderKey, a.HeaderVal)
	case parser.AssertBody:
		if a.Operator == "exists" || a.Operator == "not exists" {
			return fmt.Sprintf("BODY %s %s", a.JSONPath, a.Operator)
		}
		return fmt.Sprintf("BODY %s %s %s", a.JSONPath, a.Operator, a.Expected)
	default:
		return ""
	}
}

func prettyBody(body string) string {
	if gjson.Valid(body) {
		return bodyStyle.Render(strings.TrimSpace(string(pretty.Pretty([]byte(body)))))
	}
	return bodyStyle.Render(body)
}

func clamp(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}
