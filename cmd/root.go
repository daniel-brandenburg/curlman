package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielbrandenburg/curlman/internal/asserter"
	"github.com/danielbrandenburg/curlman/internal/discovery"
	"github.com/danielbrandenburg/curlman/internal/env"
	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/reporter"
	"github.com/danielbrandenburg/curlman/internal/runner"
	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	var envFile string
	var testPath string
	var update bool
	var failFast bool
	var verbose bool

	root := &cobra.Command{
		Use:   "curlman [test-path]",
		Short: "Run HTTP tests defined as .http files",
		Long: `Run HTTP tests found in the current directory (or --path).

test-path is an optional filter: a stem name ("create"), path prefix
("users/"), or shell glob ("tests/*" — the common directory is used).
Omit to run all tests.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(envFile, testPath, filterFromArgs(args), update, failFast, verbose)
		},
	}

	root.Flags().StringVar(&envFile, "env", ".env", "path to .env file")
	root.Flags().StringVar(&testPath, "path", "", "directory to search for tests (default: current directory)")
	root.Flags().BoolVar(&update, "update", false, "overwrite .assert files with actual responses")
	root.Flags().BoolVar(&failFast, "fail-fast", false, "stop after first failure")
	root.Flags().BoolVarP(&verbose, "verbose", "v", false, "print full request and response for every test")

	root.AddCommand(newRunCmd())
	return root
}

func newRunCmd() *cobra.Command {
	var envFile string
	var testPath string
	var update bool
	var failFast bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "run [test-path]",
		Short: "Run HTTP tests",
		Long: `Run HTTP tests found in the current directory (or --path).

test-path is an optional filter: a stem name ("create"), path prefix
("users/"), or shell glob ("tests/*" — the common directory is used).
Omit to run all tests.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(envFile, testPath, filterFromArgs(args), update, failFast, verbose)
		},
	}

	cmd.Flags().StringVar(&envFile, "env", ".env", "path to .env file")
	cmd.Flags().StringVar(&testPath, "path", "", "directory to search for tests (default: current directory)")
	cmd.Flags().BoolVar(&update, "update", false, "overwrite .assert files with actual responses")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "stop after first failure")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print full request and response for every test")

	return cmd
}

// filterFromArgs derives a test filter from CLI args.
//
// Single arg: used as-is (stem name or path prefix).
// Multiple args (shell glob expansion): strips known extensions, finds the
// longest common directory prefix, and uses that as the filter.
func filterFromArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return args[0]
	}

	// Multiple args: likely shell glob expansion. Collect parent dirs.
	dirs := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSuffix(a, ".http")
		a = strings.TrimSuffix(a, ".assert")
		d := filepath.ToSlash(filepath.Dir(a))
		if d != "." && d != "" {
			dirs = append(dirs, d)
		}
	}
	if len(dirs) == 0 {
		return args[0]
	}

	// Longest common prefix of all dirs
	common := dirs[0]
	for _, d := range dirs[1:] {
		common = commonDirPrefix(common, d)
	}
	return common
}

// commonDirPrefix returns the longest common path prefix of two slash-separated paths.
func commonDirPrefix(a, b string) string {
	partsA := strings.Split(a, "/")
	partsB := strings.Split(b, "/")
	var common []string
	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		if partsA[i] != partsB[i] {
			break
		}
		common = append(common, partsA[i])
	}
	if len(common) == 0 {
		return ""
	}
	return strings.Join(common, "/")
}

func runTests(envFile, testPath, filter string, update, failFast, verbose bool) error {
	vars, err := env.Load(envFile)
	if err != nil {
		return fmt.Errorf("loading env: %w", err)
	}

	root := testPath
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	} else {
		root, err = filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("resolving path %q: %w", testPath, err)
		}
	}

	pairs, err := discovery.Discover(root, filter)
	if err != nil {
		return fmt.Errorf("discovering tests: %w", err)
	}

	if len(pairs) == 0 {
		fmt.Println("No test files found.")
		return nil
	}

	passed := 0
	failed := 0
	skipped := 0
	hasFailures := false

	for _, pair := range pairs {
		httpContent, err := os.ReadFile(pair.HTTPFile)
		if err != nil {
			return fmt.Errorf("reading %s: %w", pair.HTTPFile, err)
		}
		substituted := env.Substitute(string(httpContent), vars)

		requests, err := parser.ParseHTTP(substituted)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", pair.HTTPFile, err)
		}

		assertBlocks := map[string]parser.AssertionBlock{}
		if pair.AssertFile != "" {
			assertContent, err := os.ReadFile(pair.AssertFile)
			if err != nil {
				return fmt.Errorf("reading %s: %w", pair.AssertFile, err)
			}
			blocks, err := parser.ParseAssert(string(assertContent))
			if err != nil {
				return fmt.Errorf("parsing %s: %w", pair.AssertFile, err)
			}
			for _, b := range blocks {
				assertBlocks[b.Name] = b
			}
		}

		for _, req := range requests {
			testLabel := pair.Name
			if req.Name != "" {
				testLabel = fmt.Sprintf("%s :: %s", pair.Name, req.Name)
			}

			// --update: run and write snapshot, skip comparison
			if update {
				resp, err := runner.Run(req)
				if err != nil {
					reporter.Error(testLabel, err)
					failed++
					hasFailures = true
					if failFast {
						break
					}
					continue
				}
				if verbose {
					printVerbose(req, resp)
				}
				if err := writeSnapshot(pair, req, resp); err != nil {
					return fmt.Errorf("writing snapshot for %s: %w", testLabel, err)
				}
				reporter.Updated(testLabel)
				continue
			}

			// No assert block: skip
			block, hasAssert := assertBlocks[req.Name]
			if !hasAssert {
				reporter.Skip(testLabel)
				skipped++
				continue
			}

			start := time.Now()
			resp, err := runner.Run(req)
			duration := time.Since(start)
			if err != nil {
				reporter.Error(testLabel, err)
				failed++
				hasFailures = true
				if failFast {
					break
				}
				continue
			}

			if verbose {
				printVerbose(req, resp)
			}

			results := asserter.Evaluate(block, resp)
			allPassed := true
			var failResults []asserter.Result
			for _, r := range results {
				if !r.Passed {
					allPassed = false
					failResults = append(failResults, r)
				}
			}

			if allPassed {
				reporter.Pass(testLabel, duration)
				passed++
			} else {
				reporter.Fail(testLabel, duration, failResults)
				failed++
				hasFailures = true
				if failFast {
					break
				}
			}
		}

		if failFast && hasFailures {
			break
		}
	}

	reporter.Summary(passed, failed, skipped)

	if hasFailures {
		os.Exit(1)
	}
	return nil
}

// writeSnapshot serializes the actual response into .assert format and writes
// it to the pair's assert file, creating it if necessary.
func writeSnapshot(pair discovery.TestPair, req parser.Request, resp runner.Response) error {
	assertFile := pair.AssertFile
	if assertFile == "" {
		// Derive assert path from http file
		assertFile = strings.TrimSuffix(pair.HTTPFile, ".http") + ".assert"
	}

	var sb strings.Builder

	blockName := req.Name
	if blockName != "" {
		sb.WriteString("### ")
		sb.WriteString(blockName)
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "STATUS %d\n", resp.StatusCode)

	// Write a few key headers
	for _, key := range []string{"content-type", "content-length"} {
		if val, ok := resp.Headers[key]; ok {
			fmt.Fprintf(&sb, "HEADER %s: %s\n", key, val)
		}
	}

	if resp.Body != "" {
		fmt.Fprintf(&sb, "BODY $ exists\n")
	}

	return os.WriteFile(assertFile, []byte(sb.String()), 0644)
}

func printVerbose(req parser.Request, resp runner.Response) {
	fmt.Printf("  → %s %s\n", req.Method, req.URL)
	fmt.Printf("  ← %d\n", resp.StatusCode)
	if resp.Body != "" {
		fmt.Printf("  %s\n", resp.Body)
	}
}
