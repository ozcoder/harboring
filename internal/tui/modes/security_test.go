package modes

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func makeSecurityEntry(url string, status int, domain string) *har.EntryDetails {
	return &har.EntryDetails{
		Method: "GET",
		URL:    url,
		Status: status,
		Domain: domain,
	}
}

func makeRawEntry(url string, headers []string) *har.Node {
	entry := &har.Node{Kind: har.Object}
	request := &har.Node{Kind: har.Object, Key: "request"}
	urlNode := &har.Node{Kind: har.String, Key: "url", Value: url}
	request.Children = []*har.Node{urlNode}

	response := &har.Node{Kind: har.Object, Key: "response"}
	headersNode := &har.Node{Kind: har.Array, Key: "headers"}
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headerObj := &har.Node{Kind: har.Object}
		nameNode := &har.Node{Kind: har.String, Key: "name", Value: parts[0]}
		valueNode := &har.Node{Kind: har.String, Key: "value", Value: parts[1]}
		headerObj.Children = []*har.Node{nameNode, valueNode}
		headersNode.Children = append(headersNode.Children, headerObj)
	}
	response.Children = []*har.Node{headersNode}
	entry.Children = []*har.Node{request, response}
	return entry
}

func TestSecurityModelComputesResults(t *testing.T) {
	th := theme.New()

	// Entry with no security headers
	e1 := makeSecurityEntry("http://example.com/page", 200, "example.com")
	r1 := makeRawEntry("http://example.com/page", nil)

	// Entry with all security headers
	e2 := makeSecurityEntry("http://secure.com/page", 200, "secure.com")
	r2 := makeRawEntry("http://secure.com/page", []string{
		"strict-transport-security: max-age=31536000",
		"x-content-type-options: nosniff",
		"content-security-policy: default-src 'self'",
		"x-frame-options: DENY",
		"referrer-policy: no-referrer",
		"permissions-policy: geolocation=()",
		"cross-origin-opener-policy: same-origin",
		"cross-origin-resource-policy: same-origin",
	})

	entries := []*har.EntryDetails{e1, e2}
	rawEntries := []*har.Node{r1, r2}

	m := NewSecurityModel(entries, rawEntries, th)

	if len(m.results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(m.results))
	}

	// Find the insecure domain result
	var insecure, secure *SecurityResult
	for i := range m.results {
		if m.results[i].Domain == "example.com" {
			insecure = &m.results[i]
		}
		if m.results[i].Domain == "secure.com" {
			secure = &m.results[i]
		}
	}

	if insecure == nil {
		t.Fatal("expected example.com result")
	}
	if insecure.MissingRequired != 4 {
		t.Errorf("expected 4 missing required headers for example.com, got %d", insecure.MissingRequired)
	}
	if insecure.MissingOptional != 4 {
		t.Errorf("expected 4 missing optional headers for example.com, got %d", insecure.MissingOptional)
	}

	if secure == nil {
		t.Fatal("expected secure.com result")
	}
	if secure.MissingRequired != 0 {
		t.Errorf("expected 0 missing required headers for secure.com, got %d", secure.MissingRequired)
	}
}

func TestSecurityModelView(t *testing.T) {
	th := theme.New()

	e1 := makeSecurityEntry("http://example.com/page", 200, "example.com")
	r1 := makeRawEntry("http://example.com/page", nil)

	m := NewSecurityModel([]*har.EntryDetails{e1}, []*har.Node{r1}, th)
	m.SetViewportSize(80, 20)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	if !strings.Contains(view, "Security Headers") {
		t.Error("expected Security Headers title in view")
	}
	if !strings.Contains(view, "example.com") {
		t.Error("expected domain in view")
	}
}

func TestSecurityModelNavigation(t *testing.T) {
	th := theme.New()

	entries := []*har.EntryDetails{
		makeSecurityEntry("http://a.com/page", 200, "a.com"),
		makeSecurityEntry("http://b.com/page", 200, "b.com"),
		makeSecurityEntry("http://c.com/page", 200, "c.com"),
	}
	rawEntries := []*har.Node{
		makeRawEntry("http://a.com/page", nil),
		makeRawEntry("http://b.com/page", nil),
		makeRawEntry("http://c.com/page", nil),
	}

	m := NewSecurityModel(entries, rawEntries, th)

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	// Move down
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.cursor != 1 {
		t.Errorf("expected cursor at 1 after down, got %d", m2.cursor)
	}

	// Move down again
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m3.cursor != 2 {
		t.Errorf("expected cursor at 2 after second down, got %d", m3.cursor)
	}

	// Move up
	m4, _ := m3.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m4.cursor != 1 {
		t.Errorf("expected cursor at 1 after up, got %d", m4.cursor)
	}
}
