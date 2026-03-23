package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danielbrandenburg/curlman/internal/asserter"
)

var (
	colorGreen  string
	colorRed    string
	colorYellow string
	colorReset  string
)

func init() {
	if shouldUseColor() {
		colorGreen  = "\033[32m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorReset  = "\033[0m"
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

// Pass prints a passing test line.
func Pass(name string, duration time.Duration) {
	fmt.Printf("%s✓%s  %s (%dms)\n", colorGreen, colorReset, name, duration.Milliseconds())
}

// Fail prints a failing test line with failure details.
func Fail(name string, duration time.Duration, failures []asserter.Result) {
	fmt.Printf("%s✗%s  %s (%dms)\n", colorRed, colorReset, name, duration.Milliseconds())
	for _, f := range failures {
		if !f.Passed {
			fmt.Printf("      %s\n", f.Message)
		}
	}
}

// Skip prints a skipped test line.
func Skip(name string) {
	fmt.Printf("%s-%s  %s (no assertions)\n", colorYellow, colorReset, name)
}

// Error prints a test that errored during execution.
func Error(name string, err error) {
	fmt.Printf("%s✗%s  %s\n", colorRed, colorReset, name)
	fmt.Printf("      ERROR: %v\n", err)
}

// Updated prints a test whose snapshot was updated.
func Updated(name string) {
	fmt.Printf("%s↺%s  %s (updated)\n", colorYellow, colorReset, name)
}

// Summary prints the final results summary.
func Summary(passed, failed, skipped int) {
	parts := []string{
		fmt.Sprintf("%d passed", passed),
		fmt.Sprintf("%d failed", failed),
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	total := passed + failed
	fmt.Printf("\nResults: %d tests run, %s\n", total, strings.Join(parts, ", "))
}
