package diff

import (
	"testing"

	"github.com/harboring/harboring/internal/har"
)

func TestDiffTreesIdentical(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"},"timings":{"wait":100}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"},"timings":{"wait":100}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs, got %d", len(diffs))
	}
}

func TestDiffTreesModifiedValue(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://other.com"}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Kind != Modified {
		t.Fatalf("expected Modified, got %v", diffs[0].Kind)
	}
}

func TestDiffTreesAddedField(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com","method":"GET"}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	added, _, _ := CountByKind(diffs)
	if added != 1 {
		t.Fatalf("expected 1 added, got %d", added)
	}
}

func TestDiffTreesRemovedField(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com","method":"GET"}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	_, removed, _ := CountByKind(diffs)
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
}

func TestDiffTreesCaseInsensitive(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"timings":{"Wait":100}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"timings":{"wait":100}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (case-insensitive), got %d", len(diffs))
	}
}

func TestDiffTreesFieldAliases(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"_transferSize":1024}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"_transfersize":1024}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (alias), got %d", len(diffs))
	}
}

func TestDiffTreesTimeVariance(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"timings":{"wait":100}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"timings":{"wait":109}}]}}`))

	// 9% difference, within 10% tolerance
	diffs := DiffTrees(left, right, Options{TimeVariancePercent: 10})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (within tolerance), got %d", len(diffs))
	}

	// 11% difference, outside 10% tolerance
	right2, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"timings":{"wait":111}}]}}`))
	diffs2 := DiffTrees(left, right2, Options{TimeVariancePercent: 10})
	if len(diffs2) != 1 {
		t.Fatalf("expected 1 diff (outside tolerance), got %d", len(diffs2))
	}
}

func TestDiffTreesIgnoreDates(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"startedDateTime":"2024-01-01T00:00:00.000Z"}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"startedDateTime":"2024-01-02T00:00:00.000Z"}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (date ignored), got %d", len(diffs))
	}
}

func TestDiffTreesKindMismatch(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":"https://example.com"}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"request":{"url":123}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (kind mismatch), got %d", len(diffs))
	}
	if diffs[0].Kind != Modified {
		t.Fatalf("expected Modified, got %v", diffs[0].Kind)
	}
}

func TestDiffTreesIgnoreContent(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"content":{"size":100,"text":"left body"}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"content":{"size":200,"text":"right body"}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs (content ignored), got %d", len(diffs))
	}
}

func TestDiffTreesIgnoreContentButNotSiblings(t *testing.T) {
	left, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"status":200,"content":{"text":"left"}}}]}}`))
	right, _ := har.ParseBytes([]byte(`{"log":{"entries":[{"response":{"status":404,"content":{"text":"right"}}}]}}`))

	diffs := DiffTrees(left, right, Options{})
	// status should differ, content should be ignored
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (status differs, content ignored), got %d", len(diffs))
	}
	if diffs[0].Kind != Modified {
		t.Fatalf("expected Modified, got %v", diffs[0].Kind)
	}
}
