package har

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProjectFiltersKeepsSpecifiedPaths(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"url":"https://example.com","method":"GET"},"response":{"status":200,"_transferSize":1024},"timings":{"wait":10,"dns":5}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.url", "timings.wait"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})
	entries := log["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})

	req := entry["request"].(map[string]interface{})
	if req["url"] != "https://example.com" {
		t.Fatalf("expected url to be preserved, got %v", req["url"])
	}
	if req["method"] != nil {
		t.Fatalf("expected method to be removed, got %v", req["method"])
	}

	if entry["response"] != nil {
		t.Fatalf("expected response to be removed, got %v", entry["response"])
	}

	timings := entry["timings"].(map[string]interface{})
	if timings["wait"] != float64(10) {
		t.Fatalf("expected wait to be 10, got %v", timings["wait"])
	}
	if timings["dns"] != nil {
		t.Fatalf("expected dns to be removed, got %v", timings["dns"])
	}
}

func TestProjectFiltersKeepsEntireSubtree(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"url":"https://example.com","method":"GET","headers":[]},"response":{"status":200}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})
	entries := log["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})

	req := entry["request"].(map[string]interface{})
	if req["url"] != "https://example.com" {
		t.Fatalf("expected url preserved, got %v", req["url"])
	}
	if req["method"] != "GET" {
		t.Fatalf("expected method preserved, got %v", req["method"])
	}
	if entry["response"] != nil {
		t.Fatalf("expected response to be removed, got %v", entry["response"])
	}
}

func TestProjectFiltersMultipleEntries(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"request":{"url":"https://a.com"},"timings":{"wait":1}},{"request":{"url":"https://b.com"},"timings":{"wait":2}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.url"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})
	entries := log["entries"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for i, e := range entries {
		entry := e.(map[string]interface{})
		req := entry["request"].(map[string]interface{})
		if req["url"] == nil {
			t.Fatalf("entry %d: expected url to be preserved", i)
		}
		if entry["timings"] != nil {
			t.Fatalf("entry %d: expected timings to be removed", i)
		}
	}
}

func TestProjectFiltersEmptyPathsReturnsOriginal(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"request":{"url":"https://a.com"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{})
	if projected != root {
		t.Fatal("empty paths should return original root")
	}
}

func TestProjectFiltersKeepsLogMetadata(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","creator":{"name":"Test","version":"1.0"},"entries":[{"request":{"url":"https://example.com"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.url"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})
	if log["version"] != "1.2" {
		t.Fatalf("expected version to be preserved, got %v", log["version"])
	}
	creator := log["creator"].(map[string]interface{})
	if creator["name"] != "Test" {
		t.Fatalf("expected creator.name to be preserved, got %v", creator["name"])
	}
}

func TestProjectFiltersInvalidPathsIgnored(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"request":{"url":"https://example.com"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"nonexistent.path"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})
	entries := log["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})
	if entry["request"] != nil {
		t.Fatalf("expected request to be removed for nonexistent path, got %v", entry["request"])
	}
}

func TestProjectFiltersExportValidJSON(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","entries":[{"request":{"url":"https://example.com","method":"GET"},"response":{"status":200},"timings":{"wait":10}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.url", "response.status"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(exported) {
		t.Fatal("projected export is not valid JSON")
	}
	if !strings.Contains(string(exported), "1.2") {
		t.Fatal("log version should still be present")
	}
}

func TestProjectFiltersGlobStar(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"timings":{"wait":10,"dns":5,"connect":3},"request":{"url":"https://example.com","method":"GET"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"timings.*"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	entry := result["log"].(map[string]interface{})["entries"].([]interface{})[0].(map[string]interface{})
	timings := entry["timings"].(map[string]interface{})
	if timings["wait"] != float64(10) || timings["dns"] != float64(5) || timings["connect"] != float64(3) {
		t.Fatalf("expected all timings to be preserved, got %v", timings)
	}
	if entry["request"] != nil {
		t.Fatal("expected request to be removed")
	}
}

func TestProjectFiltersGlobPattern(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"response":{"status":200,"_transferSize":1024,"content":{"size":512}},"request":{"url":"https://example.com"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"response.*Size*"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	entry := result["log"].(map[string]interface{})["entries"].([]interface{})[0].(map[string]interface{})
	resp := entry["response"].(map[string]interface{})
	if resp["_transferSize"] != float64(1024) {
		t.Fatalf("expected _transferSize to be preserved, got %v", resp["_transferSize"])
	}
	if resp["status"] != nil {
		t.Fatalf("expected status to be removed, got %v", resp["status"])
	}
}

func TestProjectFiltersGlobQuestionMark(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"a1":"keep","a2":"keep","b1":"drop","c":"drop"}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"a?"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	entry := result["log"].(map[string]interface{})["entries"].([]interface{})[0].(map[string]interface{})
	if entry["a1"] != "keep" || entry["a2"] != "keep" {
		t.Fatalf("expected a1 and a2 to be preserved, got %v", entry)
	}
	if entry["b1"] != nil || entry["c"] != nil {
		t.Fatalf("expected b1 and c to be removed, got %v", entry)
	}
}

func TestProjectFiltersGlobWithNested(t *testing.T) {
	data := []byte(`{"log":{"entries":[{"request":{"headers":[{"name":"Accept","value":"*/*"},{"name":"Host","value":"example.com"}],"url":"https://example.com"}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.headers.*", "request.url"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	entry := result["log"].(map[string]interface{})["entries"].([]interface{})[0].(map[string]interface{})
	req := entry["request"].(map[string]interface{})
	if req["url"] != "https://example.com" {
		t.Fatalf("expected url to be preserved, got %v", req["url"])
	}
	headers := req["headers"].([]interface{})
	if len(headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(headers))
	}
	for _, h := range headers {
		hdr := h.(map[string]interface{})
		if hdr["name"] == nil || hdr["value"] == nil {
			t.Fatalf("expected header name and value to be preserved, got %v", hdr)
		}
	}
}

func TestProjectFiltersPagesPassThrough(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","pages":[{"id":"page1","title":"Test","startedDateTime":"2024-01-01"}],"entries":[{"request":{"url":"https://example.com","method":"GET"},"timings":{"wait":10}}]}}`)
	root, err := ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	projected := ProjectFilters(root, []string{"request.url"})
	exported, err := ExportJSON(projected)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(exported, &result); err != nil {
		t.Fatal(err)
	}

	log := result["log"].(map[string]interface{})

	// Pages should pass through unfiltered
	pages := log["pages"].([]interface{})
	page := pages[0].(map[string]interface{})
	if page["id"] != "page1" {
		t.Fatalf("expected page id to be preserved, got %v", page["id"])
	}
	if page["title"] != "Test" {
		t.Fatalf("expected page title to be preserved, got %v", page["title"])
	}
	if page["startedDateTime"] != "2024-01-01" {
		t.Fatalf("expected page startedDateTime to be preserved, got %v", page["startedDateTime"])
	}

	// Entries should be filtered
	entries := log["entries"].([]interface{})
	entry := entries[0].(map[string]interface{})
	req := entry["request"].(map[string]interface{})
	if req["url"] != "https://example.com" {
		t.Fatalf("expected request.url to be preserved, got %v", req["url"])
	}
	if req["method"] != nil {
		t.Fatalf("expected request.method to be removed, got %v", req["method"])
	}
}
