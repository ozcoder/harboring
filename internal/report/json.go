package report

import (
	"encoding/json"
	"sort"

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
	Requests        int              `json:"requests"`
	TotalSize       int64            `json:"total_size"`
	TotalSizeHuman  string           `json:"total_size_human"`
	TotalTimeMs     float64          `json:"total_time_ms"`
	TTFBPercentiles PercentilesJSON  `json:"ttfb_percentiles"`
	ByResourceType  map[string]int64 `json:"by_resource_type"`
	ByDomain        map[string]int64 `json:"by_domain"`
	StatusCodes     map[string]int   `json:"status_codes"`
	Largest         *PerfJSONRequest `json:"largest"`
	Slowest         *PerfJSONRequest `json:"slowest"`
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
