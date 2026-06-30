package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/diffsplit"
)

func initTestStyles(t *testing.T) {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	cfg := config.Config{UI: config.UIConfig{Theme: "nord"}}
	InitStyles(cfg)
}

func TestComputeSplitLayout(t *testing.T) {
	layout := computeSplitLayout(81, true)
	if layout.leftColW+1+layout.rightColW != 81 {
		t.Fatalf("columns don't sum to total: %d + 1 + %d != 81", layout.leftColW, layout.rightColW)
	}
	if layout.leftCodeW != layout.leftColW-splitLineNumWidth-splitGutterWidth {
		t.Errorf("leftCodeW = %d, want %d", layout.leftCodeW, layout.leftColW-splitLineNumWidth-splitGutterWidth)
	}

	noNums := computeSplitLayout(80, false)
	if noNums.leftCodeW != noNums.leftColW-splitGutterWidth {
		t.Errorf("without line nums, leftCodeW should equal leftColW minus gutter")
	}
}

func TestPadVisible(t *testing.T) {
	if got := padVisible("abc", 6); got != "abc   " {
		t.Errorf("padVisible short = %q", got)
	}
	if got := padVisible("hello world", 5); lipgloss.Width(got) != 5 {
		t.Errorf("padVisible truncate width = %d, want 5", lipgloss.Width(got))
	}
}

func TestSplitGutterStr(t *testing.T) {
	if got := splitGutterStr(diffsplit.KindAdd, false); got != "+ │ " {
		t.Errorf("add gutter = %q", got)
	}
	if got := splitGutterStr(diffsplit.KindDel, true); got != "- ┃ " {
		t.Errorf("del cursor gutter = %q", got)
	}
	if got := splitGutterStr(diffsplit.KindContext, false); got != "  │ " {
		t.Errorf("ctx gutter = %q", got)
	}
}

func TestExpandTabs(t *testing.T) {
	if got := expandTabs("a\tb"); got != "a    b" {
		t.Errorf("expandTabs = %q", got)
	}
}

func TestRenderSplitCode_noBackgroundByDefault(t *testing.T) {
	initTestStyles(t)
	m := Model{}
	side := diffsplit.Side{
		Kind:    diffsplit.KindAdd,
		Content: `fmt.Println("new")`,
	}
	got := m.renderSplitCode(side, 40, false)
	if strings.Contains(got, "\x1b[48") {
		t.Errorf("default add line should not have background highlight: %q", got)
	}
}
