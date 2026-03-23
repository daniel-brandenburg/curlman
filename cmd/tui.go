package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielbrandenburg/curlman/internal/discovery"
	"github.com/danielbrandenburg/curlman/internal/env"
	"github.com/danielbrandenburg/curlman/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	var envFile string
	var testPath string

	cmd := &cobra.Command{
		Use:   "tui [test-path]",
		Short: "Interactive TUI for running HTTP tests",
		Long: `Launch an interactive TUI that runs HTTP tests and shows live results.

test-path is an optional filter: a stem name ("create"), path prefix
("users/"), or shell glob. Omit to run all tests.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := filterFromArgs(args)

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

			m := tui.New(pairs, vars)
			p := tea.NewProgram(m,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)
			_, err = p.Run()
			return err
		},
	}

	cmd.Flags().StringVar(&envFile, "env", ".env", "path to .env file")
	cmd.Flags().StringVar(&testPath, "path", "", "directory to search for tests (default: current directory)")
	return cmd
}
