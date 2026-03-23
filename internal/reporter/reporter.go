package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danielbrandenburg/curlman/internal/asserter"
)

// Mode controls the output style.
type Mode int

const (
	ModePretty  Mode = iota // colored, grouped by directory, ✓/✗ icons (default)
	ModePlain               // PASS/FAIL text, no grouping
	ModeVerbose             // like plain + request/response details before each result
)

// TestDetails carries request/response info for verbose output.
type TestDetails struct {
	Method     string
	URL        string
	StatusCode int
	Duration   time.Duration
	Body       string
}

// Reporter produces test output in the configured mode.
type Reporter struct {
	mode      Mode
	useColor  bool
	group     string // pretty mode: current directory group
	hasOutput bool   // pretty mode: whether any line has been printed yet
}

// New creates a Reporter for the given mode.
func New(mode Mode) *Reporter {
	return &Reporter{
		mode:     mode,
		useColor: shouldUseColor(),
	}
}

func shouldUseColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ANSI escape codes
const (
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiReset  = "\033[0m"
)

func (r *Reporter) c(code, s string) string {
	if !r.useColor {
		return s
	}
	return code + s + ansiReset
}

// --- Pretty mode helpers ---

func groupOf(name string) string {
	if i := strings.Index(name, "/"); i != -1 {
		return name[:i]
	}
	return ""
}

func shortLabel(name string) string {
	if i := strings.Index(name, "/"); i != -1 {
		return name[i+1:]
	}
	return name
}

func (r *Reporter) prettyLabel(name string) string {
	if groupOf(name) != "" {
		return shortLabel(name)
	}
	return name
}

func (r *Reporter) prettyIndent(name string) string {
	if groupOf(name) != "" {
		return "    "
	}
	return "  "
}

// printGroup emits a directory group header in pretty mode when the group changes.
func (r *Reporter) printGroup(name string) {
	if r.mode != ModePretty {
		return
	}
	g := groupOf(name)
	if g == r.group {
		return
	}
	if r.hasOutput {
		fmt.Println()
	}
	r.group = g
	if g != "" {
		fmt.Printf("  %s\n", r.c(ansiBold, g+"/"))
	}
}

// --- Public output methods ---

// Pass prints a passing test.
func (r *Reporter) Pass(name string, duration time.Duration) {
	r.printGroup(name)
	switch r.mode {
	case ModePretty:
		fmt.Printf("%s%s  %s  %s\n",
			r.prettyIndent(name),
			r.c(ansiGreen, "✓"),
			r.prettyLabel(name),
			r.c(ansiDim, fmt.Sprintf("%dms", duration.Milliseconds())),
		)
	default:
		fmt.Printf("%s  %s  %s\n",
			r.c(ansiGreen, "PASS"),
			name,
			r.c(ansiDim, fmt.Sprintf("(%dms)", duration.Milliseconds())),
		)
	}
	r.hasOutput = true
}

// Fail prints a failing test with failure details.
func (r *Reporter) Fail(name string, duration time.Duration, failures []asserter.Result) {
	r.printGroup(name)
	switch r.mode {
	case ModePretty:
		indent := r.prettyIndent(name)
		fmt.Printf("%s%s  %s  %s\n",
			indent,
			r.c(ansiRed, "✗"),
			r.prettyLabel(name),
			r.c(ansiDim, fmt.Sprintf("%dms", duration.Milliseconds())),
		)
		for _, f := range failures {
			if !f.Passed {
				fmt.Printf("%s%s\n", indent+"     ", r.c(ansiRed, f.Message))
			}
		}
	default:
		fmt.Printf("%s  %s  %s\n",
			r.c(ansiRed, "FAIL"),
			name,
			r.c(ansiDim, fmt.Sprintf("(%dms)", duration.Milliseconds())),
		)
		for _, f := range failures {
			if !f.Passed {
				fmt.Printf("      %s\n", f.Message)
			}
		}
	}
	r.hasOutput = true
}

// Skip prints a skipped test (no assertions).
func (r *Reporter) Skip(name string) {
	r.printGroup(name)
	switch r.mode {
	case ModePretty:
		fmt.Printf("%s%s  %s\n",
			r.prettyIndent(name),
			r.c(ansiYellow, "-"),
			r.c(ansiDim, r.prettyLabel(name)),
		)
	default:
		fmt.Printf("%s  %s\n", r.c(ansiYellow, "SKIP"), name)
	}
	r.hasOutput = true
}

// Error prints a test that errored during execution.
func (r *Reporter) Error(name string, err error) {
	r.printGroup(name)
	switch r.mode {
	case ModePretty:
		indent := r.prettyIndent(name)
		fmt.Printf("%s%s  %s\n",
			indent,
			r.c(ansiRed, "!"),
			r.prettyLabel(name),
		)
		fmt.Printf("%s%s\n", indent+"     ", r.c(ansiRed, err.Error()))
	default:
		fmt.Printf("%s  %s\n", r.c(ansiRed, "ERROR"), name)
		fmt.Printf("      %s\n", err.Error())
	}
	r.hasOutput = true
}

// Updated prints a test whose snapshot was updated.
func (r *Reporter) Updated(name string) {
	r.printGroup(name)
	switch r.mode {
	case ModePretty:
		fmt.Printf("%s%s  %s\n",
			r.prettyIndent(name),
			r.c(ansiYellow, "↺"),
			r.c(ansiDim, r.prettyLabel(name)+" (updated)"),
		)
	default:
		fmt.Printf("%s  %s\n", r.c(ansiYellow, "UPDT"), name)
	}
	r.hasOutput = true
}

// Details prints request/response info; only outputs in verbose mode.
func (r *Reporter) Details(d TestDetails) {
	if r.mode != ModeVerbose {
		return
	}
	fmt.Printf("  → %s %s\n", d.Method, d.URL)
	fmt.Printf("  ← %s  %s\n",
		r.c(ansiDim, fmt.Sprintf("%d", d.StatusCode)),
		r.c(ansiDim, fmt.Sprintf("%dms", d.Duration.Milliseconds())),
	)
	if d.Body != "" {
		body := strings.TrimSpace(d.Body)
		if len(body) > 300 {
			body = body[:297] + "…"
		}
		fmt.Printf("  %s\n", r.c(ansiDim, body))
	}
	fmt.Println()
}

// Summary prints the final test counts.
func (r *Reporter) Summary(passed, failed, skipped int) {
	total := passed + failed + skipped
	switch r.mode {
	case ModePretty:
		fmt.Println()
		fmt.Printf("  %s\n", r.c(ansiDim, strings.Repeat("─", 44)))
		var parts []string
		parts = append(parts, fmt.Sprintf("%d tests", total))
		if passed > 0 {
			parts = append(parts, r.c(ansiGreen, fmt.Sprintf("✓ %d passed", passed)))
		} else {
			parts = append(parts, r.c(ansiDim, "0 passed"))
		}
		if failed > 0 {
			parts = append(parts, r.c(ansiRed, fmt.Sprintf("✗ %d failed", failed)))
		} else {
			parts = append(parts, r.c(ansiDim, "0 failed"))
		}
		if skipped > 0 {
			parts = append(parts, r.c(ansiYellow, fmt.Sprintf("- %d skipped", skipped)))
		}
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	default:
		parts := []string{
			fmt.Sprintf("%d passed", passed),
			fmt.Sprintf("%d failed", failed),
		}
		if skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d skipped", skipped))
		}
		fmt.Printf("\n%d tests  %s\n", total, strings.Join(parts, "  "))
	}
}
