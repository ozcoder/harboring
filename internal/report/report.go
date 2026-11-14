package report

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/harboring/harboring/internal/har"
	"golang.org/x/net/html"
)
func findChildNode(n *har.Node, key string) *har.Node {
	if n == nil {
		return nil
	}
	for _, child := range n.Children {
		if child.Key == key {
			return child
		}
	}
	return nil
}

// formatBytes renders a byte count using binary units.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// mdCell escapes a string for safe inclusion in a Markdown table cell.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// mdTable renders a GitHub-flavored Markdown table.
func mdTable(headers []string, rows [][]string) string {
	var b strings.Builder
	cells := make([]string, len(headers))
	for i, h := range headers {
		cells[i] = mdCell(h)
	}
	b.WriteString("| ")
	b.WriteString(strings.Join(cells, " | "))
	b.WriteString(" |\n")

	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| ")
	b.WriteString(strings.Join(seps, " | "))
	b.WriteString(" |\n")

	for _, row := range rows {
		cells = cells[:0]
		for _, c := range row {
			cells = append(cells, mdCell(c))
		}
		b.WriteString("| ")
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}

// StatsReport builds a summary statistics report.
func StatsReport(entries []*har.EntryDetails, markdown bool) string {
	var totalSize int64
	var totalTime float64
	methods := map[string]int{}
	statuses := map[int]int{}

	for _, d := range entries {
		totalSize += d.Size
		totalTime += d.Time
		methods[d.Method]++
		statuses[d.Status]++
	}

	type timedEntry struct {
		method string
		url    string
		time   float64
	}
	slowest := make([]timedEntry, 0, len(entries))
	for _, d := range entries {
		slowest = append(slowest, timedEntry{method: d.Method, url: d.URL, time: d.Time})
	}
	sort.Slice(slowest, func(i, j int) bool { return slowest[i].time > slowest[j].time })
	limit := 5
	if len(slowest) < limit {
		limit = len(slowest)
	}

	if markdown {
		var b strings.Builder
		b.WriteString("# HAR Statistics\n\n")
		b.WriteString(mdTable(
			[]string{"Metric", "Value"},
			[][]string{
				{"Requests", fmt.Sprintf("%d", len(entries))},
				{"Total size", formatBytes(totalSize)},
				{"Total time", fmt.Sprintf("%.0f ms", totalTime)},
			},
		))
		b.WriteString("\n## Methods\n\n")
		b.WriteString(mdTable([]string{"Method", "Count"}, methodRows(methods)))
		b.WriteString("\n## Status Codes\n\n")
		b.WriteString(mdTable([]string{"Status", "Count"}, statusRows(statuses)))
		if limit > 0 {
			b.WriteString("\n## Top 5 Slowest\n\n")
			rows := make([][]string, 0, limit)
			for i := range limit {
				rows = append(rows, []string{
					fmt.Sprintf("%.0f ms", slowest[i].time),
					slowest[i].method,
					slowest[i].url,
				})
			}
			b.WriteString(mdTable([]string{"Time (ms)", "Method", "URL"}, rows))
		}
		return b.String()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Requests:    %d\n", len(entries))
	fmt.Fprintf(&b, "Total size:  %s\n", formatBytes(totalSize))
	fmt.Fprintf(&b, "Total time:  %.0f ms\n", totalTime)
	b.WriteString("\nMethods:\n")
	for _, r := range methodRows(methods) {
		fmt.Fprintf(&b, "  %-8s %s\n", r[0], r[1])
	}
	b.WriteString("\nStatus codes:\n")
	for _, r := range statusRows(statuses) {
		fmt.Fprintf(&b, "  %s  %s\n", r[0], r[1])
	}
	if limit > 0 {
		b.WriteString("\nTop 5 slowest:\n")
		for i := 0; i < limit; i++ {
			fmt.Fprintf(&b, "  %.0f ms  %s %s\n", slowest[i].time, slowest[i].method, slowest[i].url)
		}
	}
	return b.String()
}

func PerfReport(entries []*har.EntryDetails, markdown bool) string {
	var totalSize int64
	var totalTime float64
	var ttfbs []float64
	byType := map[string]int64{}
	byDomain := map[string]int64{}
	byStatus := map[int]int{}
	var largest, slowest *har.EntryDetails
	var largeFonts []*har.EntryDetails

	for _, d := range entries {
		totalSize += d.Size
		totalTime += d.Time
		if d.Wait > 0 {
			ttfbs = append(ttfbs, d.Wait)
		}
		rt := d.ResourceType
		if rt == "" {
			rt = "other"
		}
		byType[rt] += d.Size
		byDomain[d.Domain] += d.Size
		byStatus[d.Status]++
		if largest == nil || d.Size > largest.Size {
			largest = d
		}
		if slowest == nil || d.Time > slowest.Time {
			slowest = d
		}
		if d.ResourceType == "font" && d.Size > 20*1024 && !hasRequestHeader(d, "unicode-range") {
			largeFonts = append(largeFonts, d)
		}
	}

	sort.Float64s(ttfbs)
	p50 := percentile(ttfbs, 50)
	p90 := percentile(ttfbs, 90)
	p95 := percentile(ttfbs, 95)
	p99 := percentile(ttfbs, 99)

	if markdown {
		var b strings.Builder
		b.WriteString("# HAR Performance Report\n\n")
		b.WriteString(mdTable(
			[]string{"Metric", "Value"},
			[][]string{
				{"Requests", fmt.Sprintf("%d", len(entries))},
				{"Total size", formatBytes(totalSize)},
				{"Total load time", fmt.Sprintf("%.0f ms", totalTime)},
			},
		))
		b.WriteString("\n## TTFB Percentiles\n\n")
		b.WriteString(mdTable(
			[]string{"Percentile", "Time (ms)"},
			[][]string{
				{"p50", fmt.Sprintf("%.0f", p50)},
				{"p90", fmt.Sprintf("%.0f", p90)},
				{"p95", fmt.Sprintf("%.0f", p95)},
				{"p99", fmt.Sprintf("%.0f", p99)},
			},
		))
		b.WriteString("\n## By Resource Type\n\n")
		b.WriteString(mdTable([]string{"Type", "Size"}, sortedKV(byType)))
		b.WriteString("\n## By Domain (top 10)\n\n")
		b.WriteString(mdTable([]string{"Domain", "Size"}, topN(sortedKV(byDomain), 10)))
		b.WriteString("\n## Status Codes\n\n")
		b.WriteString(mdTable([]string{"Status", "Count"}, statusRows(byStatus)))
		b.WriteString("\n## Largest Resource\n\n")
		if largest != nil {
			b.WriteString("- **Method**: " + largest.Method + "\n")
			b.WriteString("- **URL**: " + largest.URL + "\n")
			b.WriteString("- **Size**: " + formatBytes(largest.Size) + "\n")
		}
		b.WriteString("\n## Slowest Resource\n\n")
		if slowest != nil {
			fmt.Fprintf(&b, "- **Time**: %.0f ms\n", slowest.Time)
			b.WriteString("- **Method**: " + slowest.Method + "\n")
			b.WriteString("- **URL**: " + slowest.URL + "\n")
		}
		if len(largeFonts) > 0 {
			b.WriteString("\n## Fonts\n\n")
			b.WriteString("Font requests over 20 KB without a `unicode-range` header:\n\n")
			b.WriteString(mdTable(
				[]string{"Method", "URL", "Size"},
				fontRows(largeFonts),
			))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("HAR Performance Report\n")
	b.WriteString("========================\n\n")
	fmt.Fprintf(&b, "Requests:        %d\n", len(entries))
	fmt.Fprintf(&b, "Total load time: %.0f ms\n", totalTime)
	b.WriteString("\nTTFB Percentiles:\n")
	fmt.Fprintf(&b, "  p50: %.0f ms\n", p50)
	fmt.Fprintf(&b, "  p90: %.0f ms\n", p90)
	fmt.Fprintf(&b, "  p95: %.0f ms\n", p95)
	fmt.Fprintf(&b, "  p99: %.0f ms\n", p99)
	b.WriteString("\nBy Resource Type:\n")
	for _, r := range sortedKV(byType) {
		fmt.Fprintf(&b, "  %-12s %s\n", r[0]+":", formatBytes(parseInt64(r[1])))
	}
	b.WriteString("\nBy Domain (top 10):\n")
	for i, r := range topN(sortedKV(byDomain), 10) {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "  %-30s %s\n", r[0]+":", formatBytes(parseInt64(r[1])))
	}
	b.WriteString("\nStatus Codes:\n")
	for _, r := range statusRows(byStatus) {
		fmt.Fprintf(&b, "  %s: %s\n", r[0], r[1])
	}
	b.WriteString("\nLargest resource:\n")
	if largest != nil {
		fmt.Fprintf(&b, "  %s %s (%s)\n", largest.Method, largest.URL, formatBytes(largest.Size))
	}
	b.WriteString("Slowest resource:\n")
	if slowest != nil {
		fmt.Fprintf(&b, "  %.0f ms  %s %s\n", slowest.Time, slowest.Method, slowest.URL)
	}
	if len(largeFonts) > 0 {
		b.WriteString("\nFonts (over 20 KB without unicode-range):\n")
		for _, f := range largeFonts {
			fmt.Fprintf(&b, "  %s %s (%s)\n", f.Method, f.URL, formatBytes(f.Size))
		}
	}
	return b.String()
}

func fontRows(fonts []*har.EntryDetails) [][]string {
	rows := make([][]string, len(fonts))
	for i, f := range fonts {
		rows[i] = []string{f.Method, f.URL, formatBytes(f.Size)}
	}
	return rows
}

func sortedKV(m map[string]int64) [][]string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, fmt.Sprintf("%d", m[k])})
	}
	return rows
}

func topN(rows [][]string, n int) [][]string {
	if len(rows) <= n {
		return rows
	}
	return rows[:n]
}

func CacheReport(entries []*har.EntryDetails, markdown bool) string {
	var noCache, shortCache, goodCache, overCache []*har.EntryDetails
	for _, e := range entries {
		verdict, _ := har.AnalyzeCache(e)
		switch verdict {
		case har.CacheNone:
			noCache = append(noCache, e)
		case har.CacheShort:
			shortCache = append(shortCache, e)
		case har.CacheGood:
			goodCache = append(goodCache, e)
		case har.CacheOverkill:
			overCache = append(overCache, e)
		}
	}

	total := len(entries)
	withCache := len(goodCache) + len(overCache)
	pctGood := 0
	pctWith := 0
	if total > 0 {
		pctGood = (len(goodCache) * 100) / total
		pctWith = (withCache * 100) / total
	}

	if markdown {
		var b strings.Builder
		b.WriteString("# Cache Summary\n\n")
		b.WriteString(mdTable(
			[]string{"Metric", "Value"},
			[][]string{
				{"Total Resources", fmt.Sprintf("%d", total)},
				{"Good Cache", fmt.Sprintf("%d (%d%%)", len(goodCache), pctGood)},
				{"Short Cache", fmt.Sprintf("%d", len(shortCache))},
				{"No Cache", fmt.Sprintf("%d", len(noCache))},
				{"Over-Cached", fmt.Sprintf("%d", len(overCache))},
				{"With Any Caching", fmt.Sprintf("%d (%d%%)", withCache, pctWith)},
			},
		))
		if len(noCache)+len(shortCache) > 0 {
			b.WriteString("\n## Cache Issues\n\n")
			rows := make([][]string, 0, len(noCache)+len(shortCache))
			for _, e := range noCache {
				rows = append(rows, []string{e.Method, e.URLPath, cacheDetail(e)})
			}
			for _, e := range shortCache {
				rows = append(rows, []string{e.Method, e.URLPath, cacheDetail(e)})
			}
			b.WriteString(mdTable([]string{"Method", "URL", "Detail"}, rows))
		}
		if len(goodCache) > 0 {
			b.WriteString("\n## Good Cache\n\n")
			rows := make([][]string, 0, len(goodCache))
			for _, e := range goodCache {
				rows = append(rows, []string{e.Method, e.URLPath, cacheDetail(e)})
			}
			b.WriteString(mdTable([]string{"Method", "URL", "Detail"}, rows))
		}
		if len(overCache) > 0 {
			b.WriteString("\n## Over-Cached\n\n")
			rows := make([][]string, 0, len(overCache))
			for _, e := range overCache {
				rows = append(rows, []string{e.Method, e.URLPath, fmt.Sprintf("max-age=%ds", e.MaxAge)})
			}
			b.WriteString(mdTable([]string{"Method", "URL", "Detail"}, rows))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("Cache Summary\n")
	b.WriteString("=============\n\n")
	fmt.Fprintf(&b, "Total Resources:   %d\n", total)
	fmt.Fprintf(&b, "Good Cache:        %d (%d%%)\n", len(goodCache), pctGood)
	fmt.Fprintf(&b, "Short Cache:       %d\n", len(shortCache))
	fmt.Fprintf(&b, "No Cache:          %d\n", len(noCache))
	fmt.Fprintf(&b, "Over-Cached:       %d\n", len(overCache))
	fmt.Fprintf(&b, "With Any Caching:  %d (%d%%)\n", withCache, pctWith)

	if len(noCache)+len(shortCache) > 0 {
		b.WriteString("\nCache Issues:\n")
		for _, e := range noCache {
			fmt.Fprintf(&b, "  %s %s %s\n", e.Method, e.URLPath, cacheDetail(e))
		}
		for _, e := range shortCache {
			fmt.Fprintf(&b, "  %s %s %s\n", e.Method, e.URLPath, cacheDetail(e))
		}
	}
	if len(goodCache) > 0 {
		b.WriteString("\nGood Cache:\n")
		for _, e := range goodCache {
			fmt.Fprintf(&b, "  %s %s %s\n", e.Method, e.URLPath, cacheDetail(e))
		}
	}
	if len(overCache) > 0 {
		b.WriteString("\nOver-Cached:\n")
		for _, e := range overCache {
			fmt.Fprintf(&b, "  %s %s max-age=%ds\n", e.Method, e.URLPath, e.MaxAge)
		}
	}
	return b.String()
}

func cacheDetail(e *har.EntryDetails) string {
	switch {
	case e.MaxAge >= 0:
		return fmt.Sprintf("max-age=%ds", e.MaxAge)
	case !e.HasCacheHeaders:
		return "no headers"
	case e.ETag != "":
		return "ETag"
	case e.LastModified != "":
		return "Last-Modified"
	case e.Expires != "":
		return "Expires"
	default:
		return "no headers"
	}
}

// securityHeaders lists the security response headers checked per resource.
var securityHeaders = []struct {
	Name        string
	Description string
	Required    bool
}{
	{"strict-transport-security", "HSTS - Enforces HTTPS", true},
	{"x-content-type-options", "Prevents MIME-sniffing", true},
	{"content-security-policy", "CSP - Mitigates XSS", true},
	{"x-frame-options", "Clickjacking protection", true},
	{"referrer-policy", "Controls referrer information", false},
	{"permissions-policy", "Controls browser features", false},
	{"cross-origin-opener-policy", "Isolates browsing context", false},
	{"cross-origin-resource-policy", "Controls cross-origin resources", false},
}

type securityRow struct {
	domain          string
	url             string
	present         map[string]bool
	missingRequired int
	missingOptional int
}

func analyzeSecurity(entries []*har.EntryDetails) []securityRow {
	domainMap := map[string][]*har.EntryDetails{}
	for _, e := range entries {
		if e.Domain == "" {
			continue
		}
		domainMap[e.Domain] = append(domainMap[e.Domain], e)
	}

	rows := make([]securityRow, 0, len(domainMap))
	for domain, ds := range domainMap {
		present := make(map[string]bool)
		for _, h := range securityHeaders {
			present[h.Name] = false
		}
		for _, e := range ds {
			for _, hdr := range e.ResponseHeaders {
				lower := strings.ToLower(hdr.Name)
				for _, sh := range securityHeaders {
					if lower == sh.Name {
						present[sh.Name] = true
					}
				}
			}
		}
		missingRequired := 0
		missingOptional := 0
		for _, sh := range securityHeaders {
			if !present[sh.Name] {
				if sh.Required {
					missingRequired++
				} else {
					missingOptional++
				}
			}
		}
		rows = append(rows, securityRow{
			domain:          domain,
			url:             ds[0].URL,
			present:         present,
			missingRequired: missingRequired,
			missingOptional: missingOptional,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].missingRequired != rows[j].missingRequired {
			return rows[i].missingRequired > rows[j].missingRequired
		}
		return rows[i].domain < rows[j].domain
	})
	return rows
}

// SecurityReport builds a security headers report.
func SecurityReport(entries []*har.EntryDetails, markdown bool) string {
	rows := analyzeSecurity(entries)

	if markdown {
		var b strings.Builder
		b.WriteString("# Security Headers Analysis\n\n")
		tableRows := make([][]string, 0, len(rows))
		for _, r := range rows {
			tableRows = append(tableRows, []string{
				r.domain,
				fmt.Sprintf("%d", r.missingRequired),
				fmt.Sprintf("%d", r.missingOptional),
			})
		}
		b.WriteString(mdTable(
			[]string{"Domain", "Required Missing", "Optional Missing"},
			tableRows,
		))
		for _, r := range rows {
			if r.missingRequired == 0 && r.missingOptional == 0 {
				continue
			}
			fmt.Fprintf(&b, "\n## %s\n\n", r.domain)
			b.WriteString("- URL: " + r.url + "\n\n")
			detail := make([][]string, 0, len(securityHeaders))
			for _, sh := range securityHeaders {
				status := "Present"
				if !r.present[sh.Name] {
					if sh.Required {
						status = "Missing (Required)"
					} else {
						status = "Missing (Optional)"
					}
				}
				detail = append(detail, []string{sh.Name, status})
			}
			b.WriteString(mdTable([]string{"Header", "Status"}, detail))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("Security Headers Analysis\n")
	b.WriteString("=========================\n\n")
	fmt.Fprintf(&b, "%-30s %-10s %s\n", "Domain", "Required", "Optional")
	b.WriteString(strings.Repeat("-", 60) + "\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-30s %-10s %s\n",
			truncate(r.domain, 28),
			fmt.Sprintf("%d missing", r.missingRequired),
			fmt.Sprintf("%d missing", r.missingOptional))
	}
	b.WriteString("\nDetails:\n")
	for _, r := range rows {
		if r.missingRequired == 0 && r.missingOptional == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n  %s\n", r.domain)
		for _, sh := range securityHeaders {
			var status string
            switch {
			case r.present[sh.Name]:
				status = "Present"
                break
			case sh.Required:
				status = "Missing (Required)"
                break
			default:
				status = "Missing (Optional)"
			}
			fmt.Fprintf(&b, "    %-40s %s\n", sh.Name+":", status)
			fmt.Fprintf(&b, "       %s\n", sh.Description)
		}
	}
	return b.String()
}

// TrackersReport builds a third-party tracker analysis report.
func TrackersReport(entries []*har.EntryDetails, markdown bool) string {
	r := AnalyzeTrackers(entries)

	if markdown {
		var b strings.Builder
		b.WriteString("# Third-Party Tracker Analysis\n\n")
		b.WriteString(mdTable(
			[]string{"Metric", "Value"},
			[][]string{
				{"Total Requests", fmt.Sprintf("%d", r.TotalRequests)},
				{"Tracker Requests", fmt.Sprintf("%d (%.1f%%)", r.TrackerRequests, r.TrackerPercent)},
				{"Tracker Size", r.TrackerSizeHuman + " (" + fmt.Sprintf("%.1f%%", r.TrackerSizePercent) + ")"},
			},
		))

		if len(r.ByCategory) > 0 {
			b.WriteString("\n## By Category\n\n")
			type catInfo struct {
				name  string
				count int
			}
			var cats []catInfo
			for name, count := range r.ByCategory {
				cats = append(cats, catInfo{name: name, count: count})
			}
			sort.Slice(cats, func(i, j int) bool { return cats[i].count > cats[j].count })
			rows := make([][]string, 0, len(cats))
			for _, c := range cats {
				rows = append(rows, []string{c.name, fmt.Sprintf("%d", c.count)})
			}
			b.WriteString(mdTable([]string{"Category", "Requests"}, rows))
		}

		if len(r.Domains) > 0 {
			b.WriteString("\n## Top Tracker Domains\n\n")
			rows := make([][]string, 0, len(r.Domains))
			for _, d := range r.Domains {
				rows = append(rows, []string{
					d.Domain,
					string(d.Category),
					fmt.Sprintf("%d", d.Count),
					d.SizeHuman,
				})
			}
			b.WriteString(mdTable([]string{"Domain", "Category", "Requests", "Size"}, rows))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("Third-Party Tracker Analysis\n")
	b.WriteString("============================\n\n")
	fmt.Fprintf(&b, "Total Requests:    %d\n", r.TotalRequests)
	fmt.Fprintf(&b, "Tracker Requests:  %d (%.1f%%)\n", r.TrackerRequests, r.TrackerPercent)
	fmt.Fprintf(&b, "Tracker Size:      %s (%.1f%%)\n", r.TrackerSizeHuman, r.TrackerSizePercent)

	if len(r.ByCategory) > 0 {
		b.WriteString("\nBy Category:\n")
		type catInfo struct {
			name  string
			count int
		}
		var cats []catInfo
		for name, count := range r.ByCategory {
			cats = append(cats, catInfo{name: name, count: count})
		}
		sort.Slice(cats, func(i, j int) bool { return cats[i].count > cats[j].count })
		for _, c := range cats {
			fmt.Fprintf(&b, "  %-18s %d\n", c.name+":", c.count)
		}
	}

	if len(r.Domains) > 0 {
		b.WriteString("\nTop Tracker Domains:\n")
		for _, d := range r.Domains {
			fmt.Fprintf(&b, "  %-30s %-14s %d  %s\n",
				truncate(d.Domain, 28), d.Category, d.Count, d.SizeHuman)
		}
	}
	return b.String()
}

// truncate shortens s to at most max runes, appending "...". Rune-based so
// multi-byte characters are never split.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}
	return string(r[:max-3]) + "..."
}


type auditImageIssue struct {
	Type       string
	ImageURL   string
	PageURL    string
	SizeKB     int64
	Details    string
	Suggestion string
}

type auditImageResult struct {
	Issues []auditImageIssue
}

func getNodeText(n *har.Node, key string) *har.Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Key == key {
			return c
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}

func isInPicture(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && strings.EqualFold(p.Data, "picture") {
			return true
		}
	}
	return false
}

func analyzeImages(root *har.Node, entries []*har.EntryDetails, largeKb int64) auditImageResult {
	var res auditImageResult
	imgSize := map[string]int64{}
	imgMime := map[string]string{}
	for _, d := range entries {
		rt := strings.ToLower(d.ResourceType)
		mt := strings.ToLower(d.MimeType)
		if rt == "image" || strings.HasPrefix(mt, "image/") {
			if d.Size > 0 {
				imgSize[d.URL] = d.Size
			}
			if mt != "" {
				imgMime[d.URL] = mt
			}
		}
	}

	harEntries := har.GetEntries(root)
	for idx, entry := range harEntries {
		resp := getNodeText(entry, "response")
		if resp == nil {
			continue
		}
		content := getNodeText(resp, "content")
		if content == nil {
			continue
		}
		mime := getNodeText(content, "mimeType")
		mimeType := ""
		if mime != nil && mime.Kind == har.String {
			mimeType = strings.ToLower(mime.Value.(string))
		}
		if mimeType != "" && !strings.Contains(mimeType, "text/html") {
			continue
		}
		textNode := getNodeText(content, "text")
		if textNode == nil || textNode.Kind != har.String {
			continue
		}
		htmlText := textNode.Value.(string)
		doc, err := html.Parse(strings.NewReader(htmlText))
		if err != nil {
			continue
		}
		var pageURL string
		if idx < len(entries) {
			pageURL = entries[idx].URL
		}
		var f func(*html.Node)
		imgCount := 0
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && strings.EqualFold(n.Data, "img") {
				imgCount++
				src := getAttr(n, "src")
				if src == "" {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						f(c)
					}
					return
				}
				full := src
				if pageURL != "" {
					if pu, err := url.Parse(pageURL); err == nil {
						if su, err := url.Parse(src); err == nil {
							full = pu.ResolveReference(su).String()
						}
					}
				}
				var sz int64
				if v, ok := imgSize[full]; ok {
					sz = v
				} else if v, ok := imgSize[src]; ok {
					sz = v
				}
				sizeKB := sz / 1024
				isPNG := strings.Contains(strings.ToLower(src), ".png") || strings.Contains(strings.ToLower(full), ".png")
				if !isPNG {
					if m, ok := imgMime[full]; ok && strings.Contains(m, "png") {
						isPNG = true
					} else if m, ok := imgMime[src]; ok && strings.Contains(m, "png") {
						isPNG = true
					} else {
						for _, d := range entries {
							if d.URL == full || d.URL == src {
								if strings.Contains(strings.ToLower(d.MimeType), "png") {
									isPNG = true
									break
								}
							}
						}
					}
				}
				if largeKb == 0 {
					largeKb = 150
				}
				if isPNG && sizeKB > largeKb {
					res.Issues = append(res.Issues, auditImageIssue{
						Type:       "large-png",
						ImageURL:   src,
						PageURL:    pageURL,
						SizeKB:     sizeKB,
						Details:    fmt.Sprintf("Image is a large PNG (%d KB > %d KB)", sizeKB, largeKb),
						Suggestion: "Consider converting to WebP/AVIF or compressing; ensure appropriate formats/sizes are used.",
					})
				}
				hasSrcset := hasAttr(n, "srcset")
				inPicture := isInPicture(n)
				if !inPicture && !hasSrcset {
					res.Issues = append(res.Issues, auditImageIssue{
						Type:       "missing-picture-srcset",
						ImageURL:   src,
						PageURL:    pageURL,
						SizeKB:     sizeKB,
						Details:    "Image is not in a <picture> element and has no srcset attribute",
						Suggestion: "Use <picture> with modern formats (WebP/AVIF) or add srcset for responsive images.",
					})
				}
				loading := strings.ToLower(getAttr(n, "loading"))
				if loading == "lazy" {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						f(c)
					}
					return
				}
				if loading == "eager" {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						f(c)
					}
					return
				}
				// Check if image is likely above the fold / critical
				skipLazy := false
				if imgCount <= 5 {
					skipLazy = true
				}
				if getAttr(n, "fetchpriority") == "high" {
					skipLazy = true
				}
				cls := strings.ToLower(getAttr(n, "class"))
				id := strings.ToLower(getAttr(n, "id"))
				alt := strings.ToLower(getAttr(n, "alt"))
				if strings.Contains(cls, "logo") || strings.Contains(id, "logo") || strings.Contains(alt, "logo") || strings.Contains(src, "logo") {
					skipLazy = true
				}
				if strings.Contains(cls, "hero") || strings.Contains(cls, "banner") || strings.Contains(cls, "header") {
					skipLazy = true
				}
				// Check if in header/nav
				for p := n.Parent; p != nil; p = p.Parent {
					if p.Type == html.ElementNode {
						tag := strings.ToLower(p.Data)
						if tag == "header" || tag == "nav" {
							skipLazy = true
							break
						}
						if tag == "body" {
							break
						}
					}
				}
				if skipLazy {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						f(c)
					}
					return
				}
				res.Issues = append(res.Issues, auditImageIssue{
					Type:       "missing-loading-lazy",
					ImageURL:   src,
					PageURL:    pageURL,
					SizeKB:     sizeKB,
					Details:    "Image does not have loading=\"lazy\" and is likely offscreen on page load",
					Suggestion: "Add loading=\"lazy\" for offscreen images to improve initial page load performance.",
				})
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
	}
	return res
}

// AuditImagesReport builds an image audit report.
func AuditImagesReport(root *har.Node, entries []*har.EntryDetails, largeKb int64, markdown bool) string {
	result := analyzeImages(root, entries, largeKb)
	if markdown {
		var b strings.Builder
		b.WriteString("# Image Audit\n\n")
		if len(result.Issues) == 0 {
			b.WriteString("No issues found.\n")
			return b.String()
		}
		for _, issue := range result.Issues {
			fmt.Fprintf(&b, "## Issue: %s\n\n", issue.Type)
			fmt.Fprintf(&b, "- **URL:** %s\n", issue.ImageURL)
			if issue.PageURL != "" {
				fmt.Fprintf(&b, "- **Page:** %s\n", issue.PageURL)
			}
			if issue.SizeKB >= 0 {
				fmt.Fprintf(&b, "- **Size:** %d KB\n", issue.SizeKB)
			}
			fmt.Fprintf(&b, "- **Details:** %s\n\n", issue.Details)
			if issue.Suggestion != "" {
				fmt.Fprintf(&b, "**Suggestion:** %s\n\n", issue.Suggestion)
			}
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("Image Audit\n\n")
	if len(result.Issues) == 0 {
		b.WriteString("No issues found.\n")
		return b.String()
	}
	for i, issue := range result.Issues {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, issue.Type)
		fmt.Fprintf(&b, "    Image: %s\n", issue.ImageURL)
		if issue.PageURL != "" {
			fmt.Fprintf(&b, "    Page:  %s\n", issue.PageURL)
		}
		if issue.SizeKB >= 0 {
			fmt.Fprintf(&b, "    Size:  %d KB\n", issue.SizeKB)
		}
		fmt.Fprintf(&b, "    %s\n", issue.Details)
		if issue.Suggestion != "" {
			fmt.Fprintf(&b, "    Suggestion: %s\n", issue.Suggestion)
		}
		b.WriteString("\n")
	}
	return b.String()
}

type redirectHop struct {
	Index       int
	Method      string
	Status      int
	URL         string
	RedirectURL string
	Time        float64
	Started     string
	Wait        float64
	DNS         float64
	Connect     float64
	SSL         float64
	Send        float64
	Receive     float64
	Blocked     float64
}

type redirectChain struct {
	Hops []redirectHop
}

type auditRedirectsResult struct {
	Chains []redirectChain
}

func analyzeRedirects(root *har.Node, entries []*har.EntryDetails) auditRedirectsResult {
	var res auditRedirectsResult
	if root == nil {
		return res
	}
	logNode := findChildNode(root, "log")
	if logNode == nil {
		return res
	}
	entriesNode := findChildNode(logNode, "entries")
	if entriesNode == nil || entriesNode.Kind != har.Array {
		return res
	}
	// Build lookup by request URL
	byURL := make(map[string]int)
	for i, e := range entries {
		if e != nil && e.URL != "" {
			if _, exists := byURL[e.URL]; !exists {
				byURL[e.URL] = i
			}
		}
	}
	seen := make(map[int]bool)
	for idx, entry := range entriesNode.Children {
		if seen[idx] {
			continue
		}
		resp := findChildNode(entry, "response")
		if resp == nil {
			continue
		}
		statusNode := findChildNode(resp, "status")
		if statusNode == nil || statusNode.Kind != har.Number {
			continue
		}
		status := int(statusNode.Value.(float64))
		if status < 300 || status >= 400 {
			continue
		}
		redirectNode := findChildNode(resp, "redirectURL")
		if redirectNode == nil || redirectNode.Kind != har.String || redirectNode.Value.(string) == "" {
			continue
		}
		// Start a chain
		chain := redirectChain{}
		curIdx := idx
		for depth := 0; depth < 20 && curIdx >= 0 && curIdx < len(entriesNode.Children) && !seen[curIdx]; depth++ {
			curEntry := entriesNode.Children[curIdx]
			resp := findChildNode(curEntry, "response")
			req := findChildNode(curEntry, "request")
		status := 0
		redirectURL := ""
		if resp != nil {
			if sn := findChildNode(resp, "status"); sn != nil && sn.Kind == har.Number {
				status = int(sn.Value.(float64))
			}
			if rn := findChildNode(resp, "redirectURL"); rn != nil && rn.Kind == har.String {
				redirectURL = rn.Value.(string)
			}
		}
		method, curURL := "", ""
		if req != nil {
			if m := findChildNode(req, "method"); m != nil && m.Kind == har.String {
				method = m.Value.(string)
			}
			if u := findChildNode(req, "url"); u != nil && u.Kind == har.String {
				curURL = u.Value.(string)
			}
		}
		hop := redirectHop{
			Index:       curIdx + 1,
			Method:      method,
			Status:      status,
			URL:         curURL,
			RedirectURL: redirectURL,
		}
		if curIdx < len(entries) && entries[curIdx] != nil {
			d := entries[curIdx]
			hop.Time = d.Time
			hop.Started = d.Started
			hop.Wait = d.Wait
			hop.DNS = d.DNS
			hop.Connect = d.Connect
			hop.SSL = d.SSL
			hop.Send = d.Send
			hop.Receive = d.Receive
			hop.Blocked = d.Blocked
		}
		chain.Hops = append(chain.Hops, hop)
		seen[curIdx] = true
		if status < 300 || status >= 400 || redirectURL == "" {
			break
		}
		// resolve redirectURL: if relative, try joining with current URL
		nextURL := redirectURL
		if !strings.HasPrefix(nextURL, "http://") && !strings.HasPrefix(nextURL, "https://") {
			if pu, err := url.Parse(curURL); err == nil {
				if ru, err := url.Parse(nextURL); err == nil {
					nextURL = pu.ResolveReference(ru).String()
				}
			}
		}
		nextIdx, ok := byURL[nextURL]
			if !ok {
				// try raw redirectURL
				nextIdx, ok = byURL[redirectURL]
			}
			if !ok {
				break
			}
			curIdx = nextIdx
		}
		if len(chain.Hops) > 0 {
			res.Chains = append(res.Chains, chain)
		}
	}
	return res
}

// AuditRedirectsReport builds a redirects audit report.
func AuditRedirectsReport(root *har.Node, entries []*har.EntryDetails, markdown bool) string {
	result := analyzeRedirects(root, entries)
	if markdown {
		var b strings.Builder
		b.WriteString("# Redirect Chains\n\n")
		if len(result.Chains) == 0 {
			b.WriteString("No redirects found.\n")
			return b.String()
		}
		for i, chain := range result.Chains {
			fmt.Fprintf(&b, "## Chain %d (%d hops)\n\n", i+1, len(chain.Hops))
			for j, h := range chain.Hops {
				fmt.Fprintf(&b, "**Hop %d** (entry #%d) `%s %d`\n\n", j+1, h.Index, h.Method, h.Status)
				fmt.Fprintf(&b, "- URL: %s\n", h.URL)
				if h.RedirectURL != "" {
					fmt.Fprintf(&b, "- Redirect: %s\n", h.RedirectURL)
				}
				if h.Started != "" {
					fmt.Fprintf(&b, "- Started: %s\n", h.Started)
				}
				fmt.Fprintf(&b, "- Timing: Total %.0fms | Blocked %.0fms | DNS %.0fms | Connect %.0fms | SSL %.0fms | Send %.0fms | Wait %.0fms | Receive %.0fms\n\n",
					h.Time, h.Blocked, h.DNS, h.Connect, h.SSL, h.Send, h.Wait, h.Receive)
			}
		}
		return b.String()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Redirect chains: %d\n\n", len(result.Chains))
	if len(result.Chains) == 0 {
		b.WriteString("No redirects found.\n")
		return b.String()
	}
	for i, chain := range result.Chains {
		fmt.Fprintf(&b, "Chain %d (%d hops)\n", i+1, len(chain.Hops))
		for j, h := range chain.Hops {
			prefix := "     "
			if j == 0 {
				prefix = "  -> "
			}
			fmt.Fprintf(&b, "%s[%d] %s %d  %s\n", prefix, h.Index, h.Method, h.Status, h.URL)
			fmt.Fprintf(&b, "       Total %.0fms | Blocked %.0fms | DNS %.0fms | Connect %.0fms | SSL %.0fms | Send %.0fms | Wait %.0fms | Receive %.0fms\n",
				h.Time, h.Blocked, h.DNS, h.Connect, h.SSL, h.Send, h.Wait, h.Receive)
			if h.RedirectURL != "" {
				fmt.Fprintf(&b, "       Redirect: %s\n", h.RedirectURL)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func priorityChanges(entries []*har.EntryDetails) []*har.EntryDetails {
	changed := make([]*har.EntryDetails, 0, len(entries))
	for _, e := range entries {
		if e.Priority == "" || e.InitialPriority == "" {
			continue
		}
		if e.Priority != e.InitialPriority {
			changed = append(changed, e)
		}
	}
	sort.Slice(changed, func(i, j int) bool { return changed[i].URL < changed[j].URL })
	return changed
}


// PriorityReport builds a report of requests with changed resource priority.
func PriorityReport(entries []*har.EntryDetails, markdown bool) string {
	changed := priorityChanges(entries)
	if len(changed) == 0 {
		if markdown {
			return "# Priority Changes\n\nNo requests had a priority change.\n"
		}
		return "Requests with a priority change: 0\n\nNo requests had a priority change.\n"
	}
	if markdown {
		var b strings.Builder
		b.WriteString("# Priority Changes\n\n")
		headers := []string{"Method", "Version", "Status", "Initial", "Priority", "URL"}
		rows := make([][]string, 0, len(changed))
		for _, e := range changed {
			rows = append(rows, []string{e.Method, e.HTTPVersion, fmt.Sprintf("%d", e.Status), e.InitialPriority, e.Priority, e.URL})
		}
		b.WriteString(mdTable(headers, rows))
		return b.String()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Requests with a priority change: %d\n\n", len(changed))
	for _, e := range changed {
		fmt.Fprintf(&b, "  %-6s %-8s %-3d %-10s -> %s\n", e.Method, e.HTTPVersion, e.Status, e.InitialPriority, e.Priority)
		fmt.Fprintf(&b, "    %s\n", e.URL)
	}
	return b.String()
}
