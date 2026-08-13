package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "harboring-test-*")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(tmpDir, "harboring")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func harboring(args ...string) (string, string, error) {
	cmd := exec.Command(binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestValidate(t *testing.T) {
	out, _, err := harboring("validate", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "VALID") {
		t.Fatalf("expected VALID, got: %s", out)
	}
	if !strings.Contains(out, "21 entries") {
		t.Fatalf("expected 21 entries, got: %s", out)
	}
}

func TestValidateMissingFile(t *testing.T) {
	_, stderr, err := harboring("validate", "nonexistent.har")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("expected error message, got: %s", stderr)
	}
}

func TestValidateNoArgs(t *testing.T) {
	_, stderr, err := harboring("validate")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "no HAR file") {
		t.Fatalf("expected 'no HAR file', got: %s", stderr)
	}
}

func TestStats(t *testing.T) {
	out, _, err := harboring("stats", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Requests:") {
		t.Fatalf("expected Requests: in output, got: %s", out)
	}
	if !strings.Contains(out, "Total size:") {
		t.Fatalf("expected Total size: in output, got: %s", out)
	}
	if !strings.Contains(out, "Methods:") {
		t.Fatalf("expected Methods: in output, got: %s", out)
	}
	if !strings.Contains(out, "Status codes:") {
		t.Fatalf("expected Status codes: in output, got: %s", out)
	}
	if !strings.Contains(out, "GET") {
		t.Fatalf("expected GET method, got: %s", out)
	}
}

func TestShrinkStdout(t *testing.T) {
	orig, err := os.ReadFile("../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := harboring("shrink", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}

	// Remove trailing newline added by fmt.Println
	got := strings.TrimSuffix(out, "\n")

	if !bytes.Equal(orig, []byte(got)) {
		t.Fatal("shrink output differs from original file (byte-identical expected)")
	}
}

func TestExpandShrinkRoundTrip(t *testing.T) {
	orig, err := os.ReadFile("../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}

	// expand
	expanded, _, err := harboring("expand", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}

	// expand output must be valid JSON
	if !json.Valid([]byte(expanded)) {
		t.Fatal("expand output is not valid JSON")
	}

	// must have newlines (pretty-printed)
	if !strings.Contains(expanded, "\n") {
		t.Fatal("expand output should be pretty-printed with newlines")
	}

	// shrink the expanded output via stdin simulation — write to temp file instead
	tmpFile := filepath.Join(t.TempDir(), "expanded.har")
	if err := os.WriteFile(tmpFile, []byte(expanded), 0644); err != nil {
		t.Fatal(err)
	}

	shrunk, _, err := harboring("shrink", tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSuffix(shrunk, "\n")
	if !bytes.Equal(orig, []byte(got)) {
		t.Fatal("expand|shrink round-trip produced different content")
	}
}

func TestExpandPrettyPrint(t *testing.T) {
	out, _, err := harboring("expand", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("expand output is not valid JSON")
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatal("expand output should start with {")
	}
}

func TestShrinkFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "output.har")
	orig, err := os.ReadFile("../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = harboring("shrink", "../../examples/pageload.har", "-o", tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(orig, got) {
		t.Fatal("shrink -o produces different content")
	}
}

func TestExpandFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "output.har")

	_, _, err := harboring("expand", "../../examples/pageload.har", "-o", tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatal("expand -o output is not valid JSON")
	}
	if !strings.Contains(string(got), "\n") {
		t.Fatal("expand -o output should be pretty-printed")
	}
}

func TestScrub(t *testing.T) {
	out, _, err := harboring("scrub", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("scrub output is not valid JSON")
	}
}

func TestScrubFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "scrubbed.har")

	_, _, err := harboring("scrub", "../../examples/pageload.har", "-o", tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatal("scrub -o output is not valid JSON")
	}
}

func TestDump(t *testing.T) {
	out, _, err := harboring("dump", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("dump output is not valid JSON")
	}
	if !strings.Contains(out, `"log"`) {
		t.Fatal("dump output should contain log key")
	}
}

func TestDumpScrubShrink(t *testing.T) {
	out, _, err := harboring("dump", "--scrub", "--shrink", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("dump --scrub --shrink output is not valid JSON")
	}
}

func TestHelp(t *testing.T) {
	out, _, err := harboring("help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "harboring") || !strings.Contains(out, "Usage:") {
		t.Fatalf("expected help text, got: %s", out)
	}
}

func TestHelpFlag(t *testing.T) {
	out, _, err := harboring("-h")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected help text, got: %s", out)
	}
}

func TestUnknownFlag(t *testing.T) {
	_, stderr, err := harboring("--unknown-flag")
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Fatalf("expected 'unknown flag' error, got: %s", stderr)
	}
}

func TestNoArgs(t *testing.T) {
	out, errOut, err := harboring()
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage message in stdout, got stdout=%q stderr=%q", out, errOut)
	}
}

func TestDumpScrub(t *testing.T) {
	out, _, err := harboring("dump", "--scrub", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("dump --scrub output is not valid JSON")
	}
}

func TestDumpShrink(t *testing.T) {
	out, _, err := harboring("dump", "--shrink", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("dump --shrink output is not valid JSON")
	}
}

func TestExpandNonExistentFile(t *testing.T) {
	_, stderr, err := harboring("expand", "nope.har")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("expected error message, got: %s", stderr)
	}
}

func TestStatsOnFullHar(t *testing.T) {
	out, _, err := harboring("stats", "../../examples/full.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Requests:") {
		t.Fatalf("expected Requests: in output, got: %s", out)
	}
}

func TestValidateFullHar(t *testing.T) {
	out, _, err := harboring("validate", "../../examples/full.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "VALID") {
		t.Fatalf("expected VALID, got: %s", out)
	}
}

func TestDefaultView(t *testing.T) {
	// Harboring with just a file name should default to view (TUI mode).
	// We can't really test TUI in a CLI test, so just verify it tries to parse
	// and fails gracefully or starts the program (which would hang).
	// Instead, let's verify the no-args error works: view with no file
	_, stderr, err := harboring("view")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "no HAR file") {
		t.Fatalf("expected 'no HAR file', got: %s", stderr)
	}
}

func TestPerf(t *testing.T) {
	out, _, err := harboring("perf", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Requests:") {
		t.Fatalf("expected Requests: in output, got: %s", out)
	}
	if !strings.Contains(out, "Total load time:") {
		t.Fatalf("expected Total load time: in output, got: %s", out)
	}
	if !strings.Contains(out, "TTFB Percentiles:") {
		t.Fatalf("expected TTFB Percentiles: in output, got: %s", out)
	}
	if !strings.Contains(out, "By Resource Type:") {
		t.Fatalf("expected By Resource Type: in output, got: %s", out)
	}
	if !strings.Contains(out, "By Domain (top 10):") {
		t.Fatalf("expected By Domain: in output, got: %s", out)
	}
	if !strings.Contains(out, "Status Codes:") {
		t.Fatalf("expected Status Codes: in output, got: %s", out)
	}
	if !strings.Contains(out, "Largest resource:") {
		t.Fatalf("expected Largest resource: in output, got: %s", out)
	}
	if !strings.Contains(out, "Slowest resource:") {
		t.Fatalf("expected Slowest resource: in output, got: %s", out)
	}
}
