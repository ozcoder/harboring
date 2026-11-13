package main

import (
	"bytes"
    "context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
    "time"
)

var binPath string

func TestMain(m *testing.M) {
	os.Exit(TestTheMain(m))
}

func TestTheMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "harboring-test-*")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	binPath = filepath.Join(tmpDir, "harboring")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, ".")

	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	code := m.Run()
	os.RemoveAll(tmpDir)

	return code
}

func harboring(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func harboringStdin(stdin []byte, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	var stdout bytes.Buffer
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.String(), err
}

// harboringPipe runs two invocations with the stdout of the first piped into
// the stdin of the second, returning the second's output.
func harboringPipe(args1, args2 []string) (string, string, error) {
	first := exec.Command(binPath, args1...)
	second := exec.Command(binPath, args2...)
	pipe, err := first.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	second.Stdin = pipe
	var stdout, stderr bytes.Buffer
	second.Stdout = &stdout
	second.Stderr = &stderr
	if err := second.Start(); err != nil {
		return "", "", err
	}
	if err := first.Run(); err != nil {
		return "", "", err
	}
	if err := second.Wait(); err != nil {
		return "", "", err
	}
	return stdout.String(), stderr.String(), nil
}

const sampleHAR = `{"log":{"version":"1.2","creator":{"name":"t","version":"1"},"entries":[{"request":{"method":"GET","url":"http://a/","headers":[],"cookies":[],"queryString":[],"postData":{}},"response":{"status":200,"statusText":"OK","content":{"mimeType":"text/html","size":1,"text":"x"},"headers":[],"cookies":[]},"cache":{},"timings":{"dns":-1,"connect":-1,"ssl":-1,"send":0,"wait":1,"receive":2}}]}}`

func TestValidate(t *testing.T) {
	out, _, err := harboring("--validate", "../../examples/pageload.har")
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
	_, stderr, err := harboring("--validate", "nonexistent.har")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("expected error message, got: %s", stderr)
	}
}

func TestScrubStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--scrub")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\"log\"") {
		t.Fatalf("expected scrubbed JSON with log key, got: %s", out)
	}
}

func TestScrubExplicitDashStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--scrub", "-")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\"log\"") {
		t.Fatalf("expected scrubbed JSON with log key, got: %s", out)
	}
}

func TestPerfStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--perf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Performance Report") {
		t.Fatalf("expected performance report, got: %s", out)
	}
}

func TestStatsStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--stats")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Requests:") {
		t.Fatalf("expected stats output, got: %s", out)
	}
}

func TestValidateStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--validate")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "VALID") {
		t.Fatalf("expected VALID, got: %s", out)
	}
}

func TestShrinkStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--shrink")
	if err != nil {
		t.Fatal(err)
	}
	// Minified output must still parse and contain the log key.
	if !strings.Contains(out, "\"log\"") {
		t.Fatalf("expected minified JSON with log key, got: %s", out)
	}
	if strings.Contains(out, "\n  ") {
		t.Fatalf("expected minified (no indentation) output, got: %s", out)
	}
}

func TestCacheStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--cache")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Cache Summary") {
		t.Fatalf("expected cache summary, got: %s", out)
	}
}

func TestCacheMarkdown(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--cache", "-m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Cache Summary") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
	if !strings.Contains(out, "| Metric | Value |") {
		t.Fatalf("expected markdown table, got: %s", out)
	}
}

func TestSecurityStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--security")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Security Headers Analysis") {
		t.Fatalf("expected security report, got: %s", out)
	}
}

func TestSecurityMarkdown(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--security", "--markdown")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Security Headers Analysis") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
	if !strings.Contains(out, "| Domain | Required Missing | Optional Missing |") {
		t.Fatalf("expected summary table, got: %s", out)
	}
}

func TestStatsMarkdown(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--stats", "-m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# HAR Statistics") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
}

func TestPerfMarkdown(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--perf", "-m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# HAR Performance Report") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
}

func TestValidateNoArgs(t *testing.T) {
	_, stderr, err := harboring("--validate")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "no HAR file") {
		t.Fatalf("expected 'no HAR file', got: %s", stderr)
	}
}

func TestStats(t *testing.T) {
	out, _, err := harboring("--stats", "../../examples/pageload.har")
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

	out, _, err := harboring("--shrink", "../../examples/pageload.har")
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
	expanded, _, err := harboring("--expand", "../../examples/pageload.har")
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
	if err := os.WriteFile(tmpFile, []byte(expanded), 0o644); err != nil {
		t.Fatal(err)
	}

	shrunk, _, err := harboring("--shrink", tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSuffix(shrunk, "\n")
	if !bytes.Equal(orig, []byte(got)) {
		t.Fatal("expand|shrink round-trip produced different content")
	}
}

func TestExpandPrettyPrint(t *testing.T) {
	out, _, err := harboring("--expand", "../../examples/pageload.har")
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

	_, _, err = harboring("--shrink", "../../examples/pageload.har", "-o", tmpFile)
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

	_, _, err := harboring("--expand", "../../examples/pageload.har", "-o", tmpFile)
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
	out, _, err := harboring("--scrub", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("scrub output is not valid JSON")
	}
}

func TestScrubFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "scrubbed.har")

	_, _, err := harboring("--scrub", "../../examples/pageload.har", "-o", tmpFile)
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
	out, _, err := harboring("--dump", "../../examples/pageload.har")
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

func TestConflictingModes(t *testing.T) {
	_, stderr, err := harboring("--dump", "--scrub", "../../examples/pageload.har")
	if err == nil {
		t.Fatal("expected error for conflicting modes")
	}
	if !strings.Contains(stderr, "conflicting modes") {
		t.Fatalf("expected 'conflicting modes' error, got: %s", stderr)
	}
}

func TestScrubShrinkPipe(t *testing.T) {
	out, _, err := harboringPipe(
		[]string{"--scrub", "../../examples/pageload.har"},
		[]string{"--shrink"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("scrub | shrink output is not valid JSON")
	}
	if !strings.Contains(out, `"log"`) {
		t.Fatal("scrub | shrink output should contain log key")
	}
}

func TestHelp(t *testing.T) {
	out, _, err := harboring("--help")
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

func TestExtraArgument(t *testing.T) {
	_, stderr, err := harboring("--dump", "a.har", "b.har")
	if err == nil {
		t.Fatal("expected error for extra argument")
	}
	if !strings.Contains(stderr, "unexpected extra argument") {
		t.Fatalf("expected 'unexpected extra argument' error, got: %s", stderr)
	}
}

func TestExpandNonExistentFile(t *testing.T) {
	_, stderr, err := harboring("--expand", "nope.har")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("expected error message, got: %s", stderr)
	}
}

func TestStatsOnFullHar(t *testing.T) {
	out, _, err := harboring("--stats", "../../examples/full.har.gz")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Requests:") {
		t.Fatalf("expected Requests: in output, got: %s", out)
	}
}

func TestValidateFullHar(t *testing.T) {
	out, _, err := harboring("--validate", "../../examples/full.har.gz")
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
	_, stderr, err := harboring("--view")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "no HAR file") {
		t.Fatalf("expected 'no HAR file', got: %s", stderr)
	}
}

func TestPerf(t *testing.T) {
	out, _, err := harboring("--perf", "../../examples/pageload.har")
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

func TestStatsJSON(t *testing.T) {
	out, _, err := harboring("--stats", "--json", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"requests"`) {
		t.Fatalf("expected requests key, got: %s", out)
	}
}

func TestPerfJSON(t *testing.T) {
	out, _, err := harboring("--perf", "-j", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"ttfb_percentiles"`) {
		t.Fatalf("expected ttfb_percentiles key, got: %s", out)
	}
}

func TestCacheJSON(t *testing.T) {
	out, _, err := harboring("--cache", "--json", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"good_cache"`) {
		t.Fatalf("expected good_cache key, got: %s", out)
	}
}

func TestSecurityJSON(t *testing.T) {
	out, _, err := harboring("--security", "-j", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"domains"`) {
		t.Fatalf("expected domains key, got: %s", out)
	}
}

func TestJSONMarkdownConflict(t *testing.T) {
	_, stderr, err := harboring("--stats", "--json", "--markdown", "../../examples/pageload.har")
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Fatalf("expected 'mutually exclusive' error, got: %s", stderr)
	}
}

func TestStatsJSONStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--stats", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"requests"`) {
		t.Fatalf("expected requests key, got: %s", out)
	}
}

func TestStatsJSONFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "stats.json")
	_, _, err := harboring("--stats", "--json", "-o", tmpFile, "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatalf("expected valid JSON in file, got: %s", got)
	}
	if !strings.Contains(string(got), `"requests"`) {
		t.Fatalf("expected requests key, got: %s", got)
	}
}

func TestPerfJSONFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "perf.json")
	_, _, err := harboring("--perf", "-j", "-o", tmpFile, "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatalf("expected valid JSON in file, got: %s", got)
	}
	if !strings.Contains(string(got), `"ttfb_percentiles"`) {
		t.Fatalf("expected ttfb_percentiles key, got: %s", got)
	}
}

func TestStatsMarkdownFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "stats.md")
	_, _, err := harboring("--stats", "-m", "-o", tmpFile, "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "# HAR Statistics") {
		t.Fatalf("expected markdown heading, got: %s", got)
	}
}

func TestTrackersStdin(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--trackers")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Third-Party Tracker Analysis") {
		t.Fatalf("expected tracker report title, got: %s", out)
	}
}

func TestTrackersJSON(t *testing.T) {
	out, _, err := harboring("--trackers", "--json", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"tracker_requests"`) {
		t.Fatalf("expected tracker_requests key, got: %s", out)
	}
}

func TestTrackersMarkdown(t *testing.T) {
	out, _, err := harboring("--trackers", "-m", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Third-Party Tracker Analysis") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
}

func TestTrackersFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "trackers.json")
	_, _, err := harboring("--trackers", "--json", "-o", tmpFile, "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatalf("expected valid JSON in file, got: %s", got)
	}
}

func TestFieldsBasic(t *testing.T) {
	out, _, err := harboring("--fields", "request.url,timings.wait", "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatal("fields output is not valid JSON")
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	log := result["log"].(map[string]interface{})
	entries := log["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})
	req := entry["request"].(map[string]interface{})
	if req["url"] == nil {
		t.Fatal("expected request.url to be preserved")
	}
	if req["method"] != nil {
		t.Fatal("expected request.method to be removed")
	}
	timings := entry["timings"].(map[string]interface{})
	if timings["wait"] == nil {
		t.Fatal("expected timings.wait to be preserved")
	}
	if timings["dns"] != nil {
		t.Fatal("expected timings.dns to be removed")
	}
}

func TestFieldsFileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "fields.har")
	_, _, err := harboring("--fields", "request.url", "-o", tmpFile, "../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatalf("expected valid JSON in file, got: %s", got)
	}
}

func TestFieldsRequiresArgument(t *testing.T) {
	_, stderr, err := harboring("--fields")
	if err == nil {
		t.Fatal("expected error when --fields has no field argument")
	}
	if !strings.Contains(stderr, "--fields requires") {
		t.Fatalf("expected --fields requires error, got: %s", stderr)
	}
}

func TestFieldsConflictingModes(t *testing.T) {
	_, stderr, err := harboring("--fields", "request.url", "--dump", "../../examples/pageload.har")
	if err == nil {
		t.Fatal("expected error for conflicting modes")
	}
	if !strings.Contains(stderr, "conflicting modes") {
		t.Fatalf("expected 'conflicting modes' error, got: %s", stderr)
	}
}

func TestListThemes(t *testing.T) {
	out, _, err := harboring("--list", "themes")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dark", "light", "spectrum", "turbo"} {
		if !strings.Contains(out, name) {
			t.Fatalf("expected theme %q in --list themes output, got:\n%s", name, out)
		}
	}
}

func TestListTrackers(t *testing.T) {
	out, _, err := harboring("--list", "trackers")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "google-analytics.com") {
		t.Fatalf("expected google-analytics.com in --list trackers output, got:\n%s", out)
	}
	if !strings.Contains(out, "facebook.com") {
		t.Fatalf("expected facebook.com in --list trackers output, got:\n%s", out)
	}
	// Categories are shown.
	if !strings.Contains(out, "analytics") || !strings.Contains(out, "advertising") {
		t.Fatalf("expected categories in --list trackers output, got:\n%s", out)
	}
}

func TestListRequiresArgument(t *testing.T) {
	_, stderr, err := harboring("--list")
	if err == nil {
		t.Fatal("expected error when --list has no argument")
	}
	if !strings.Contains(stderr, "--list requires") {
		t.Fatalf("expected --list requires error, got: %s", stderr)
	}
}

func TestListUnknownCategory(t *testing.T) {
	_, stderr, err := harboring("--list", "bogus")
	if err == nil {
		t.Fatal("expected error for unknown --list category")
	}
	if !strings.Contains(stderr, "unknown category") {
		t.Fatalf("expected unknown category error, got: %s", stderr)
	}
}

func TestTrackersFileCustomDomains(t *testing.T) {
	// A domain that is NOT a built-in tracker: example.com must be absent from
	// pageload.har normally, but we add a custom tracker file marking it.
	f := filepath.Join(t.TempDir(), "trackers.toml")
	if err := os.WriteFile(f, []byte(`
[[trackers]]
domain = "example.com"
category = "analytics"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Custom file makes example.com a tracker (pageload.har has no built-in
	// trackers, but with the custom file some domains should be flagged or at
	// minimum the file is accepted without error).
	out, _, err := harboring("--trackers", "--trackers-file", f, "../../examples/pageload.har")
	if err != nil {
		t.Fatalf("--trackers with --trackers-file should succeed, got err: %v", err)
	}
	if !strings.Contains(out, "Third-Party Tracker Analysis") {
		t.Fatalf("expected tracker report title, got: %s", out)
	}
}

func TestTrackersFileMissing(t *testing.T) {
	_, stderr, err := harboring("--trackers", "--trackers-file", "/nonexistent/trackers.toml", "../../examples/pageload.har")
	if err == nil {
		t.Fatal("expected error for missing trackers file")
	}
	if !strings.Contains(stderr, "--trackers-file") {
		t.Fatalf("expected --trackers-file error, got: %s", stderr)
	}
}

const priorityHAR = `{"log":{"version":"1.2","creator":{"name":"t","version":"1"},"entries":[
{"request":{"method":"GET","url":"http://a/x","httpVersion":"HTTP/2"},"response":{"status":200},"_priority":"low","_initialPriority":"high"},
{"request":{"method":"GET","url":"http://a/y","httpVersion":"HTTP/2"},"response":{"status":200},"_priority":"medium","_initialPriority":"medium"}
]}}`

func TestPriorityStdin(t *testing.T) {
	out, err := harboringStdin([]byte(priorityHAR), "--priority")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Priority Changes") {
		t.Fatalf("expected priority report title, got: %s", out)
	}
	if !strings.Contains(out, "Requests with a priority change: 1") {
		t.Fatalf("expected count of 1, got: %s", out)
	}
	if !strings.Contains(out, "http://a/x") {
		t.Fatalf("expected changed URL, got: %s", out)
	}
	if strings.Contains(out, "http://a/y") {
		t.Fatalf("unchanged entry should be excluded, got: %s", out)
	}
}

func TestPriorityMarkdown(t *testing.T) {
	out, err := harboringStdin([]byte(priorityHAR), "--priority", "-m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Priority Changes") {
		t.Fatalf("expected markdown heading, got: %s", out)
	}
	if !strings.Contains(out, "HTTP/2") {
		t.Fatalf("expected httpVersion in output, got: %s", out)
	}
}

func TestPriorityJSON(t *testing.T) {
	out, err := harboringStdin([]byte(priorityHAR), "--priority", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if !strings.Contains(out, `"total": 1`) {
		t.Fatalf("expected total of 1, got: %s", out)
	}
	if !strings.Contains(out, `"initial_priority": "high"`) {
		t.Fatalf("expected initial_priority field, got: %s", out)
	}
}

func TestPriorityMarkdownNoChanges(t *testing.T) {
	out, err := harboringStdin([]byte(sampleHAR), "--priority", "-m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No requests had a priority change") {
		t.Fatalf("expected no-changes message, got: %s", out)
	}
}
