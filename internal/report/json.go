package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/harboring/harboring/internal/har"
)
// --- Stats JSON ---

type StatsJSON struct {
	Requests       int                `json:"requests"`
	TotalSize      int64              `json:"total_size"`
	TotalSizeHuman string             `json:"total_size_human"`
	TotalTimeMs    float64            `json:"total_time_ms"`
	Methods        map[string]int     `json:"methods"`
	StatusCodes    map[string]int     `json:"status_codes"`
	Slowest        []StatsJSONRequest `json:"slowest"`
}

type StatsJSONRequest struct {
	Method string  `json:"method"`
	URL    string  `json:"url"`
	TimeMs float64 `json:"time_ms"`
}

func StatsJSONReport(entries []*har.EntryDetails) ([]byte, error) {
	var totalSize int64
	var totalTime float64
	methods := map[string]int{}
	statuses := map[int]int{}

	type timedEntry struct {
		method string
		url    string
		time   float64
	}
	slowest := make([]timedEntry, 0, len(entries))

	for _, d := range entries {
		totalSize += d.Size
		totalTime += d.Time
		methods[d.Method]++
		statuses[d.Status]++
		slowest = append(slowest, timedEntry{method: d.Method, url: d.URL, time: d.Time})
	}
	sort.Slice(slowest, func(i, j int) bool { return slowest[i].time > slowest[j].time })
	limit := 5
	if len(slowest) < limit {
		limit = len(slowest)
	}
	statusCodes := map[string]int{}
	for _, r := range statusRows(statuses) {
		statusCodes[r[0]] = int(parseInt64(r[1]))
	}
	out := StatsJSON{
		Requests:       len(entries),
		TotalSize:      totalSize,
		TotalSizeHuman: formatBytes(totalSize),
		TotalTimeMs:    totalTime,
		Methods:        methods,
		StatusCodes:    statusCodes,
		Slowest:        make([]StatsJSONRequest, limit),
	}
	for i := 0; i < limit; i++ {
		out.Slowest[i] = StatsJSONRequest{
			Method: slowest[i].method,
			URL:    slowest[i].url,
			TimeMs: slowest[i].time,
		}
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- Perf JSON ---

type PerfJSON struct {
	Requests        int               `json:"requests"`
	TotalSize       int64             `json:"total_size"`
	TotalSizeHuman  string            `json:"total_size_human"`
	TotalTimeMs     float64           `json:"total_time_ms"`
	TTFBPercentiles PercentilesJSON   `json:"ttfb_percentiles"`
	ByResourceType  map[string]int64  `json:"by_resource_type"`
	ByDomain        map[string]int64  `json:"by_domain"`
	StatusCodes     map[string]int    `json:"status_codes"`
	Largest         *PerfJSONRequest  `json:"largest"`
	Slowest         *PerfJSONRequest  `json:"slowest"`
	LargeFonts      []PerfJSONRequest `json:"large_fonts,omitempty"`
}

type PercentilesJSON struct {
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

type PerfJSONRequest struct {
	Method    string  `json:"method"`
	URL       string  `json:"url"`
	Size      int64   `json:"size"`
	SizeHuman string  `json:"size_human"`
	TimeMs    float64 `json:"time_ms"`
}

func PerfJSONReport(entries []*har.EntryDetails) ([]byte, error) {
	var totalSize int64
	var totalTime float64
	var ttfbs []float64
	byType := map[string]int64{}
	byDomain := map[string]int64{}
	byStatus := map[int]int{}
	var largest, slowest *har.EntryDetails
	var largeFonts []PerfJSONRequest

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
			largeFonts = append(largeFonts, PerfJSONRequest{
				Method:    d.Method,
				URL:       d.URL,
				Size:      d.Size,
				SizeHuman: formatBytes(d.Size),
				TimeMs:    d.Time,
			})
		}
	}

	sort.Float64s(ttfbs)

	out := PerfJSON{
		Requests:       len(entries),
		TotalSize:      totalSize,
		TotalSizeHuman: formatBytes(totalSize),
		TotalTimeMs:    totalTime,
		TTFBPercentiles: PercentilesJSON{
			P50: percentile(ttfbs, 50),
			P90: percentile(ttfbs, 90),
			P95: percentile(ttfbs, 95),
			P99: percentile(ttfbs, 99),
		},
		ByResourceType: byType,
		ByDomain:       byDomain,
		StatusCodes:    map[string]int{},
	}
	for _, r := range statusRows(byStatus) {
		out.StatusCodes[r[0]] = int(parseInt64(r[1]))
	}
	if largest != nil {
		out.Largest = &PerfJSONRequest{
			Method:    largest.Method,
			URL:       largest.URL,
			Size:      largest.Size,
			SizeHuman: formatBytes(largest.Size),
			TimeMs:    largest.Time,
		}
	}
	if slowest != nil {
		out.Slowest = &PerfJSONRequest{
			Method:    slowest.Method,
			URL:       slowest.URL,
			Size:      slowest.Size,
			SizeHuman: formatBytes(slowest.Size),
			TimeMs:    slowest.Time,
		}
	}
	if len(largeFonts) > 0 {
		out.LargeFonts = largeFonts
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- Cache JSON ---

type CacheJSON struct {
	Total            int              `json:"total"`
	GoodCache        int              `json:"good_cache"`
	GoodCachePercent int              `json:"good_cache_percent"`
	ShortCache       int              `json:"short_cache"`
	NoCache          int              `json:"no_cache"`
	OverCached       int              `json:"over_cached"`
	WithAnyCaching   int              `json:"with_any_caching"`
	WithPercent      int              `json:"with_any_caching_percent"`
	Issues           []CacheJSONEntry `json:"issues"`
	Good             []CacheJSONEntry `json:"good"`
	OverCachedDetail []CacheJSONEntry `json:"over_cached_detail"`
}

type CacheJSONEntry struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

func CacheJSONReport(entries []*har.EntryDetails) ([]byte, error) {
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

	entryList := func(list []*har.EntryDetails) []CacheJSONEntry {
		out := make([]CacheJSONEntry, len(list))
		for i, e := range list {
			out[i] = CacheJSONEntry{Method: e.Method, URL: e.URLPath, Detail: cacheDetail(e)}
		}
		return out
	}

	out := CacheJSON{
		Total:            total,
		GoodCache:        len(goodCache),
		GoodCachePercent: pctGood,
		ShortCache:       len(shortCache),
		NoCache:          len(noCache),
		OverCached:       len(overCache),
		WithAnyCaching:   withCache,
		WithPercent:      pctWith,
		Issues:           entryList(append(noCache, shortCache...)),
		Good:             entryList(goodCache),
		OverCachedDetail: entryList(overCache),
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- Security JSON ---

type SecurityJSON struct {
	Domains []SecurityJSONDomain `json:"domains"`
}

type SecurityJSONDomain struct {
	Domain          string          `json:"domain"`
	URL             string          `json:"url"`
	MissingRequired int             `json:"missing_required"`
	MissingOptional int             `json:"missing_optional"`
	Headers         map[string]bool `json:"headers"`
}

func SecurityJSONReport(entries []*har.EntryDetails) ([]byte, error) {
	rows := analyzeSecurity(entries)
	out := SecurityJSON{
		Domains: make([]SecurityJSONDomain, len(rows)),
	}
	for i, r := range rows {
		out.Domains[i] = SecurityJSONDomain{
			Domain:          r.domain,
			URL:             r.url,
			MissingRequired: r.missingRequired,
			MissingOptional: r.missingOptional,
			Headers:         r.present,
		}
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- Trackers JSON ---

func TrackersJSONReport(entries []*har.EntryDetails) ([]byte, error) {
	r := AnalyzeTrackers(entries)
	return json.MarshalIndent(r, "", "  ")
}

// --- Priority Changes JSON ---

type PriorityJSON struct {
	Total   int                 `json:"total"`
	Changes []PriorityJSONEntry `json:"changes"`
}

type PriorityJSONEntry struct {
	URL             string `json:"url"`
	Method          string `json:"method"`
	HTTPVersion     string `json:"http_version"`
	Status          int    `json:"status"`
	InitialPriority string `json:"initial_priority"`
	Priority        string `json:"priority"`
}

// --- Audit Images JSON ---

type AuditImagesJSON struct {
	Issues []AuditImageIssueJSON `json:"issues"`
}

type AuditImageIssueJSON struct {
	Type       string `json:"type"`
	ImageURL   string `json:"image_url"`
	PageURL    string `json:"page_url,omitempty"`
	SizeKB     int64  `json:"size_kb,omitempty"`
	Details    string `json:"details"`
	Suggestion string `json:"suggestion,omitempty"`
}

func AuditImagesJSONReport(root *har.Node, entries []*har.EntryDetails, largeKb int64) ([]byte, error) {
	result := analyzeImages(root, entries, largeKb)
	out := AuditImagesJSON{
		Issues: make([]AuditImageIssueJSON, 0, len(result.Issues)),
	}
	for _, issue := range result.Issues {
		out.Issues = append(out.Issues, AuditImageIssueJSON{
			Type:       issue.Type,
			ImageURL:   issue.ImageURL,
			PageURL:    issue.PageURL,
			SizeKB:     issue.SizeKB,
			Details:    issue.Details,
			Suggestion: issue.Suggestion,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}

func PriorityJSONReport(entries []*har.EntryDetails) ([]byte, error) {
	changed := priorityChanges(entries)
	out := PriorityJSON{
		Total:   len(changed),
		Changes: make([]PriorityJSONEntry, 0, len(changed)),
	}
	for _, e := range changed {
		out.Changes = append(out.Changes, PriorityJSONEntry{
			URL:             e.URL,
			Method:          e.Method,
			HTTPVersion:     e.HTTPVersion,
			Status:          e.Status,
			InitialPriority: e.InitialPriority,
			Priority:        e.Priority,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- Audit Redirects JSON ---

type AuditRedirectsJSON struct {
	Chains []AuditRedirectChainJSON `json:"chains"`
}

type AuditRedirectChainJSON struct {
	Hops []AuditRedirectHopJSON `json:"hops"`
}

type AuditRedirectHopJSON struct {
	Index       int     `json:"index"`
	Method      string  `json:"method"`
	Status      int     `json:"status"`
	URL         string  `json:"url"`
	RedirectURL string  `json:"redirect_url"`
	Time        float64 `json:"time_ms"`
	Started     string  `json:"started,omitempty"`
	Wait        float64 `json:"wait_ms"`
	DNS         float64 `json:"dns_ms"`
	Connect     float64 `json:"connect_ms"`
	SSL         float64 `json:"ssl_ms"`
	Send        float64 `json:"send_ms"`
	Receive     float64 `json:"receive_ms"`
	Blocked     float64 `json:"blocked_ms"`
}

func AuditRedirectsJSONReport(root *har.Node, entries []*har.EntryDetails) ([]byte, error) {
	res := analyzeRedirects(root, entries)
	out := AuditRedirectsJSON{
		Chains: make([]AuditRedirectChainJSON, 0, len(res.Chains)),
	}
	for _, c := range res.Chains {
		chain := AuditRedirectChainJSON{Hops: make([]AuditRedirectHopJSON, 0, len(c.Hops))}
		for _, h := range c.Hops {
			chain.Hops = append(chain.Hops, AuditRedirectHopJSON{
				Index:       h.Index,
				Method:      h.Method,
				Status:      h.Status,
				URL:         h.URL,
				RedirectURL: h.RedirectURL,
				Time:        h.Time,
				Started:     h.Started,
				Wait:        h.Wait,
				DNS:         h.DNS,
				Connect:     h.Connect,
				SSL:         h.SSL,
				Send:        h.Send,
				Receive:     h.Receive,
				Blocked:     h.Blocked,
			})
		}
		out.Chains = append(out.Chains, chain)
	}
	return json.MarshalIndent(out, "", "  ")
}

func hasRequestHeader(d *har.EntryDetails, name string) bool {
	lower := strings.ToLower(name)
	for _, h := range d.RequestHeaders {
		if strings.ToLower(h.Name) == lower {
			return true
		}
	}
	return false
}

func parseInt64(s string) int64 {
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return 0
	}
	return v
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	k := (p / 100.0) * (float64(len(sorted)) - 1)
	lo := int(k)
	if lo < 0 {
		lo = 0
	}
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	f := k - float64(lo)
	return sorted[lo] + f*(sorted[hi]-sorted[lo])
}

func statusRows(statuses map[int]int) [][]string {
	keys := make([]int, 0, len(statuses))
	for k := range statuses {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{fmt.Sprintf("%d", k), fmt.Sprintf("%d", statuses[k])})
	}
	return rows
}

func methodRows(methods map[string]int) [][]string {
	keys := make([]string, 0, len(methods))
	for k := range methods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, fmt.Sprintf("%d", methods[k])})
	}
	return rows
}
