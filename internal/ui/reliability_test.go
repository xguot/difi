package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/git"
	"github.com/xguot/difi/internal/vcs"
)

type fakeVCS struct {
	vcs.GitVCS
	files   []string
	stats   map[string][2]int
	err     error
	diffErr error
}

func (f *fakeVCS) GetCurrentBranch() string                  { return "test" }
func (f *fakeVCS) GetRepoName() string                       { return "test" }
func (f *fakeVCS) ListChangedFiles(string) ([]string, error) { return f.files, f.err }
func (f *fakeVCS) DiffStats(string) (int, int, error) {
	a, d := 0, 0
	for _, stats := range f.stats {
		a += stats[0]
		d += stats[1]
	}
	return a, d, f.err
}
func (f *fakeVCS) DiffStatsByFile(string) (map[string][2]int, error) { return f.stats, f.err }
func (f *fakeVCS) DiffCmd(target, path string) tea.Cmd {
	return func() tea.Msg {
		return vcs.DiffMsg{Content: "@@ -10 +10 @@\n-old\n+" + path + "\n", Err: f.diffErr}
	}
}

func testModel(f *fakeVCS) Model {
	return NewModel(config.Config{UI: config.UIConfig{Theme: "git", LineNumbers: "absolute"}}, "HEAD", "", f, true, "test")
}

func apply(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func drain(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			m = drain(m, child)
		}
		return m
	}
	m, cmd = apply(m, msg)
	return drain(m, cmd)
}

func key(value string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)} }

func TestCountedNavigation(t *testing.T) {
	f := &fakeVCS{}
	for i := 0; i < 25; i++ {
		f.files = append(f.files, fmt.Sprintf("file%02d", i))
	}
	m := testModel(f)
	m.inputBuffer = "10"
	m, _ = apply(m, key("j"))
	if m.fileList.Index() != 10 {
		t.Fatalf("10j moved to file %d", m.fileList.Index())
	}
	m.inputBuffer = "7"
	m, _ = apply(m, key("k"))
	if m.fileList.Index() != 3 {
		t.Fatalf("7k moved to file %d", m.fileList.Index())
	}
	m.focus, m.diffCursor = FocusDiff, 0
	m.diffLines = strings.Split(strings.Repeat(" context\n", 30), "\n")
	m.inputBuffer = "10"
	m, _ = apply(m, key("j"))
	if m.diffCursor != 10 {
		t.Fatalf("10j moved to diff line %d", m.diffCursor)
	}
}

func TestTabCompletionCyclesOriginalMatches(t *testing.T) {
	for _, partial := range []string{"", "set no", "colorscheme g", "theme g"} {
		m := testModel(&fakeVCS{})
		m.commandBuffer = partial
		m.tabComplete()
		first := m.commandBuffer
		count := len(m.tabCycleMatches)
		if count < 2 {
			t.Fatalf("need multiple completions for %q", partial)
		}
		m.tabComplete()
		if m.commandBuffer == first {
			t.Fatalf("completion did not cycle for %q", partial)
		}
		for i := 1; i < count; i++ {
			m.tabComplete()
		}
		if m.commandBuffer != first {
			t.Fatalf("completion did not wrap for %q", partial)
		}
		m.resetTabCycle()
		m.commandBuffer = "set hy"
		m.tabComplete()
		if m.commandBuffer != "set hybrid" {
			t.Fatalf("completion did not reset: %q", m.commandBuffer)
		}
	}
}

func TestRefreshAfterEditingUpdatesFilesAndTotals(t *testing.T) {
	f := &fakeVCS{files: []string{"a", "b"}, stats: map[string][2]int{"a": {1, 1}}}
	m := testModel(f)
	m = drain(m, m.Init())
	f.files = []string{"b", "c"}
	f.stats = map[string][2]int{"b": {2, 3}, "c": {4, 5}}
	m, cmd := apply(m, vcs.EditorFinishedMsg{})
	m = drain(m, cmd)
	if m.selectedPath != "b" || len(m.fileList.Items()) != 2 || m.statsAdded != 6 || m.statsDeleted != 8 {
		t.Fatalf("refresh kept stale state: selected=%q items=%d stats=+%d -%d", m.selectedPath, len(m.fileList.Items()), m.statsAdded, m.statsDeleted)
	}
	f.files, f.stats = nil, nil
	m.commandBuffer = "w"
	_, cmd = m.executeCommand()
	m = drain(m, cmd)
	if m.selectedPath != "" || len(m.diffLines) != 0 || m.statsAdded != 0 || m.statsDeleted != 0 || m.diffLoading {
		t.Fatalf("empty refresh kept stale state: %+v", m)
	}
	// Refresh must work from an initially empty tree too.
	f.files = []string{"new"}
	m.commandBuffer = "w"
	_, cmd = m.executeCommand()
	m = drain(m, cmd)
	if m.selectedPath != "new" {
		t.Fatalf("empty tree did not refresh: %q", m.selectedPath)
	}
}

func TestOutOfOrderDiffAndRefreshResults(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"a", "b"}})
	first := m.requestDiff()
	m.selectedPath = "b"
	m.fileList.Select(1)
	second := m.requestDiff()
	m = drain(m, second)
	m = drain(m, first)
	if !strings.Contains(m.diffContent, "+b") {
		t.Fatalf("older diff replaced current selection: %q", m.diffContent)
	}
	oldRefresh := m.refreshRequest
	m.refreshRequest++
	m, _ = apply(m, sessionMsg{Request: oldRefresh, Files: []string{"stale"}})
	if m.selectedPath != "b" {
		t.Fatal("older refresh replaced selection")
	}
	m.statsAdded = 7
	m, _ = apply(m, StatsMsg{Request: oldRefresh, Added: 99})
	if m.statsAdded != 7 {
		t.Fatal("older stats replaced current totals")
	}
}

func TestErrorsAreVisible(t *testing.T) {
	m := testModel(&fakeVCS{err: errors.New("invalid target")})
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if view := stripAnsi(m.View()); !strings.Contains(view, "Unable to load changes") || strings.Contains(view, "No changes found") {
		t.Fatalf("misleading error view:\n%s", view)
	}
	m = testModel(&fakeVCS{files: []string{"a"}, diffErr: errors.New("cannot read file")})
	m = drain(m, m.Init())
	m, _ = apply(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if view := stripAnsi(m.View()); !strings.Contains(view, "Unable to load diff") {
		t.Fatalf("missing diff error:\n%s", view)
	}
}

func TestFileLineNumbersAndPatchLikeContent(t *testing.T) {
	content := "diff --git a/file b/file\n--- a/file\n+++ b/file\n@@ -10,2 +20,2 @@\n context\n--- old content\n+++ new content\n@@ -40 +50 @@\n-old\n+new\n"
	result := prepareDiff(content, "file", "git", "nord")
	if result.Added != 2 || result.Deleted != 2 {
		t.Fatalf("patch-like content miscounted: %+v", result)
	}
	for i, line := range result.FileLines {
		if want := git.CalculateFileLine(result.Lines, i); line != want {
			t.Errorf("line %d maps to %d, want %d", i, line, want)
		}
	}
	if !reflect.DeepEqual(result.FileLines, []int{20, 20, 21, 21, 50, 50, 50}) {
		t.Fatalf("wrong file numbers: %v", result.FileLines)
	}
}

func TestPipedStatsUseSamePathsAsTree(t *testing.T) {
	text := "diff --git i/path with spaces w/path with spaces\n@@ -1 +1 @@\n---old\n+++new\n"
	m := NewModel(config.Config{}, "HEAD", text, &fakeVCS{}, true, "test")
	stats := m.computePipedStatsCmd()().(StatsMsg)
	if stats.ByFile[m.selectedPath] != [2]int{1, 1} {
		t.Fatalf("tree path %q differs from stats %v", m.selectedPath, stats.ByFile)
	}
}

func TestThemeChangeReloadsHighlightingAndRejectsOldWork(t *testing.T) {
	m := NewModel(config.Config{UI: config.UIConfig{Theme: "nord"}}, "HEAD", "", &fakeVCS{files: []string{"file.go"}}, true, "test")
	old := m.loadDiffCmd()
	m.commandBuffer = "colorscheme dracula"
	_, cmd := m.executeCommand()
	if cmd == nil || m.treeDelegate.Config.UI.Theme != "dracula" {
		t.Fatal("theme change did not reload the diff")
	}
	m = drain(m, cmd)
	highlighted := append([]string(nil), m.diffHighlighted...)
	m = drain(m, old)
	if !reflect.DeepEqual(m.diffHighlighted, highlighted) {
		t.Fatal("old theme result replaced new highlighting")
	}
}

func TestMetadataAndBinaryNotices(t *testing.T) {
	for _, tt := range []struct{ text, notice string }{
		{"diff --git a/a b/a\nold mode 100644\nnew mode 100755\n", "File metadata changed"},
		{"diff --git a/a b/a\nBinary files a/a and b/a differ\n", "Binary file changed"},
	} {
		if result := prepareDiff(tt.text, "a", "git", "nord"); result.Notice != tt.notice {
			t.Fatalf("notice = %q, want %q", result.Notice, tt.notice)
		}
	}
}
