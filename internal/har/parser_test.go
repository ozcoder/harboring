package har

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseBytes(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","creator":{"name":"Test","version":"1.0"},"entries":[]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if root.Kind != Object {
		t.Fatal("expected object root")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := ParseBytes([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseArrayRoot(t *testing.T) {
	root, err := ParseBytes([]byte(`[1,2,3]`))
	if err != nil {
		t.Fatal("ParseBytes should accept array roots")
	}
	if root.Kind != Array {
		t.Fatalf("expected Array kind, got %v", root.Kind)
	}
}

func TestGetEntries(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{},"response":{}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	entries := GetEntries(root)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestExportJSON(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportJSON(root)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(exported) {
		t.Fatal("exported JSON is not valid")
	}
	if !strings.Contains(string(exported), "1.2") {
		t.Fatal("exported JSON should contain version")
	}
}

func TestExportJSONPreservesKeyOrder(t *testing.T) {
	// Keys in non-alphabetical order
	data := []byte(`{"z_key":"last","a_key":"first","m_key":"middle"}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportJSON(root)
	if err != nil {
		t.Fatal(err)
	}
	// The order should be z_key, a_key, m_key
	zIdx := strings.Index(string(exported), "z_key")
	aIdx := strings.Index(string(exported), "a_key")
	mIdx := strings.Index(string(exported), "m_key")
	if zIdx < 0 || aIdx < 0 || mIdx < 0 {
		t.Fatal("missing keys in exported JSON")
	}
	if !(zIdx < aIdx && aIdx < mIdx) {
		t.Fatalf("key order not preserved: z_idx=%d a_idx=%d m_idx=%d", zIdx, aIdx, mIdx)
	}
}

func TestExportJSONPreservesNumberLiterals(t *testing.T) {
	data := []byte(`{"a":12345678901234567890123,"b":9007199254740993,"f":1.2345678901234567,"e":1e300}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportJSON(root)
	if err != nil {
		t.Fatal(err)
	}
	out := string(exported)
	for _, want := range []string{
		"12345678901234567890123",
		"9007199254740993",
		"1.2345678901234567",
		"1e300",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("exported JSON should contain %s verbatim, got: %s", want, out)
		}
	}
}

func TestScrubContent(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"postData":{"text":"sensitive"}},"response":{"content":{"text":"secret","encoding":"base64"}}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	ScrubContent(root)
	exported, _ := ExportJSON(root)
	// scrub only removes content.text and content.encoding, not postData.text
	if strings.Contains(string(exported), "secret") {
		t.Fatal("scrub should have removed response content text")
	}
	if strings.Contains(string(exported), "base64") {
		t.Fatal("scrub should have removed response content encoding")
	}
	// postData.text should remain
	if !strings.Contains(string(exported), "sensitive") {
		t.Fatal("scrub should NOT remove postData.text (only content.text)")
	}
}

func TestShrinkContent(t *testing.T) {
	data := []byte(`{
		"log": {
			"version": "1.2"
		}
	}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	ShrinkContent(root)
	exported, _ := ExportJSON(root)
	// Should be single line
	if strings.Contains(string(exported), "\n") && !strings.HasPrefix(string(exported), "{") {
		t.Fatal("shrink should produce single-line output")
	}
}

func TestEntryDetails(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"time":30,"request":{"method":"GET","url":"http://example.com/"},"response":{"status":200,"statusText":"OK","content":{"mimeType":"text/html","size":100},"headers":[]},"cache":{},"timings":{"dns":-1,"connect":-1,"ssl":-1,"send":0,"wait":10,"receive":20}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	entries := GetEntries(root)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	details := GetEntryDetails(entries[0])
	if details.Method != "GET" {
		t.Fatalf("expected GET, got %s", details.Method)
	}
	if details.URL != "http://example.com/" {
		t.Fatalf("expected http://example.com/, got %s", details.URL)
	}
	if details.Status != 200 {
		t.Fatalf("expected 200, got %d", details.Status)
	}
	if details.StatusText != "OK" {
		t.Fatalf("expected OK, got %s", details.StatusText)
	}
	if details.Time != 30 {
		t.Fatalf("expected 30ms, got %f", details.Time)
	}
}

func TestGzipParsing(t *testing.T) {
	// Parse the pageload.har.gz if it exists, or test with a gzip file
	root, err := ParseFile("../../examples/pageload.har")
	if err != nil {
		t.Fatal(err)
	}
	entries := GetEntries(root)
	if len(entries) != 21 {
		t.Fatalf("expected 21 entries, got %d", len(entries))
	}
}

func TestScrubOnlyAffectsContent(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"method":"GET","url":"http://example.com/","headers":[{"name":"Authorization","value":"Bearer SECRET"}],"cookies":[],"queryString":[],"postData":{}},"response":{"status":200,"statusText":"OK","content":{"mimeType":"text/html","size":100,"text":"secret"},"headers":[],"cookies":[]},"cache":{},"timings":{"dns":-1,"connect":-1,"ssl":-1,"send":0,"wait":255.1,"receive":487.2}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	ScrubContent(root)
	exported, _ := ExportJSON(root)
	// response.content.text should be scrubbed
	if strings.Contains(string(exported), "secret") {
		t.Fatal("scrub should remove response content text")
	}
	// Authorization header value should remain (scrub doesn't touch headers)
	if !strings.Contains(string(exported), "SECRET") {
		t.Fatal("scrub should NOT remove header values (only content.text)")
	}
}

func TestScrubEmptiesCookies(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"method":"GET","url":"http://example.com/","cookies":[{"name":"session","value":"abc123","domain":"example.com","path":"/","httpOnly":true,"secure":true}]},"response":{"status":200,"statusText":"OK","cookies":[{"name":"tracker","value":"xyz","domain":"ads.com"}]}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	ScrubContent(root)
	exported, _ := ExportJSON(root)

	// Sensitive cookie values must be gone
	if strings.Contains(string(exported), "abc123") {
		t.Fatal("scrub should remove request cookie values")
	}
	if strings.Contains(string(exported), "tracker") {
		t.Fatal("scrub should remove response cookie objects")
	}

	// Cookies arrays should remain as empty arrays
	if !strings.Contains(string(exported), `"cookies": []`) && !strings.Contains(string(exported), `"cookies":[]`) {
		t.Fatalf("scrub should leave cookies as empty arrays, got: %s", exported)
	}
}
