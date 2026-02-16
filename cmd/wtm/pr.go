package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr <number> [directory]",
	Short: "Checkout a pull request (alias for 'add pr/<number>')",
	Long: `Checkout a pull request by number. This is a convenience alias for 'gh wt add pr/<number>'.

Examples:
  gh wt pr 123           # Checkout PR #123 to pr-123/
  gh wt pr 123 my-dir    # Checkout PR #123 to my-dir/`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runPr,
}

func runPr(cmd *cobra.Command, args []string) error {
	// Parse PR number
	prNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number: %s", args[0])
	}

	// Validate PR number
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be positive")
	}

	// Build add command arguments
	addArgs := []string{fmt.Sprintf("pr/%d", prNumber)}
	if len(args) > 1 {
		addArgs = append(addArgs, args[1])
	}

	// Call add command
	return runAdd(cmd, addArgs)
}
