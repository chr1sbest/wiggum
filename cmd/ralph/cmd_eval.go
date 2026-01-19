package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chr1sbest/wiggum/internal/eval"
)

func evalCmd(args []string) int {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`eval 🧪  Run evaluation suites against ralph and oneshot approaches

Usage:
  ralph eval <subcommand> [flags]

Subcommands:
  list         List available evaluation suites
  run          Run an evaluation suite
  compare      Compare ralph vs oneshot results

Examples:
  ralph eval list
  ralph eval run flask --approach ralph
  ralph eval compare flask

Run 'ralph eval <subcommand> -h' for details.
`)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return 0
		}
		fmt.Println(err)
		fs.Usage()
		return 1
	}

	// Show usage if no subcommand provided
	if fs.NArg() == 0 {
		fs.Usage()
		return 0
	}

	subcommand := fs.Arg(0)
	subArgs := fs.Args()[1:]

	switch subcommand {
	case "list":
		return evalListCmd(subArgs)
	case "run":
		return evalRunCmd(subArgs)
	case "compare":
		return evalCompareCmd(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown eval subcommand: %s\n", subcommand)
		fs.Usage()
		return 1
	}
}

func evalListCmd(args []string) int {
	fs := flag.NewFlagSet("eval list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`eval list 📋  List available evaluation suites

Usage:
  ralph eval list

Description:
  Discovers and displays all evaluation suites by scanning for suite.yaml files
  in the evals/suites/ directory.
`)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return 0
		}
		fmt.Println(err)
		return 1
	}

	suitesDir := "evals/suites"
	entries, err := os.ReadDir(suitesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read suites directory: %v\n", err)
		return 1
	}

	var suites []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		suiteYaml := filepath.Join(suitesDir, entry.Name(), "suite.yaml")
		if _, err := os.Stat(suiteYaml); err == nil {
			suites = append(suites, entry.Name())
		}
	}

	if len(suites) == 0 {
		fmt.Println("No evaluation suites found.")
		return 0
	}

	fmt.Println("Available evaluation suites:")
	fmt.Println()
	for _, suite := range suites {
		config, err := eval.LoadSuite(suite)
		if err != nil {
			fmt.Printf("  %s (error loading: %v)\n", suite, err)
			continue
		}
		fmt.Printf("  %s - %s\n", config.Name, config.Description)
	}

	return 0
}

func evalRunCmd(args []string) int {
	fs := flag.NewFlagSet("eval run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	approach := fs.String("approach", "ralph", "Evaluation approach (ralph or oneshot)")
	model := fs.String("model", "sonnet", "Claude model to use")
	testOnly := fs.String("test-only", "", "Run tests only against existing project directory")

	fs.Usage = func() {
		fmt.Print(`eval run 🏃  Run an evaluation suite

Usage:
  ralph eval run <suite> [flags]

Flags:
  --approach string    Evaluation approach: ralph or oneshot (default "ralph")
  --model string       Claude model to use (default "sonnet")
  --test-only string   Run tests only against existing project directory

Examples:
  ralph eval run flask --approach ralph
  ralph eval run logagg --approach oneshot --model opus
  ralph eval run flask --test-only /path/to/existing/project
`)
	}

	// Reorder args: move flags before positional args so flag.Parse works correctly
	// Go's flag package stops at first non-flag argument
	var reordered []string
	var positional []string
	for i := 0; i < len(args); i++ {
		if len(args[i]) > 0 && args[i][0] == '-' {
			reordered = append(reordered, args[i])
			// If it's a flag that takes a value, grab the next arg too
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				// Check if it's a known flag that takes a value
				if args[i] == "-approach" || args[i] == "--approach" ||
					args[i] == "-model" || args[i] == "--model" ||
					args[i] == "-test-only" || args[i] == "--test-only" {
					i++
					reordered = append(reordered, args[i])
				}
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	reordered = append(reordered, positional...)

	if err := fs.Parse(reordered); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return 0
		}
		fmt.Println(err)
		fs.Usage()
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: suite name required")
		fs.Usage()
		return 1
	}

	suite := fs.Arg(0)

	// Validate suite exists
	suiteYaml := filepath.Join("evals", "suites", suite, "suite.yaml")
	if _, err := os.Stat(suiteYaml); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Suite '%s' not found. Run 'ralph eval list' to see available suites.\n", suite)
		return 1
	}

	// Handle test-only mode
	if *testOnly != "" {
		// Just run tests against existing project
		suiteConfig, err := eval.LoadSuite(suite)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load suite: %v\n", err)
			return 1
		}

		result, err := eval.RunSharedTests(*testOnly, suiteConfig, 8000)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
			return 1
		}

		fmt.Printf("\nTests: %d/%d passed\n", result.Passed, result.Total)
		return 0
	}

	// Validate approach
	if *approach != "ralph" && *approach != "oneshot" {
		fmt.Fprintf(os.Stderr, "Invalid approach '%s'. Must be 'ralph' or 'oneshot'.\n", *approach)
		return 1
	}

	// Create config and run evaluation using Go implementation
	config := eval.NewRunConfig(suite, *approach, *model)

	_, err := eval.Run(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run evaluation: %v\n", err)
		return 1
	}

	return 0
}

func evalCompareCmd(args []string) int {
	fs := flag.NewFlagSet("eval compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	model := fs.String("model", "sonnet", "Claude model to compare (default \"sonnet\")")

	fs.Usage = func() {
		fmt.Print(`eval compare 📊  Compare ralph vs oneshot results

Usage:
  ralph eval compare <suite> [flags]

Flags:
  --model string       Claude model to compare (default "sonnet")

Description:
  Compares the most recent ralph and oneshot evaluation results for the
  specified suite, displaying metrics side-by-side.

Examples:
  ralph eval compare flask
  ralph eval compare logagg --model opus
`)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return 0
		}
		fmt.Println(err)
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: suite name required")
		fs.Usage()
		return 1
	}

	suite := fs.Arg(0)

	// Validate suite exists
	suiteYaml := filepath.Join("evals", "suites", suite, "suite.yaml")
	if _, err := os.Stat(suiteYaml); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Suite '%s' not found. Run 'ralph eval list' to see available suites.\n", suite)
		return 1
	}

	// Check if results exist
	resultsDir := "evals/results"
	if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "No results directory found. Run evaluations first with 'ralph eval run'.\n")
		return 1
	}

	// Use Go implementation to compare results
	if err := eval.Compare(suite, *model); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compare evaluations: %v\n", err)
		return 1
	}

	return 0
}
