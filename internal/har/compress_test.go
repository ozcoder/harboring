package har

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const minimalHAR = `{"log":{"version":"1.2","creator":{"name":"t","version":"1"},"entries":[{"request":{"method":"GET","url":"http://a/","headers":[],"cookies":[],"queryString":[],"postData":{}},"response":{"status":200,"statusText":"OK","content":{"mimeType":"text/html","size":1,"text":"x"},"headers":[],"cookies":[]},"cache":{},"timings":{"dns":-1,"connect":-1,"ssl":-1,"send":0,"wait":1,"receive":2}}]}}`

func TestDecompressGzip(t *testing.T) {
	plain := []byte(minimalHAR)
	gz, err := Compress(plain, FormatGzip)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decompress(gz)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != minimalHAR {
		t.Fatalf("gzip round-trip mismatch:\n%s", got)
	}
}

func TestDecompressBzip2(t *testing.T) {
	plain := []byte(minimalHAR)
	bz, err := Compress(plain, FormatBzip2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decompress(bz)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != minimalHAR {
		t.Fatalf("bzip2 round-trip mismatch:\n%s", got)
	}
}

func TestParseReader(t *testing.T) {
	cases := map[string][]byte{
		"plain": []byte(minimalHAR),
		"gzip":  mustCompress(t, FormatGzip, []byte(minimalHAR)),
		"bzip2": mustCompress(t, FormatBzip2, []byte(minimalHAR)),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			root, err := ParseReader(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("ParseReader(%s): %v", name, err)
			}
			if got := len(GetEntries(root)); got != 1 {
				t.Fatalf("ParseReader(%s): expected 1 entry, got %d", name, got)
			}
		})
	}
}

func mustCompress(t *testing.T, f CompressionFormat, data []byte) []byte {
	t.Helper()
	out, err := Compress(data, f)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDecompressPlainPassthrough(t *testing.T) {
	plain := []byte(minimalHAR)
	got, err := Decompress(plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != minimalHAR {
		t.Fatalf("plain data should pass through unchanged, got:\n%s", got)
	}
}

func TestParseFileBzip2(t *testing.T) {
	bz, err := Compress([]byte(minimalHAR), FormatBzip2)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample.har.bz2")
	if err := os.WriteFile(path, bz, 0644); err != nil {
		t.Fatal(err)
	}
	root, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(GetEntries(root)); got != 1 {
		t.Fatalf("expected 1 entry from bzip2 file, got %d", got)
	}
}

func TestParseFileGzip(t *testing.T) {
	gz, err := Compress([]byte(minimalHAR), FormatGzip)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample.har.gz")
	if err := os.WriteFile(path, gz, 0644); err != nil {
		t.Fatal(err)
	}
	root, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(GetEntries(root)); got != 1 {
		t.Fatalf("expected 1 entry from gzip file, got %d", got)
	}
}

func TestMinifyFileBzip2RoundTrip(t *testing.T) {
	bz, err := Compress([]byte(minimalHAR), FormatBzip2)
	if err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(t.TempDir(), "in.har.bz2")
	out := filepath.Join(t.TempDir(), "out.har.bz2")
	if err := os.WriteFile(in, bz, 0644); err != nil {
		t.Fatal(err)
	}
	if err := MinifyFile(in, out); err != nil {
		t.Fatal(err)
	}
	root, err := ParseFile(out)
	if err != nil {
		t.Fatalf("bzip2 output should parse: %v", err)
	}
	if len(GetEntries(root)) != 1 {
		t.Fatalf("expected 1 entry after minify round-trip, got %d", len(GetEntries(root)))
	}
}

func TestWriteFileAutoExtension(t *testing.T) {
	data := []byte(minimalHAR)
	dir := t.TempDir()

	plainPath := filepath.Join(dir, "a.har")
	bzPath := filepath.Join(dir, "b.har.bz2")
	gzPath := filepath.Join(dir, "c.har.gz")

	for _, p := range []string{plainPath, bzPath, gzPath} {
		if err := WriteFileAuto(p, data); err != nil {
			t.Fatalf("WriteFileAuto(%s): %v", p, err)
		}
	}

	// Every output, regardless of extension, must parse back to the same HAR.
	for _, p := range []string{plainPath, bzPath, gzPath} {
		root, err := ParseFile(p)
		if err != nil {
			t.Fatalf("ParseFile(%s) after WriteFileAuto: %v", p, err)
		}
		if len(GetEntries(root)) != 1 {
			t.Fatalf("ParseFile(%s): expected 1 entry, got %d", p, len(GetEntries(root)))
		}
	}
}
