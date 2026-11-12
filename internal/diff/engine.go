package diff

import (
	"fmt"
	"strings"

	"github.com/harboring/harboring/internal/har"
)

// Variety describes the type of difference between two nodes.
type Variety int

const (
	Identical Variety = iota
	Added
	Removed
	Modified
)

// NodeDiff represents the diff status of a single node path.
type NodeDiff struct {
	Path     string
	Kind     Variety
	Left     *har.Node
	Right    *har.Node
	LeftVal  string
	RightVal string
}

// Options configures the diff comparison.
type Options struct {
	// TimeVariancePercent is the maximum allowed percentage difference for
	// numeric values to be considered equal (e.g. 10 means 10% tolerance).
	TimeVariancePercent float64
}

// fieldAliases maps field names that differ between browser engines to a
// canonical name. Keys are lowercase.
var fieldAliases = map[string]string{
	"_transferSize":          "_transferSize",
	"_transfersize":          "_transferSize",
	"transfersize":           "_transferSize",
	"_encodedBodySize":       "_encodedBodySize",
	"_encoded bodysize":      "_encodedBodySize",
	"_decodedBodySize":       "_decodedBodySize",
	"_decoded bodysize":      "_decodedBodySize",
	"_resourceType":          "_resourceType",
	"_resourcetype":          "_resourceType",
	"resourcetype":           "_resourceType",
	"_initiator":             "_initiator",
	"_initiator_type":        "_initiatorType",
	"initiatortype":          "_initiatorType",
	"_priority":              "_priority",
	"_initialpriority":       "_initialPriority",
	"_initial_priority":      "_initialPriority",
	"_securityDetails":       "_securityDetails",
	"_securitydetails":       "_securityDetails",
	"_serverPort":            "_serverPort",
	"_serverport":            "_serverPort",
	"serveripaddress":        "serverIPAddress",
	"serverIPAddress":        "serverIPAddress",
	"contentencoding":        "contentEncoding",
	"content-encoding":       "contentEncoding",
	"cachcontrol":            "cacheControl",
	"cache-control":          "cacheControl",
	"lastmodified":           "lastModified",
	"last-modified":          "lastModified",
	"contenttype":            "mimeType",
	"content-type":           "mimeType",
	"content_length":         "bodySize",
	"content-length":         "bodySize",
	"bodySize":               "bodySize",
	"bodysize":               "bodySize",
	"requeststarteddatetime": "startedDateTime",
}

// dateFields contains field names that hold date/time values and should be
// ignored for diff purposes.
var dateFields = map[string]bool{
	"startedDateTime":     true,
	"starteddatetime":     true,
	"lastModified":        true,
	"last-modified":       true,
	"expires":             true,
	"date":                true,
	"if-modified-since":   true,
	"if-none-match":       true,
	"if-unmodified-since": true,
}

// ignoredFields contains field names whose entire subtrees should be skipped
// during diff. The "content" field holds response bodies which often differ
// between browsers due to content negotiation / user-agent sniffing.
var ignoredFields = map[string]bool{
	"content": true,
}

// CanonicalName normalizes a field name by lowercasing, replacing hyphens
// and spaces with underscores, and looking up aliases.
func CanonicalName(name string) string {
	lower := strings.ToLower(name)
	canonical := strings.ReplaceAll(lower, "-", "_")
	canonical = strings.ReplaceAll(canonical, " ", "_")
	if alias, ok := fieldAliases[canonical]; ok {
		return alias
	}
	return canonical
}

// isDateField returns true if the field name refers to a date/time value.
func isDateField(name string) bool {
	lower := strings.ToLower(name)
	if dateFields[lower] {
		return true
	}
	// Also detect fields ending with common date suffixes
	for _, suffix := range []string{"date", "time", "datetime", "modified", "expires"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// isIgnoredField returns true if the field name should be entirely skipped
// during diff comparison.
func isIgnoredField(name string) bool {
	return ignoredFields[strings.ToLower(name)]
}

// CompareTrees compares two HAR trees and returns a list of differences.
func CompareTrees(left, right *har.Node, opts Options) []NodeDiff {
	var diffs []NodeDiff
	diffWalk(left, right, "", &diffs, opts)
	return diffs
}

func diffWalk(left, right *har.Node, path string, diffs *[]NodeDiff, opts Options) {
	if left == nil && right == nil {
		return
	}
	if left == nil {
		*diffs = append(*diffs, NodeDiff{Path: path, Kind: Added, Right: right})
		return
	}
	if right == nil {
		*diffs = append(*diffs, NodeDiff{Path: path, Kind: Removed, Left: left})
		return
	}

	// Both present — compare by kind
	if left.Kind != right.Kind {
		*diffs = append(*diffs, NodeDiff{
			Path:     path,
			Kind:     Modified,
			Left:     left,
			Right:    right,
			LeftVal:  nodeSummary(left),
			RightVal: nodeSummary(right),
		})
		return
	}

	switch left.Kind {
	case har.Object:
		diffObjects(left, right, path, diffs, opts)
	case har.Array:
		diffArrays(left, right, path, diffs, opts)
	default:
		diffLeaves(left, right, path, diffs, opts)
	}
}

func diffObjects(left, right *har.Node, path string, diffs *[]NodeDiff, opts Options) {
	// Build maps of children by canonical name
	leftMap := make(map[string]*har.Node)
	rightMap := make(map[string]*har.Node)
	for _, c := range left.Children {
		leftMap[CanonicalName(c.Key)] = c
	}
	for _, c := range right.Children {
		rightMap[CanonicalName(c.Key)] = c
	}

	// Check all left keys
	for _, lc := range left.Children {
		canon := CanonicalName(lc.Key)
		childPath := joinPath(path, lc.Key)

		// Skip ignored fields entirely
		if isIgnoredField(lc.Key) {
			continue
		}

		rc, ok := rightMap[canon]
		if !ok {
			*diffs = append(*diffs, NodeDiff{Path: childPath, Kind: Removed, Left: lc})
			continue
		}
		diffWalk(lc, rc, childPath, diffs, opts)
	}

	// Check for keys only in right
	for _, rc := range right.Children {
		canon := CanonicalName(rc.Key)

		// Skip ignored fields entirely
		if isIgnoredField(rc.Key) {
			continue
		}

		if _, ok := leftMap[canon]; !ok {
			childPath := joinPath(path, rc.Key)
			*diffs = append(*diffs, NodeDiff{Path: childPath, Kind: Added, Right: rc})
		}
	}
}

func diffArrays(left, right *har.Node, path string, diffs *[]NodeDiff, opts Options) {
	maxLen := len(left.Children)
	if len(right.Children) > maxLen {
		maxLen = len(right.Children)
	}
	for i := range maxLen {
		childPath := joinPath(path, "") // arrays use index-based paths internally
		var lc, rc *har.Node
		if i < len(left.Children) {
			lc = left.Children[i]
		}
		if i < len(right.Children) {
			rc = right.Children[i]
		}
		switch {
		case lc == nil:
			*diffs = append(*diffs, NodeDiff{Path: childPath, Kind: Added, Right: rc})
			break
		case rc == nil:
			*diffs = append(*diffs, NodeDiff{Path: childPath, Kind: Removed, Left: lc})
			break
		default:
			diffWalk(lc, rc, childPath, diffs, opts)
		}
	}
}

func diffLeaves(left, right *har.Node, path string, diffs *[]NodeDiff, opts Options) {
	leftVal := leafText(left)
	rightVal := leafText(right)

	// Skip date fields
	if isDateField(extractFieldName(path)) {
		return
	}

	// Case-insensitive string comparison
	if strings.EqualFold(leftVal, rightVal) {
		return
	}

	// Numeric comparison with tolerance
	if isNumeric(leftVal) && isNumeric(rightVal) {
		if valuesWithinTolerance(leftVal, rightVal, opts.TimeVariancePercent) {
			return
		}
	}

	*diffs = append(*diffs, NodeDiff{
		Path:     path,
		Kind:     Modified,
		Left:     left,
		Right:    right,
		LeftVal:  leftVal,
		RightVal: rightVal,
	})
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}
	if key == "" {
		return parent
	}
	return parent + "." + key
}

func extractFieldName(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func nodeSummary(n *har.Node) string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case har.Object:
		return fmt.Sprintf("{%d fields}", len(n.Children))
	case har.Array:
		return fmt.Sprintf("[%d items]", len(n.Children))
	default:
		return leafText(n)
	}
}

func leafText(n *har.Node) string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case har.String:
		if s, ok := n.Value.(string); ok {
			return s
		}
	case har.Number:
		return n.NumberText()
	case har.Bool:
		if b, ok := n.Value.(bool); ok {
			if b {
				return "true"
			}
			return "false"
		}
	case har.Null:
		return "null"
	}
	return ""
}

func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '+' || c == 'e' || c == 'E' {
			continue
		}
		return false
	}
	return true
}

func parseNumber(s string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f, err == nil
}

func valuesWithinTolerance(a, b string, pct float64) bool {
	if pct <= 0 {
		return false
	}
	fa, okA := parseNumber(a)
	fb, okB := parseNumber(b)
	if !okA || !okB {
		return false
	}
	if fa == fb {
		return true
	}
	base := fa
	if base == 0 {
		base = fb
	}
	if base == 0 {
		return false
	}
	diff := fa - fb
	if diff < 0 {
		diff = -diff
	}
	if base < 0 {
		base = -base
	}
	return (diff / base * 100) <= pct
}

// HasDifferences returns true if any diffs are not Identical.
func HasDifferences(diffs []NodeDiff) bool {
	for _, d := range diffs {
		if d.Kind != Identical {
			return true
		}
	}
	return false
}

// CountByKind returns the count of each diff kind.
func CountByKind(diffs []NodeDiff) (added, removed, modified int) {
	for _, d := range diffs {
		switch d.Kind {
		case Added:
			added++
		case Removed:
			removed++
		case Modified:
			modified++
		}
	}
	return
}
