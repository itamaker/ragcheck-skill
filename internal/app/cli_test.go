package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests swap os.Stdout/os.Stderr, so none of them call t.Parallel.

const (
	exampleQrels = "../../examples/qrels.json"
	exampleRun   = "../../examples/run.json"
	exampleJudge = "../../examples/judge.json"
)

func runCapturing(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	code = Run(args)

	os.Stdout, os.Stderr = oldOut, oldErr
	_ = outW.Close()
	_ = errW.Close()
	read := func(f *os.File) string {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		_ = f.Close()
		return sb.String()
	}
	return code, read(outR), read(errR)
}

func TestHelpAndVersionExitZero(t *testing.T) {
	cases := [][]string{
		{"-h"}, {"-help"}, {"--help"}, {"help"}, {"help", "-h"}, {"help", "--help"},
		{"help", "score"}, {"help", "judge"}, {"help", "tui"}, {"help", "interactive"},
		{"score", "-h"}, {"score", "-help"}, {"score", "--help"}, {"score", "help"},
		{"judge", "-h"}, {"judge", "-help"}, {"judge", "--help"}, {"judge", "help"},
		{"tui", "-h"}, {"tui", "--help"}, {"tui", "help"}, {"interactive", "--help"}, {"interactive", "help"},
		{"version"}, {"-version"}, {"--version"},
		{"score", "version"}, {"score", "-version"}, {"score", "--version"},
		{"judge", "version"}, {"judge", "-version"}, {"judge", "--version"},
		{"tui", "version"}, {"tui", "--version"}, {"interactive", "version"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, stdout, stderr := runCapturing(t, args...)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr %q)", code, stderr)
			}
			if strings.TrimSpace(stdout) == "" {
				t.Errorf("expected output on stdout")
			}
			if stderr != "" {
				t.Errorf("unexpected stderr %q", stderr)
			}
		})
	}
}

func TestHelpContent(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"--help"}, []string{"ragcheck score", "ragcheck judge", "must not"}},
		{[]string{"score", "-h"}, []string{"Usage: ragcheck score", "-qrels", "-run", "-k", "-json"}},
		{[]string{"judge", "--help"}, []string{"Usage: ragcheck judge", "-input", "-json"}},
		{[]string{"tui", "-h"}, []string{"Usage: ragcheck", "terminal UI"}},
		{[]string{"help", "score"}, []string{"Usage: ragcheck score", "-qrels"}},
		{[]string{"version"}, []string{"ragcheck "}},
	}
	for _, tc := range cases {
		_, stdout, _ := runCapturing(t, tc.args...)
		for _, want := range tc.want {
			if !strings.Contains(stdout, want) {
				t.Errorf("%v: output is missing %q:\n%s", tc.args, want, stdout)
			}
		}
	}
}

func TestSubcommandHelpWordMatchesFlag(t *testing.T) {
	for _, name := range []string{"score", "judge", "tui"} {
		_, viaFlag, _ := runCapturing(t, name, "-h")
		_, viaWord, _ := runCapturing(t, name, "help")
		_, viaTop, _ := runCapturing(t, "help", name)
		if viaFlag == "" || viaFlag != viaWord || viaFlag != viaTop {
			t.Errorf("%s help differs between -h, help and help %s:\n%q\n%q\n%q", name, name, viaFlag, viaWord, viaTop)
		}
	}
}

func TestHelpAfterHelpPrintsTopLevelUsage(t *testing.T) {
	_, top, _ := runCapturing(t, "help")
	for _, args := range [][]string{{"help", "--help"}, {"help", "-h"}, {"help", "version"}} {
		code, out, _ := runCapturing(t, args...)
		if code != 0 || out != top {
			t.Errorf("%v: code %d, output differs from top-level help", args, code)
		}
	}
}

func TestVersionOutput(t *testing.T) {
	old := Version
	defer func() { Version = old }()

	Version = "1.2.3"
	for _, args := range [][]string{{"version"}, {"--version"}, {"-version"}, {"score", "--version"}, {"judge", "version"}} {
		if _, stdout, _ := runCapturing(t, args...); strings.TrimSpace(stdout) != "ragcheck 1.2.3" {
			t.Errorf("%v: version output = %q", args, stdout)
		}
	}

	// Without an injected version the output still names the program.
	Version = "dev"
	if _, stdout, _ := runCapturing(t, "--version"); !strings.HasPrefix(stdout, "ragcheck ") {
		t.Errorf("dev version output = %q", stdout)
	}
}

func TestFlagValueNamedHelpOrVersionIsNotHijacked(t *testing.T) {
	cases := [][]string{
		{"score", "-qrels", "help", "-run", exampleRun},
		{"score", "-qrels", "version", "-run", exampleRun},
		{"score", "-qrels", exampleQrels, "-run", "help"},
		{"score", "-qrels", exampleQrels, "-run", "version"},
		{"judge", "-input", "help"},
		{"judge", "-input", "version"},
		{"judge", "-input", "--version"},
	}
	for _, args := range cases {
		code, stdout, stderr := runCapturing(t, args...)
		// The value is treated as a path that does not exist: a failure (1),
		// never a help or version screen.
		if code != 1 {
			t.Errorf("%v: exit code = %d, want 1", args, code)
		}
		if stdout != "" || stderr == "" {
			t.Errorf("%v: stdout %q stderr %q, want an error on stderr only", args, stdout, stderr)
		}
	}
}

func TestRealErrorsKeepExitCodes(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	cases := []struct {
		args []string
		want int
	}{
		{[]string{"bogus"}, 2},
		{[]string{"--bogus"}, 2},
		{[]string{"help", "bogus"}, 2},
		{[]string{"score"}, 2},
		{[]string{"score", "-qrels", exampleQrels}, 2},
		{[]string{"score", "-bogus"}, 2},
		{[]string{"score", "-k", "notanumber"}, 2},
		{[]string{"score", "-qrels", missing, "-run", exampleRun}, 1},
		{[]string{"score", "-qrels", exampleQrels, "-run", missing}, 1},
		{[]string{"judge"}, 2},
		{[]string{"judge", "-bogus"}, 2},
		{[]string{"judge", "-input", missing}, 1},
	}
	for _, tc := range cases {
		code, stdout, stderr := runCapturing(t, tc.args...)
		if code != tc.want {
			t.Errorf("%v: exit code = %d, want %d", tc.args, code, tc.want)
		}
		if stdout != "" {
			t.Errorf("%v: errors should not print to stdout, got %q", tc.args, stdout)
		}
		if stderr == "" {
			t.Errorf("%v: expected an error message on stderr", tc.args)
		}
	}
}

func TestUnknownSubcommandPrintsUsageToStderr(t *testing.T) {
	code, stdout, stderr := runCapturing(t, "bogus")
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{`unknown subcommand "bogus"`, "Usage:", "ragcheck score"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, stderr)
		}
	}
}

func TestExampleCommandsStillWork(t *testing.T) {
	code, stdout, stderr := runCapturing(t, "score", "-qrels", exampleQrels, "-run", exampleRun, "-k", "3")
	if code != 0 || !strings.Contains(stdout, "Queries: ") || stderr != "" {
		t.Errorf("score: code %d stdout %q stderr %q", code, stdout, stderr)
	}
	code, stdout, stderr = runCapturing(t, "judge", "-input", exampleJudge)
	if code != 0 || !strings.Contains(stdout, "Groundedness: ") || stderr != "" {
		t.Errorf("judge: code %d stdout %q stderr %q", code, stdout, stderr)
	}
}
