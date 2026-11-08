package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/config"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func makeApp() AppModel {
	return makeAppWithTheme(theme.New())
}

func makeAppWithTheme(th *theme.Engine) AppModel {
	root := &har.Node{Kind: har.Object, Key: "log"}
	entries := &har.Node{Kind: har.Array, Key: "entries"}
	n1 := &har.Node{Kind: har.Object, Key: "0"}
	url := &har.Node{Kind: har.String, Key: "url", Value: "http://example.com"}
	n1.Children = []*har.Node{url}
	entries.Children = []*har.Node{n1}
	root.Children = []*har.Node{entries}

	m := NewApp(root, "test.har", th)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	return result.(AppModel)
}

func TestScrubConfirmFlow(t *testing.T) {
	m := makeApp()

	res, _ := m.handleKey(tea.KeyPressMsg{Text: "s"})
	m2 := res.(AppModel)
	if !m2.popover.IsActive() {
		t.Fatal("expected confirm popover to open after pressing s")
	}

	res, cmd := m2.Update(tea.KeyPressMsg{Text: "y"})
	m3 := res.(AppModel)
	if cmd == nil {
		t.Fatal("expected a command from the confirm model")
	}
	msg := cmd()
	res, _ = m3.Update(msg)
	m4 := res.(AppModel)
	if m4.popover.IsActive() {
		t.Fatal("expected popover closed after confirming scrub")
	}
	if m4.statusMsg == "" {
		t.Fatal("expected a status message after confirming scrub")
	}
}

func TestScrubCancelFlow(t *testing.T) {
	m := makeApp()

	res, _ := m.handleKey(tea.KeyPressMsg{Text: "s"})
	m2 := res.(AppModel)

	res, cmd := m2.Update(tea.KeyPressMsg{Text: "n"})
	m3 := res.(AppModel)
	if cmd == nil {
		t.Fatal("expected a command from the confirm model")
	}
	msg := cmd()
	res, _ = m3.Update(msg)
	m4 := res.(AppModel)
	if m4.popover.IsActive() {
		t.Fatal("expected popover closed after cancelling scrub")
	}
	if m4.statusMsg != "Scrub cancelled" {
		t.Fatalf("expected cancel status message, got %q", m4.statusMsg)
	}
}

func TestHeaderAndFooter(t *testing.T) {
	m := makeApp()

	view := m.View().Content
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Fatalf("expected 24 lines, got %d", len(lines))
	}

	first := lines[0]
	if !strings.Contains(first, "Tree") {
		t.Fatalf("expected view name in header (first line), got %q", first)
	}

	last := lines[len(lines)-1]
	if !strings.Contains(stripANSI(last), "q:Quit") {
		t.Fatalf("expected controls in footer (last line), got %q", last)
	}
}

func TestFooterAtBottomSmallContent(t *testing.T) {
	m := makeApp()
	m.tree.SetCursor(0)

	view := m.View().Content
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Fatalf("expected 24 lines even with short content, got %d", len(lines))
	}

	first := lines[0]
	if !strings.Contains(first, "Tree") {
		t.Fatalf("expected view name in header (first line), got %q", first)
	}

	last := lines[len(lines)-1]
	if !strings.Contains(stripANSI(last), "q:Quit") {
		t.Fatalf("expected controls in footer (last line), got %q", last)
	}
}

func TestFooterGlobalFullNamesWhenWide(t *testing.T) {
	m := makeApp()
	m.width = 200

	_, global := m.renderControlsFooter()
	plain := stripANSI(global)
	for _, name := range []string{"Tree", "Requests", "Timings", "Summary", "Cache", "Security", "Timeline"} {
		if !strings.Contains(plain, name) {
			t.Fatalf("expected full view name %q in footer, got %q", name, plain)
		}
	}
}

func TestFooterGlobalTruncatesNamesToFit(t *testing.T) {
	m := makeApp() // width 100

	_, global := m.renderControlsFooter()
	plain := stripANSI(global)
	if !strings.Contains(plain, "2:Reques") {
		t.Fatalf("expected Requests truncated to fit, got %q", plain)
	}
	if strings.Contains(plain, "2:Requests") {
		t.Fatalf("expected Requests to be truncated at width 100, got %q", plain)
	}
	if !strings.Contains(plain, "q:Quit") {
		t.Fatalf("expected global actions to survive truncation, got %q", plain)
	}
}

func TestFooterSingleSpacesAndBoldKeys(t *testing.T) {
	m := makeApp()

	specific, global := m.renderControlsFooter()
	for name, line := range map[string]string{"specific": specific, "global": global} {
		plain := stripANSI(line)
		if strings.Contains(plain, "  ") {
			t.Fatalf("expected single spaces only in %s footer, got %q", name, plain)
		}
		if !strings.Contains(line, "\x1b[1") {
			t.Fatalf("expected bold keys in %s footer, got %q", name, line)
		}
	}
}

func TestSecurityModeSwitch(t *testing.T) {
	m := makeApp()

	// Press "6" to switch to security mode
	result, _ := m.handleKey(tea.KeyPressMsg{Text: "6"})
	m2 := result.(AppModel)

	if m2.mode != ModeSecurity {
		t.Fatalf("expected ModeSecurity, got %d", m2.mode)
	}

	view := m2.View().Content
	if !strings.Contains(view, "Security") {
		t.Errorf("expected security content in view")
	}
}

func TestCLIThemePrecedenceAndPersistence(t *testing.T) {
	// Hermetic config dir: LoadStartup must not see a real theme.toml, and
	// the t-press below persists settings, which must not touch the user's
	// actual config.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// A --theme flag pins the startup theme; persisted settings must not
	// override it via applySettings.
	th := theme.New()
	th.LoadStartup("turbo")
	settings := config.DefaultSettings()
	settings.Theme = "light" // persisted "last picked" from a previous run
	m := makeApp()
	m.theme = th
	m.settings = settings
	m.applySettings()
	if m.theme.Name() != "turbo" {
		t.Fatalf("expected --theme turbo to win over persisted light, got %q", m.theme.Name())
	}

	// Without a CLI flag, the persisted theme applies.
	th2 := theme.New()
	th2.LoadStartup("")
	m2 := makeApp()
	m2.theme = th2
	m2.settings = config.DefaultSettings()
	m2.settings.Theme = "spectrum"
	m2.applySettings()
	if m2.theme.Name() != "spectrum" {
		t.Fatalf("expected persisted spectrum to apply with no CLI flag, got %q", m2.theme.Name())
	}

	// Pressing t switches AND persists the new theme to settings.
	th3 := theme.New()
	th3.LoadStartup("")
	m3 := makeApp()
	m3.theme = th3
	m3.settings = config.DefaultSettings()
	before := m3.theme.Name()
	res, _ := m3.handleKey(tea.KeyPressMsg{Text: "t"})
	m4 := res.(AppModel)
	if m4.theme.Name() == before {
		t.Fatalf("expected theme to change after t, still %q", m4.theme.Name())
	}
	if m4.settings.Theme != m4.theme.Name() {
		t.Fatalf("expected settings.Theme persisted to %q after toggle, got %q", m4.theme.Name(), m4.settings.Theme)
	}
}

func TestRequestsTypeFilterModeToggle(t *testing.T) {
	m := makeApp()
	m.mode = ModeRequests

	// Press "f" to enter filter mode.
	res, _ := m.Update(tea.KeyPressMsg{Text: "f"})
	m2 := res.(AppModel)
	if !m2.requests.FilterMode() {
		t.Fatal("expected filter mode active after pressing f")
	}

	// Toggle a type with space.
	res, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m3 := res.(AppModel)
	if !m3.requests.FilterMode() {
		t.Fatal("expected filter mode still active after space")
	}

	// Exit filter mode with esc.
	res, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m4 := res.(AppModel)
	if m4.requests.FilterMode() {
		t.Fatal("expected filter mode inactive after esc")
	}
}
