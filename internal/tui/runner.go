package tui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielbrandenburg/curlman/internal/asserter"
	"github.com/danielbrandenburg/curlman/internal/discovery"
	"github.com/danielbrandenburg/curlman/internal/env"
	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
)

// sem limits the number of concurrent HTTP requests from the TUI runner.
var sem = make(chan struct{}, 5)

// runTestCmd returns a tea.Cmd that executes the test for pair and sends back
// a testResultMsg when complete. Parallelism is capped by sem.
func runTestCmd(index int, pair discovery.TestPair, vars map[string]string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		// Parse .http file
		httpContent, err := os.ReadFile(pair.HTTPFile)
		if err != nil {
			return testResultMsg{index: index, status: StatusError, err: fmt.Errorf("reading %s: %w", pair.HTTPFile, err)}
		}
		substituted := env.Substitute(string(httpContent), vars)

		requests, err := parser.ParseHTTP(substituted)
		if err != nil {
			return testResultMsg{index: index, status: StatusError, err: fmt.Errorf("parsing %s: %w", pair.HTTPFile, err)}
		}
		if len(requests) == 0 {
			return testResultMsg{index: index, status: StatusSkip}
		}
		req := requests[0]

		// Parse .assert file
		var block parser.AssertionBlock
		hasAssert := false
		if pair.AssertFile != "" {
			assertContent, err := os.ReadFile(pair.AssertFile)
			if err != nil {
				return testResultMsg{index: index, status: StatusError, err: fmt.Errorf("reading %s: %w", pair.AssertFile, err), request: &req}
			}
			blocks, err := parser.ParseAssert(string(assertContent))
			if err != nil {
				return testResultMsg{index: index, status: StatusError, err: fmt.Errorf("parsing %s: %w", pair.AssertFile, err), request: &req}
			}
			for _, b := range blocks {
				if b.Name == req.Name {
					block = b
					hasAssert = true
					break
				}
			}
		}

		if !hasAssert {
			return testResultMsg{index: index, status: StatusSkip, lastRun: time.Now(), request: &req}
		}

		// Execute request
		ranAt := time.Now()
		resp, err := runner.Run(req)
		if err != nil {
			return testResultMsg{index: index, status: StatusError, lastRun: ranAt, err: err, request: &req}
		}

		// Evaluate assertions
		results := asserter.Evaluate(block, resp)
		allPassed := true
		for _, r := range results {
			if !r.Passed {
				allPassed = false
				break
			}
		}

		status := StatusPass
		if !allPassed {
			status = StatusFail
		}

		return testResultMsg{
			index:      index,
			status:     status,
			lastRun:    ranAt,
			request:    &req,
			response:   &resp,
			assertions: block.Assertions,
			results:    results,
		}
	}
}
