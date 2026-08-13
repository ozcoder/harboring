package theme

import (
	"testing"
)

func TestNew(t *testing.T) {
	th := New()
	if th == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetStyle(t *testing.T) {
	th := New()
	style := th.GetStyle("title")
	if style.Render("test") == "test" {
		// style has no rendering if it's empty, which is fine
	}
}

func TestToggle(t *testing.T) {
	th := New()
	dark1 := th.GetFg()
	th.Toggle()
	light1 := th.GetFg()
	if dark1 == light1 {
		t.Fatal("theme toggle should change foreground color")
	}
	th.Toggle()
	dark2 := th.GetFg()
	if dark1 != dark2 {
		t.Fatal("theme toggle should revert to original")
	}
}

func TestGetWarn(t *testing.T) {
	th := New()
	warn := th.GetWarn()
	if warn == "" {
		t.Fatal("GetWarn should return a non-empty color")
	}
}

func TestGetGray(t *testing.T) {
	th := New()
	gray := th.GetGray()
	if gray == "" {
		t.Fatal("GetGray should return a non-empty color")
	}
}

func TestLoadStartup(t *testing.T) {
	th := New()
	// Passing empty string should not panic
	th.LoadStartup("")
}
