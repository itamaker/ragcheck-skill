package app

import (
	"fmt"
	"os"
	"runtime/debug"
)

// Version is the release version. main sets it (GoReleaser injects
// main.version); "dev" falls back to the module version from build info.
var Version = "dev"

func Run(args []string) int {
	if len(args) == 0 {
		return runTUI()
	}

	// "help" and "version" directly after a subcommand behave like the flags.
	// Only the first argument is checked, so a flag value such as
	// "-qrels help" is never hijacked.
	if len(args) > 1 && (args[0] == "score" || args[0] == "judge" || args[0] == "tui" || args[0] == "interactive") {
		switch {
		case args[1] == "help":
			return runHelp([]string{args[0]})
		case isVersionArg(args[1]):
			return runVersion()
		}
	}

	switch args[0] {
	case "score":
		return runScore(args[1:])
	case "judge":
		return runJudge(args[1:])
	case "tui", "interactive":
		if len(args) > 1 && isHelpArg(args[1]) {
			return runHelp([]string{"tui"})
		}
		return runTUI()
	case "help", "-h", "-help", "--help":
		return runHelp(args[1:])
	case "version", "-version", "--version":
		return runVersion()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usageText)
		return 2
	}
}

// runHelp prints top-level usage, or the usage of the named subcommand.
func runHelp(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stdout, usageText)
		return 0
	}
	switch args[0] {
	case "score":
		return runScore([]string{"-h"})
	case "judge":
		return runJudge([]string{"-h"})
	case "tui", "interactive":
		fmt.Fprint(os.Stdout, tuiUsageText)
		return 0
	case "help", "version", "-h", "-help", "--help":
		fmt.Fprint(os.Stdout, usageText)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usageText)
		return 2
	}
}

func runVersion() int {
	fmt.Fprintln(os.Stdout, "ragcheck "+versionString())
	return 0
}

func isVersionArg(arg string) bool {
	switch arg {
	case "version", "-version", "--version":
		return true
	}
	return false
}

func isHelpArg(arg string) bool {
	switch arg {
	case "-h", "-help", "--help":
		return true
	}
	return false
}

func versionString() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

const usageText = `ragcheck scores retrieval runs (Precision, Recall, HitRate, MRR, MAP, nDCG @k)
and judges RAG answers offline.

Usage:
  ragcheck score -qrels FILE -run FILE [-k N] [-json]
  ragcheck judge -input FILE [-json]
  ragcheck help [score|judge|tui]
  ragcheck version

Examples:
  ragcheck score -qrels examples/qrels.json -run examples/run.json -k 3
  ragcheck judge -input examples/judge.json -json

Notes:
  Running ragcheck with no arguments (or "tui") launches an interactive
  terminal UI. Scripts and coding agents must not do that: use score and judge.

Exit codes: 0 success, 1 failure, 2 usage error.
`

const tuiUsageText = `Usage: ragcheck [tui]

Launches the interactive terminal UI. It needs a real terminal; scripts and
coding agents should use "ragcheck score" and "ragcheck judge" instead.
`
