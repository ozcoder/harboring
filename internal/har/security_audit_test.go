package har

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTightLimit temporarily lowers the decompression/input cap so the
// resource-exhaustion guards can be exercised with small inputs.
func withTightLimit(t *testing.T) {
	t.Helper()
	prev := maxDecompressedBytes
	maxDecompressedBytes = 1024
	t.Cleanup(func() { maxDecompressedBytes = prev })
}

func TestParseDeeplyNestedDoesNotCrash(t *testing.T) {
	// Build a JSON document nested far beyond maxParseDepth. Before the depth
	// guard this recursed until the goroutine stack overflowed (panic/crash).
	var b strings.Builder
	const depth = 20000
	for _ = range depth {
		b.WriteString(`{"a":`)
	}
	b.WriteString(`0`)
	for _ = range depth {
		b.WriteString(`}`)
	}

	_, err := ParseBytes([]byte(b.String()))
	if err == nil {
		t.Fatal("expected error for deeply nested input, got nil")
	}
	if strings.Contains(err.Error(), "nested too deeply") {
		return
	}
	t.Fatalf("expected 'nested too deeply' error, got: %v", err)
}

func TestDecompressBombRejected(t *testing.T) {
	withTightLimit(t)

	// A tiny gzip that expands to well over the (tightened) cap.
	var raw bytes.Buffer
	for i := 0; i < 5000; i++ {
		raw.WriteString("AAAAAAAAAA")
	}
	var gz bytes.Buffer
	gw := gzip.NewWriter(&gz)
	if _, err := gw.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := Decompress(gz.Bytes())
	if err == nil {
		t.Fatal("expected decompression-bomb rejection, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected limit error, got: %v", err)
	}
}

func TestReadAllLimitedRejectsOversized(t *testing.T) {
	withTightLimit(t)

	big := strings.Repeat("x", 5000)
	_, err := ReadAllLimited(strings.NewReader(big))
	if err == nil {
		t.Fatal("expected oversized-input rejection, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected limit error, got: %v", err)
	}
}

func TestReadFileLimitedRejectsOversized(t *testing.T) {
	withTightLimit(t)

	path := filepath.Join(t.TempDir(), "big.har")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 5000)), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFileLimited(path)
	if err == nil {
		t.Fatal("expected oversized-file rejection, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected limit error, got: %v", err)
	}
}

func TestValidInputStillParsesUnderLimit(t *testing.T) {
	withTightLimit(t)
	// minimalHAR is tiny, well under the tightened cap.
	if _, err := ParseBytes([]byte(minimalHAR)); err != nil {
		t.Fatalf("valid small input should parse: %v", err)
	}
}
