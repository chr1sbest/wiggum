package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

	fmt.Printf("eval subcommand '%s' not yet implemented\n", fs.Arg(0))
	return 1
}
