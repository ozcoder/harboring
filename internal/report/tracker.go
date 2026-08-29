package report

import (
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/harboring/harboring/internal/har"
)

// TrackerCategory classifies a tracking domain.
type TrackerCategory string

const (
	TrackerAnalytics      TrackerCategory = "analytics"
	TrackerAdvertising    TrackerCategory = "advertising"
	TrackerSocial         TrackerCategory = "social"
	TrackerFingerprinting TrackerCategory = "fingerprinting"
	TrackerOther          TrackerCategory = "other"
)

// trackerEntry maps a base domain to its category.
type trackerEntry struct {
	domain   string
	category TrackerCategory
}

// knownTrackers is a curated list of well-known third-party tracking domains
// based on EasyPrivacy and Disconnect lists. Suffix matching is used: a request
// domain is a tracker if it equals a known domain or is a subdomain of one.
var knownTrackers = []trackerEntry{
	// Analytics
	{domain: "google-analytics.com", category: TrackerAnalytics},
	{domain: "googletagmanager.com", category: TrackerAnalytics},
	{domain: "analytics.google.com", category: TrackerAnalytics},
	{domain: "hotjar.com", category: TrackerAnalytics},
	{domain: "mixpanel.com", category: TrackerAnalytics},
	{domain: "segment.io", category: TrackerAnalytics},
	{domain: "segment.com", category: TrackerAnalytics},
	{domain: "amplitude.com", category: TrackerAnalytics},
	{domain: "fullstory.com", category: TrackerAnalytics},
	{domain: "heap.io", category: TrackerAnalytics},
	{domain: "mouseflow.com", category: TrackerAnalytics},
	{domain: "crazyegg.com", category: TrackerAnalytics},
	{domain: "newrelic.com", category: TrackerAnalytics},
	{domain: "nr-data.net", category: TrackerAnalytics},
	{domain: "chartbeat.com", category: TrackerAnalytics},
	{domain: "chartbeat.net", category: TrackerAnalytics},
	{domain: "optimizely.com", category: TrackerAnalytics},
	{domain: "parse.ly", category: TrackerAnalytics},
	{domain: "parsely.com", category: TrackerAnalytics},
	{domain: "indicative.com", category: TrackerAnalytics},
	{domain: "klaviyo.com", category: TrackerAnalytics},
	{domain: "sentry.io", category: TrackerAnalytics},
	{domain: "bugsnag.com", category: TrackerAnalytics},
	{domain: "instana.io", category: TrackerAnalytics},
	{domain: "datadoghq.com", category: TrackerAnalytics},
	{domain: "demdex.net", category: TrackerAnalytics},

	// Advertising
	{domain: "googlesyndication.com", category: TrackerAdvertising},
	{domain: "doubleclick.net", category: TrackerAdvertising},
	{domain: "googleadservices.com", category: TrackerAdvertising},
	{domain: "googleadsserving.cn", category: TrackerAdvertising},
	{domain: "amazon-adsystem.com", category: TrackerAdvertising},
	{domain: "media.net", category: TrackerAdvertising},
	{domain: "taboola.com", category: TrackerAdvertising},
	{domain: "outbrain.com", category: TrackerAdvertising},
	{domain: "criteo.com", category: TrackerAdvertising},
	{domain: "criteo.net", category: TrackerAdvertising},
	{domain: "adnxs.com", category: TrackerAdvertising},
	{domain: "pubmatic.com", category: TrackerAdvertising},
	{domain: "rubiconproject.com", category: TrackerAdvertising},
	{domain: "openx.net", category: TrackerAdvertising},
	{domain: "indexexchange.com", category: TrackerAdvertising},
	{domain: "sharethrough.com", category: TrackerAdvertising},
	{domain: "casalemedia.com", category: TrackerAdvertising},
	{domain: "adsrvr.org", category: TrackerAdvertising},
	{domain: "bidswitch.net", category: TrackerAdvertising},
	{domain: "moatads.com", category: TrackerAdvertising},
	{domain: "spotxchange.com", category: TrackerAdvertising},
	{domain: "turn.com", category: TrackerAdvertising},
	{domain: "advertising.com", category: TrackerAdvertising},
	{domain: "simpli.fi", category: TrackerAdvertising},
	{domain: "adroll.com", category: TrackerAdvertising},
	{domain: "liadm.com", category: TrackerAdvertising},
	{domain: "blueconic.net", category: TrackerAdvertising},
	{domain: "eyeota.net", category: TrackerAdvertising},
	{domain: "mediaplex.com", category: TrackerAdvertising},
	{domain: "exelator.com", category: TrackerAdvertising},
	{domain: "rlcdn.com", category: TrackerAdvertising},
	{domain: "id5-sync.com", category: TrackerAdvertising},
	{domain: "crwdcntrl.net", category: TrackerAdvertising},
	{domain: "bounceexchange.com", category: TrackerAdvertising},
	{domain: "agkn.com", category: TrackerAdvertising},
	{domain: "ipify.org", category: TrackerAdvertising},

	// Social
	{domain: "facebook.com", category: TrackerSocial},
	{domain: "facebook.net", category: TrackerSocial},
	{domain: "fbcdn.net", category: TrackerSocial},
	{domain: "twitter.com", category: TrackerSocial},
	{domain: "twimg.com", category: TrackerSocial},
	{domain: "platform.twitter.com", category: TrackerSocial},
	{domain: "linkedin.com", category: TrackerSocial},
	{domain: "licdn.com", category: TrackerSocial},
	{domain: "pinterest.com", category: TrackerSocial},
	{domain: "pinimg.com", category: TrackerSocial},
	{domain: "instagram.com", category: TrackerSocial},
	{domain: "addthis.com", category: TrackerSocial},
	{domain: "sharethis.com", category: TrackerSocial},
	{domain: "snap.licdn.com", category: TrackerSocial},
	{domain: "pixel.facebook.com", category: TrackerSocial},

	// Fingerprinting
	{domain: "fingerprint.com", category: TrackerFingerprinting},
	{domain: "fpjs.io", category: TrackerFingerprinting},
	{domain: "fpremoteconfig.com", category: TrackerFingerprinting},
	{domain: "halfminor.com", category: TrackerFingerprinting},
	{domain: "brott.co", category: TrackerFingerprinting},

	// Other tracking
	{domain: "yieldmo.com", category: TrackerOther},
	{domain: "branch.io", category: TrackerOther},
	{domain: "adjust.com", category: TrackerOther},
	{domain: "appsflyer.com", category: TrackerOther},
	{domain: "branch.io", category: TrackerOther},
	{domain: "hasoffers.com", category: TrackerOther},
	{domain: "kochava.com", category: TrackerOther},
	{domain: "singular.net", category: TrackerOther},
	{domain: "tenjin.com", category: TrackerOther},
	{domain: "traction.com", category: TrackerOther},
}

// TrackerDef describes a tracker domain and its category. It is used for
// user-defined trackers supplied via --trackers-file.
type TrackerDef struct {
	Domain   string          `toml:"domain"`
	Category TrackerCategory `toml:"category"`
}

// trackerDomainSet holds the effective tracker lookup map, including the
// built-in list plus any user-supplied trackers. It is guarded by a mutex so a
// --trackers-file can be loaded before analysis at runtime.
var (
	trackerMu        sync.RWMutex
	trackerDomainSet map[string]TrackerCategory
)

// TrackerListFile is the TOML shape accepted by --trackers-file.
type TrackerListFile struct {
	Trackers []TrackerDef `toml:"trackers"`
}

func init() {
	trackerDomainSet = make(map[string]TrackerCategory, len(knownTrackers))
	for _, t := range knownTrackers {
		trackerDomainSet[t.domain] = t.category
	}
}

// AddTrackers merges user-defined tracker domains into the lookup set. The
// built-in list is always included; user entries are additive and override the
// category if the same domain is already known.
func AddTrackers(defs []TrackerDef) {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	for _, d := range defs {
		domain := strings.ToLower(strings.TrimSpace(d.Domain))
		if domain == "" {
			continue
		}
		cat := d.Category
		if cat == "" {
			cat = TrackerOther
		}
		trackerDomainSet[domain] = cat
	}
}

// LoadTrackersFile loads a TOML tracker list from path and adds its entries to
// the effective tracker set. A missing or empty file contributes nothing.
func LoadTrackersFile(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var file TrackerListFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return err
	}
	AddTrackers(file.Trackers)
	return nil
}

// KnownTrackers returns the list of tracker entries currently in the effective
// set (built-in plus any user-supplied), sorted by domain.
func KnownTrackers() []TrackerDef {
	trackerMu.RLock()
	defer trackerMu.RUnlock()
	out := make([]TrackerDef, 0, len(trackerDomainSet))
	for domain, cat := range trackerDomainSet {
		out = append(out, TrackerDef{Domain: domain, Category: cat})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Domain < out[j].Domain })
	return out
}

// IsTracker reports whether domain matches a known tracker using suffix matching.
// For example, "www.google-analytics.com" matches "google-analytics.com".
func IsTracker(domain string) (TrackerCategory, bool) {
	d := strings.ToLower(domain)
	trackerMu.RLock()
	category, ok := trackerDomainSet[d]
	trackerMu.RUnlock()
	if ok {
		return category, true
	}
	// suffix match: check if domain is a subdomain of a known tracker
	trackerMu.RLock()
	defer trackerMu.RUnlock()
	for trackerDomain, cat := range trackerDomainSet {
		if strings.HasSuffix(d, "."+trackerDomain) {
			return cat, true
		}
	}
	return "", false
}

// TrackerDomain holds per-domain tracker analysis results.
type TrackerDomain struct {
	Domain    string          `json:"domain"`
	Category  TrackerCategory `json:"category"`
	Count     int             `json:"count"`
	Size      int64           `json:"size"`
	SizeHuman string          `json:"size_human"`
}

// TrackerResult holds the complete tracker analysis for a set of entries.
type TrackerResult struct {
	TotalRequests      int             `json:"-"`
	TrackerRequests    int             `json:"tracker_requests"`
	TrackerPercent     float64         `json:"tracker_percent"`
	TrackerSize        int64           `json:"tracker_size"`
	TrackerSizeHuman   string          `json:"tracker_size_human"`
	TrackerSizePercent float64         `json:"tracker_size_percent"`
	ByCategory         map[string]int  `json:"by_category"`
	Domains            []TrackerDomain `json:"domains"`
}

// AnalyzeTrackers scans entries and returns tracker analysis results.
func AnalyzeTrackers(entries []*har.EntryDetails) TrackerResult {
	result := TrackerResult{
		TotalRequests: len(entries),
		ByCategory:    make(map[string]int),
	}

	type domainInfo struct {
		category TrackerCategory
		count    int
		size     int64
	}
	domainMap := map[string]*domainInfo{}

	for _, e := range entries {
		cat, isTracker := IsTracker(e.Domain)
		if !isTracker {
			continue
		}
		result.TrackerRequests++
		result.TrackerSize += e.Size
		result.ByCategory[string(cat)]++

		di, ok := domainMap[e.Domain]
		if !ok {
			di = &domainInfo{category: cat}
			domainMap[e.Domain] = di
		}
		di.count++
		di.size += e.Size
	}

	var totalSize int64
	for _, e := range entries {
		totalSize += e.Size
	}
	if len(entries) > 0 {
		result.TrackerPercent = float64(result.TrackerRequests) / float64(len(entries)) * 100
	}
	if totalSize > 0 {
		result.TrackerSizePercent = float64(result.TrackerSize) / float64(totalSize) * 100
	}
	result.TrackerSizeHuman = formatBytes(result.TrackerSize)

	result.Domains = make([]TrackerDomain, 0, len(domainMap))
	for domain, di := range domainMap {
		result.Domains = append(result.Domains, TrackerDomain{
			Domain:    domain,
			Category:  di.category,
			Count:     di.count,
			Size:      di.size,
			SizeHuman: formatBytes(di.size),
		})
	}
	sort.Slice(result.Domains, func(i, j int) bool {
		return result.Domains[i].Size > result.Domains[j].Size
	})

	return result
}
