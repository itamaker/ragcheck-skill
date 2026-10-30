package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// parseFlags parses args into fs. Help requests (-h, -help, --help) print the
// flag usage to stdout and report help=true; parse errors are written to
// stderr. Callers return 0 for help and 2 for any other error.
func parseFlags(fs *flag.FlagSet, usageLine string, args []string) (help bool, err error) {
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage = func() {
		fmt.Fprintf(&buf, "Usage: %s\n\nFlags:\n", usageLine)
		fs.PrintDefaults()
	}
	err = fs.Parse(args)
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, flag.ErrHelp):
		_, _ = buf.WriteTo(os.Stdout)
		return true, nil
	default:
		_, _ = buf.WriteTo(os.Stderr)
		return false, err
	}
}

func runScore(args []string) int {
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	qrelsPath := fs.String("qrels", "", "path to qrels JSON")
	runPath := fs.String("run", "", "path to retrieval run JSON")
	k := fs.Int("k", 5, "top-k cutoff")
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")

	if help, err := parseFlags(fs, "ragcheck score -qrels FILE -run FILE [-k N] [-json]", args); err != nil {
		return 2
	} else if help {
		return 0
	}
	if *qrelsPath == "" || *runPath == "" {
		fmt.Fprintln(os.Stderr, "both -qrels and -run are required")
		return 2
	}

	qrels, err := loadQrels(*qrelsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	runResults, err := loadRun(*runPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	metrics := score(qrels, runResults, *k)
	if *jsonOutput {
		body, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(body))
		return 0
	}

	fmt.Printf("Queries: %d\n", metrics.Queries)
	fmt.Printf("Missing runs: %d\n", metrics.MissingRuns)
	fmt.Printf("Precision@%d: %.3f\n", metrics.K, metrics.PrecisionAtK)
	fmt.Printf("Recall@%d: %.3f\n", metrics.K, metrics.RecallAtK)
	fmt.Printf("HitRate@%d: %.3f\n", metrics.K, metrics.HitRateAtK)
	fmt.Printf("MRR@%d: %.3f\n", metrics.K, metrics.MRRAtK)
	fmt.Printf("MAP@%d: %.3f\n", metrics.K, metrics.MAPAtK)
	fmt.Printf("nDCG@%d: %.3f\n", metrics.K, metrics.NDCGAtK)
	return 0
}

func runJudge(args []string) int {
	fs := flag.NewFlagSet("judge", flag.ContinueOnError)
	inputPath := fs.String("input", "", "path to a judge input JSON file")
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")

	if help, err := parseFlags(fs, "ragcheck judge -input FILE [-json]", args); err != nil {
		return 2
	} else if help {
		return 0
	}
	if *inputPath == "" {
		fmt.Fprintln(os.Stderr, "-input is required")
		return 2
	}

	records, err := loadJudgeRecords(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	report := judge(records)
	if *jsonOutput {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(body))
		return 0
	}

	printJudgeReport(report)
	return 0
}

func printJudgeReport(report JudgeReport) {
	fmt.Printf("Queries: %d\n", report.Queries)
	fmt.Printf("Answer relevance: %.3f\n", report.AnswerRelevance)
	fmt.Printf("Context relevance: %.3f\n", report.ContextRelevance)
	fmt.Printf("Groundedness: %.3f\n", report.Groundedness)
	fmt.Printf("Reference coverage: %.3f\n", report.ReferenceCoverage)
	for _, result := range report.Results {
		fmt.Printf("- %s: relevance=%.3f groundedness=%.3f\n", result.QueryID, result.AnswerRelevance, result.Groundedness)
		if len(result.UnsupportedTokens) > 0 {
			fmt.Printf("  unsupported: %s\n", strings.Join(result.UnsupportedTokens, ", "))
		}
	}
}
