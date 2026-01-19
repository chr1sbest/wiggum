package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	fs.Usage = func() {
		fmt.Print(`eval run 🏃  Run an evaluation suite

Usage:
  ralph eval run <suite> [flags]

Flags:
  --approach string    Evaluation approach: ralph or oneshot (default "ralph")
  --model string       Claude model to use (default "sonnet")

Examples:
  ralph eval run flask --approach ralph
  ralph eval run logagg --approach oneshot --model opus
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
					args[i] == "-model" || args[i] == "--model" {
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

	// Validate approach
	if *approach != "ralph" && *approach != "oneshot" {
		fmt.Fprintf(os.Stderr, "Invalid approach '%s'. Must be 'ralph' or 'oneshot'.\n", *approach)
		return 1
	}

	// Call evals/run.sh with the suite name and flags
	script := "./evals/run.sh"
	cmdArgs := []string{suite, *approach}
	if *model != "" {
		cmdArgs = append(cmdArgs, *model)
	}

	cmd := exec.Command(script, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Failed to run evaluation: %v\n", err)
		return 1
	}

	return 0
}

func evalCompareCmd(args []string) int {
	fs := flag.NewFlagSet("eval compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Print(`eval compare 📊  Compare ralph vs oneshot results

Usage:
  ralph eval compare <suite>

Description:
  Compares the most recent ralph and oneshot evaluation results for the
  specified suite, displaying metrics side-by-side.

Examples:
  ralph eval compare flask
  ralph eval compare logagg
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

	// Call evals/compare_evals.sh with the suite name
	script := "./evals/compare_evals.sh"
	cmd := exec.Command(script, suite)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// Check if it's a "no results found" error by examining stderr
		// For now, just return the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Failed to compare evaluations: %v\n", err)
		return 1
	}

	return 0
}
