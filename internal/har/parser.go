package har

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Header struct {
	Name  string
	Value string
}

type EntryDetails struct {
	Method      string
	URL         string
	URLPath     string
	HTTPVersion string
	Status      int
	StatusText  string
	Size        int64
	Time        float64
	Started     string
	DNS         float64
	Connect     float64
	SSL         float64
	Send        float64
	Wait        float64
	Receive     float64
	Blocked     float64

	ContentEncoding    string
	MimeType           string
	IsCompressed       bool
	ShouldBeCompressed bool
	ResourceType       string
	Domain             string

	CacheControl    string
	ETag            string
	LastModified    string
	Expires         string
	MaxAge          int
	NoCache         bool
	NoStore         bool
	MustRevalidate  bool
	HasCacheHeaders bool

	Priority        string
	InitialPriority string

	RequestHeaders  []Header
	ResponseHeaders []Header
}

func ParseFile(path string) (*Node, error) {
	data, err := ReadFileLimited(path)
	if err != nil {
		return nil, err
	}

	decompressed, err := Decompress(data)
	if err != nil {
		return nil, err
	}

	return ParseBytes(decompressed)
}

func ParseBytes(data []byte) (*Node, error) {
	raw, err := decodeOrdered(data)
	if err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return buildTreeFromOrdered(raw, "", 0, nil), nil
}

func buildTreeFromOrdered(v *orderedValue, key string, depth int, parent *Node) *Node {
	if depth > maxParseDepth {
		return &Node{Key: key, Parent: parent, Kind: Null, Depth: depth}
	}
	n := &Node{
		Key:    key,
		Parent: parent,
		Depth:  depth,
	}

	switch v.typ {
	case jsonObject:
		n.Kind = Object
		n.Collapsed = depth > 0
		n.IsKnown = IsKnownField(parentKey(parent), key)
		for _, k := range v.keys {
			child := buildTreeFromOrdered(v.fields[k], k, depth+1, n)
			n.Children = append(n.Children, child)
		}

	case jsonArray:
		n.Kind = Array
		n.Collapsed = depth > 0
		n.IsKnown = IsKnownField(parentKey(parent), key)
		for i, item := range v.items {
			child := buildTreeFromOrdered(item, fmt.Sprintf("[%d]", i), depth+1, n)
			n.Children = append(n.Children, child)
		}

	case jsonString:
		n.Kind = String
		n.Value = v.strVal
		n.IsKnown = IsKnownField(parentKey(parent), key)

	case jsonNumber:
		n.Kind = Number
		n.Value = mustParseFloat(v.numStr)
		n.NumStr = v.numStr
		n.IsKnown = IsKnownField(parentKey(parent), key)

	case jsonBool:
		n.Kind = Bool
		n.Value = v.boolVal
		n.IsKnown = IsKnownField(parentKey(parent), key)

	case jsonNull:
		n.Kind = Null
		n.Value = nil
		n.IsKnown = IsKnownField(parentKey(parent), key)
	}

	return n
}

func parentKey(n *Node) string {
	if n == nil {
		return ""
	}
	return n.Key
}

func ExportJSON(root *Node) ([]byte, error) {
	v := nodeToOrdered(root)
	minified, err := v.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, minified, "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func flattenAll(n *Node, result *[]*Node) {
	*result = append(*result, n)
	for _, child := range n.Children {
		flattenAll(child, result)
	}
}

func GetEntries(root *Node) []*Node {
	if root == nil || root.Kind != Object {
		return nil
	}
	logNode := findChild(root, "log")
	if logNode == nil {
		return nil
	}
	entriesNode := findChild(logNode, "entries")
	if entriesNode == nil || entriesNode.Kind != Array {
		return nil
	}
	return entriesNode.Children
}

func findChild(n *Node, key string) *Node {
	for _, child := range n.Children {
		if child.Key == key {
			return child
		}
	}
	return nil
}

func parseHeaders(node *Node) []Header {
	var headers []Header
	if node == nil || node.Kind != Array {
		return headers
	}
	for _, header := range node.Children {
		if header.Kind != Object {
			continue
		}
		name := findChild(header, "name")
		value := findChild(header, "value")
		if name != nil && name.Kind == String && value != nil && value.Kind == String {
			headers = append(headers, Header{
				Name:  name.Value.(string),
				Value: value.Value.(string),
			})
		}
	}
	return headers
}

func GetEntryDetails(entry *Node) *EntryDetails {
	d := &EntryDetails{}

	reqNode := findChild(entry, "request")
	if reqNode != nil {
		if m := findChild(reqNode, "method"); m != nil && m.Kind == String {
			d.Method = m.Value.(string)
		}
		if u := findChild(reqNode, "url"); u != nil && u.Kind == String {
			d.URL = u.Value.(string)
			d.URLPath = extractPath(d.URL)
			if parsed, err := url.Parse(d.URL); err == nil {
				d.Domain = parsed.Hostname()
			}
		}
		if h := findChild(reqNode, "httpVersion"); h != nil && h.Kind == String {
			d.HTTPVersion = h.Value.(string)
		}
		if h := findChild(reqNode, "headers"); h != nil && h.Kind == Array {
			d.RequestHeaders = parseHeaders(h)
		}
	}

	respNode := findChild(entry, "response")
	if respNode != nil {
		if s := findChild(respNode, "status"); s != nil && s.Kind == Number {
			d.Status = int(s.Value.(float64))
		}
		if st := findChild(respNode, "statusText"); st != nil && st.Kind == String {
			d.StatusText = st.Value.(string)
		}
		if c := findChild(respNode, "content"); c != nil {
			if sz := findChild(c, "size"); sz != nil && sz.Kind == Number {
				d.Size = int64(sz.Value.(float64))
			}
			if mt := findChild(c, "mimeType"); mt != nil && mt.Kind == String {
				d.MimeType = mt.Value.(string)
			}
		}
		if h := findChild(respNode, "headers"); h != nil && h.Kind == Array {
			d.ResponseHeaders = parseHeaders(h)
			for _, header := range h.Children {
				if header.Kind != Object {
					continue
				}
				name := findChild(header, "name")
				value := findChild(header, "value")
				if name != nil && name.Kind == String && value != nil && value.Kind == String {
					nameLower := strings.ToLower(name.Value.(string))
					val := value.Value.(string)
					switch nameLower {
					case "content-encoding":
						d.ContentEncoding = val
					case "cache-control":
						d.CacheControl = val
						d.HasCacheHeaders = true
						d.MaxAge = parseMaxAge(val)
						d.NoCache = strings.Contains(strings.ToLower(val), "no-cache")
						d.NoStore = strings.Contains(strings.ToLower(val), "no-store")
						d.MustRevalidate = strings.Contains(strings.ToLower(val), "must-revalidate")
					case "etag":
						d.ETag = val
						d.HasCacheHeaders = true
					case "last-modified":
						d.LastModified = val
						d.HasCacheHeaders = true
					case "expires":
						d.Expires = val
						d.HasCacheHeaders = true
					}
				}
			}
		}
	}

	if rt := findChild(entry, "_resourceType"); rt != nil && rt.Kind == String {
		d.ResourceType = rt.Value.(string)
	} else {
		d.ResourceType = inferResourceType(d.MimeType)
	}

	if p := findChild(entry, "_priority"); p != nil && p.Kind == String {
		d.Priority = p.Value.(string)
	}
	if p := findChild(entry, "_initialPriority"); p != nil && p.Kind == String {
		d.InitialPriority = p.Value.(string)
	}

	if t := findChild(entry, "time"); t != nil && t.Kind == Number {
		d.Time = t.Value.(float64)
	}
	if sd := findChild(entry, "startedDateTime"); sd != nil && sd.Kind == String {
		d.Started = sd.Value.(string)
	}

	timingsNode := findChild(entry, "timings")
	if timingsNode != nil {
		d.DNS = getTiming(timingsNode, "dns")
		d.Connect = getTiming(timingsNode, "connect")
		d.SSL = getTiming(timingsNode, "ssl")
		d.Send = getTiming(timingsNode, "send")
		d.Wait = getTiming(timingsNode, "wait")
		d.Receive = getTiming(timingsNode, "receive")
		d.Blocked = getTiming(timingsNode, "blocked")
	}

	d.IsCompressed = d.ContentEncoding != ""
	d.ShouldBeCompressed = shouldCompress(d.MimeType)

	return d
}

func extractPath(rawURL string) string {
	if parsed, err := url.Parse(rawURL); err == nil && parsed.Path != "" {
		return parsed.Path
	}
	if idx := strings.Index(rawURL, "://"); idx != -1 {
		path := rawURL[idx+3:]
		if slashIdx := strings.Index(path, "/"); slashIdx != -1 {
			return path[slashIdx:]
		}
	}
	return rawURL
}

var compressableTypes = []string{
	"text/html",
	"text/css",
	"text/plain",
	"text/xml",
	"text/javascript",
	"application/javascript",
	"application/json",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/atom+xml",
	"application/manifest+json",
	"application/vnd.ms-fontobject",
	"application/opensearchdescription+xml",
	"image/svg+xml",
}

func shouldCompress(mimeType string) bool {
	if mimeType == "" {
		return false
	}
	mimeLower := strings.ToLower(mimeType)
	if strings.HasPrefix(mimeLower, "text/") {
		return true
	}
	for _, ct := range compressableTypes {
		if strings.HasPrefix(mimeLower, ct) {
			return true
		}
	}
	return false
}

func inferResourceType(mimeType string) string {
	if mimeType == "" {
		return "other"
	}
	mt := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(mt, "text/html"):
		return "document"
	case strings.HasPrefix(mt, "text/css"):
		return "stylesheet"
	case strings.HasPrefix(mt, "application/javascript"),
		strings.HasPrefix(mt, "text/javascript"),
		strings.HasPrefix(mt, "application/x-javascript"):
		return "script"
	case strings.HasPrefix(mt, "image/"):
		return "image"
	case strings.HasPrefix(mt, "font/"),
		strings.HasPrefix(mt, "application/font-"),
		strings.HasPrefix(mt, "application/x-font-"):
		return "font"
	case strings.HasPrefix(mt, "application/json"),
		strings.HasPrefix(mt, "application/xml"),
		strings.HasPrefix(mt, "text/xml"),
		strings.HasPrefix(mt, "application/x-www-form-urlencoded"):
		return "xhr"
	case strings.HasPrefix(mt, "text/"):
		return "text"
	case strings.HasPrefix(mt, "application/"):
		return "script"
	case strings.HasPrefix(mt, "video/"):
		return "media"
	case strings.HasPrefix(mt, "audio/"):
		return "media"
	}
	return "other"
}

func getTiming(timings *Node, key string) float64 {
	n := findChild(timings, key)
	if n == nil {
		return 0
	}
	switch n.Kind {
	case Number:
		return n.Value.(float64)
	case Null:
		return 0
	}
	if s, ok := n.Value.(string); ok {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return f
	}
	return 0
}

func parseMaxAge(cacheControl string) int {
	lower := strings.ToLower(cacheControl)
	parts := strings.Split(lower, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			val := strings.TrimPrefix(part, "max-age=")
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		}
	}
	return -1
}

type CacheVerdict int

const (
	CacheNone CacheVerdict = iota
	CacheShort
	CacheGood
	CacheOverkill
)

func AnalyzeCache(e *EntryDetails) (CacheVerdict, string) {
	if e.NoStore {
		return CacheNone, "no-store set"
	}

	if e.Status == 200 || e.Status == 301 || e.Status == 302 || e.Status == 304 || e.Status == 307 {
		if e.MaxAge > 0 {
			switch {
			case e.MaxAge < 60:
				return CacheShort, "max-age too short (<1min)"
			case e.MaxAge <= 86400:
				return CacheGood, "good cache lifetime"
			case e.MaxAge <= 31536000:
				return CacheGood, "long cache lifetime"
			default:
				return CacheOverkill, "max-age >1 year"
			}
		}

		if e.ETag != "" || e.LastModified != "" {
			return CacheGood, "has validation headers"
		}

		if e.Expires != "" {
			return CacheGood, "has expires header"
		}
	}

	if !e.HasCacheHeaders {
		return CacheNone, "no cache headers"
	}

	return CacheShort, "minimal caching"
}

func mustParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
