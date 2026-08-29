package har

var knownTopLevel = map[string]bool{
	"log": true,
}

var knownLog = map[string]bool{
	"version": true,
	"creator": true,
	"browser": true,
	"pages":   true,
	"entries": true,
	"comment": true,
}

var knownCreator = map[string]bool{
	"name":    true,
	"version": true,
	"comment": true,
}

var knownBrowser = map[string]bool{
	"name":    true,
	"version": true,
	"comment": true,
}

var knownPage = map[string]bool{
	"startedDateTime": true,
	"id":              true,
	"title":           true,
	"pageTimings":     true,
	"comment":         true,
}

var knownEntry = map[string]bool{
	"pageref":            true,
	"startedDateTime":    true,
	"time":               true,
	"request":            true,
	"response":           true,
	"cache":              true,
	"timings":            true,
	"serverIPAddress":    true,
	"connection":         true,
	"comment":            true,
	"serverTiming":       true,
	"_resourceType":      true,
	"_initiator":         true,
	"_webSocketMessages": true,
	"_headers":           true,
	"_redirects":         true,
}

var knownRequest = map[string]bool{
	"method":      true,
	"url":         true,
	"httpVersion": true,
	"cookies":     true,
	"headers":     true,
	"queryString": true,
	"postData":    true,
	"headersSize": true,
	"bodySize":    true,
	"comment":     true,
}

var knownResponse = map[string]bool{
	"status":      true,
	"statusText":  true,
	"httpVersion": true,
	"cookies":     true,
	"headers":     true,
	"content":     true,
	"redirectURL": true,
	"headersSize": true,
	"bodySize":    true,
	"comment":     true,
}

var knownContent = map[string]bool{
	"size":        true,
	"mimeType":    true,
	"text":        true,
	"encoding":    true,
	"compression": true,
	"comment":     true,
}

var knownTimings = map[string]bool{
	"blocked": true,
	"dns":     true,
	"connect": true,
	"send":    true,
	"wait":    true,
	"receive": true,
	"ssl":     true,
	"comment": true,
}

var knownCookie = map[string]bool{
	"name":     true,
	"value":    true,
	"path":     true,
	"domain":   true,
	"expires":  true,
	"httpOnly": true,
	"secure":   true,
	"sameSite": true,
	"comment":  true,
}

var knownCache = map[string]bool{
	"beforeRequest": true,
	"afterRequest":  true,
	"comment":       true,
}

var knownQueryString = map[string]bool{
	"name":    true,
	"value":   true,
	"comment": true,
}

var knownFieldsByParent = map[string]map[string]bool{
	"log":         knownLog,
	"creator":     knownCreator,
	"browser":     knownBrowser,
	"pages":       knownPage,
	"entries":     knownEntry,
	"request":     knownRequest,
	"response":    knownResponse,
	"content":     knownContent,
	"timings":     knownTimings,
	"cookie":      knownCookie,
	"cookies":     knownCookie,
	"cache":       knownCache,
	"queryString": knownQueryString,
}

func IsKnownField(parentKey string, fieldKey string) bool {
	if parentKey == "" {
		return knownTopLevel[fieldKey]
	}
	if fields, ok := knownFieldsByParent[parentKey]; ok {
		return fields[fieldKey]
	}
	return false
}
