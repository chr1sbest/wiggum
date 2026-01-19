package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printUsage()
		os.Exit(0)
	}
	if os.Args[1] == "--version" {
		fmt.Println(versionLine())
		os.Exit(0)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(runCmd(os.Args[2:]))
	case "init", "new-project":
		newProjectCmd(os.Args[2:])
	case "add", "new-work":
		newWorkCmd(os.Args[2:])
	case "fix":
		fixCmd(os.Args[2:])
	case "pr":
		os.Exit(prCmd(os.Args[2:]))
	case "upgrade":
		os.Exit(upgradeCmd(os.Args[2:]))
	case "eval":
		os.Exit(evalCmd(os.Args[2:]))
	case "version":
		fmt.Println(versionLine())
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ralph 🖍️

"I'm a helper!"
— Ralph Wiggum

Usage:
  ralph <command> [flags]

Commands:
  run          Run the main loop (Ralph does the work)
  init         Start a new Ralph project (fresh crayons!)
  add          Add more work for Ralph to think about
  fix          Create tasks from a GitHub issue
  pr           Push branch and open a pull request
  eval         Run evaluation suites against ralph and oneshot approaches
  upgrade      Check for updates and upgrade Ralph
  version      Show Ralph's version number
  help         Show this message again (I like explaining)

Examples:
  # New project
  mkdir myproject && cd myproject
  ralph init requirements.md
  ralph run

  # Existing repo
  cd my-existing-repo
  ralph init
  ralph add "Add unit tests for the auth module"

Notes:
  - Ralph works on one thing at a time.
  - If nothing happens, that means it worked.

Run 'ralph <command> -h' for details.`)
}
