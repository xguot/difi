package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/review"
)

// Settle diff loads while leaving cursor blink timers to the real event loop.
func navigationKey(m Model, msg tea.KeyMsg) Model {
	m, _ = apply(m, msg)
	if m.diffLoading {
		m = drain(m, m.loadDiffCmd())
	}
	return m
}

func TestFileFilterInputCancelAndClear(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"a.go", "src/Écho.go", "docs/guide.md"}})
	m = drain(m, m.Init())
	m, _ = apply(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	original := m.selectedPath
	m = navigationKey(m, key("/"))
	m = navigationKey(m, key("é"))
	if m.fileFilter != "é" || m.selectedPath != "src/Écho.go" || len(m.fileList.Items()) != 1 {
		t.Fatalf("Unicode filtering: query=%q path=%q", m.fileFilter, m.selectedPath)
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.fileFilter != "" {
		t.Fatalf("backspace split Unicode input: %q", m.fileFilter)
	}
	m = navigationKey(m, key("no-such-file"))
	if len(m.fileList.Items()) != 0 || !strings.Contains(stripAnsi(m.View()), "No files matching") {
		t.Fatal("missing no-match state")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.fileFilter != "" || m.selectedPath != original || m.promptMode != promptNone {
		t.Fatal("cancel did not restore the original filter and selection")
	}
	m = navigationKey(m, key("/"))
	m = navigationKey(m, key(".go"))
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.promptMode != promptNone || len(m.fileList.Items()) != 2 {
		t.Fatal("committed filter was lost")
	}
	if !strings.Contains(stripAnsi(m.View()), "Files /.go") {
		t.Fatalf("filter footer is not visible:\n%s", stripAnsi(m.View()))
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.fileFilter != "" || len(m.fileList.Items()) != 3 {
		t.Fatal("Esc did not clear the committed filter")
	}
}

func TestFilterSurvivesRefreshAndSearchesCollapsedDirectories(t *testing.T) {
	f := &fakeVCS{files: []string{"a", "src/first", "src/second"}}
	m := NewModel(config.Config{}, "HEAD", "", f, false, "test")
	m.rebuildFileList("a")
	m.treeState.ToggleExpand("src")
	m = navigationKey(m, key("/"))
	m = navigationKey(m, key("src/"))
	if len(m.fileList.Items()) != 2 || m.selectedPath != "src/first" {
		t.Fatal("filter omitted collapsed files")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	f.files = []string{"a", "src/new"}
	m = drain(m, m.refreshSession())
	if m.fileFilter != "src/" || len(m.fileList.Items()) != 1 || m.selectedPath != "src/new" {
		t.Fatal("refresh lost the filter or kept stale files")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.treeState.Root.Children["src"].Expanded || m.selectedPath != "src/new" {
		t.Fatal("clearing filter hid the selected file")
	}
}

func TestLiteralSearchWrapCountsAndCancel(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"a"}})
	content := "@@ -1,4 +1,4 @@\n fmt.Println(\n-old\n+FMT.Println(\n écho\n"
	result := prepareDiff(content, "a", "git", "nord")
	result.Request = m.diffRequest
	m, _ = apply(m, result)
	m.focus = FocusDiff
	m = navigationKey(m, key("/"))
	m = navigationKey(m, key("fmt.Println("))
	if !reflect.DeepEqual(m.searchMatches, []int{1, 3}) {
		t.Fatalf("literal search matches = %v", m.searchMatches)
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.diffCursor != 3 {
		t.Fatalf("search did not find next match: %d", m.diffCursor)
	}
	m = navigationKey(m, key("n"))
	if m.diffCursor != 1 {
		t.Fatal("forward search did not wrap")
	}
	m = navigationKey(m, key("N"))
	if m.diffCursor != 3 {
		t.Fatal("backward search did not wrap")
	}
	m.inputBuffer = "1000001"
	m = navigationKey(m, key("n"))
	if m.diffCursor != 1 {
		t.Fatal("large repeat count was not applied modulo the matches")
	}
	m = navigationKey(m, key("/"))
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	m = navigationKey(m, key("absent"))
	if !strings.Contains(m.navigationStatus(), "no matches") {
		t.Fatal("missing no-match feedback")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchQuery != "fmt.Println(" || m.diffCursor != 1 {
		t.Fatal("cancel did not restore the search and cursor")
	}
	m.commandBuffer = "noh"
	m.executeCommand()
	if m.searchQuery != "" || len(m.searchMatches) != 0 {
		t.Fatal(":noh did not clear search")
	}
}

func TestHunkNavigationAndLegacyFocusKeys(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"a"}})
	result := prepareDiff("@@ -1 +1 @@\n context\n-old\n+new\n@@ -20 +20 @@\n-second\n+second\n@@ -40 +40 @@\n-third\n+third\n", "a", "git", "nord")
	result.Request = m.diffRequest
	m, _ = apply(m, result)
	if !reflect.DeepEqual(m.diffHunks, []int{2, 5, 8}) {
		t.Fatalf("hunks = %v", m.diffHunks)
	}
	m.diffCursor = 2
	m = navigationKey(m, key("]"))
	m = navigationKey(m, key("c"))
	if m.diffCursor != 5 || m.focus != FocusDiff {
		t.Fatal("]c did not jump to next change")
	}
	m = navigationKey(m, key("["))
	m = navigationKey(m, key("c"))
	if m.diffCursor != 2 {
		t.Fatal("[c did not jump to previous change")
	}
	m = navigationKey(m, key("2"))
	m = navigationKey(m, key("]"))
	m = navigationKey(m, key("c"))
	if m.diffCursor != 8 {
		t.Fatalf("2]c moved to %d", m.diffCursor)
	}
	m = navigationKey(m, key("}"))
	if m.diffCursor != 8 {
		t.Fatal("hunk navigation wrapped past the final change")
	}
	m = navigationKey(m, key("["))
	if m.focus != FocusTree {
		t.Fatal("legacy [ focus key changed")
	}
	m = navigationKey(m, key("j"))
	if m.pendingHunk != "" {
		t.Fatal("unrelated key did not cancel hunk prefix")
	}
}

func TestScopeLabelsAndEmptyStdin(t *testing.T) {
	for _, tt := range []struct {
		options review.Options
		want    string
	}{
		{review.Options{Scope: review.Staged}, "HEAD ➜ index (staged)"},
		{review.Options{Scope: review.Unstaged}, "index ➜ working tree (unstaged)"},
		{review.Options{NoUntracked: true}, "tracked only"},
	} {
		m := NewModel(config.Config{}, "HEAD", "", &fakeVCS{}, true, "test", Options{Review: tt.options})
		if !strings.Contains(m.comparisonLabel(), tt.want) {
			t.Fatalf("scope label = %q", m.comparisonLabel())
		}
	}
	m := NewModel(config.Config{}, "HEAD", "", &fakeVCS{files: []string{"must-not-appear"}}, true, "test", Options{FromStdin: true})
	m = drain(m, m.Init())
	if len(m.fileList.Items()) != 0 || m.emptyMessage() != "No changes in piped patch" {
		t.Fatal("empty patch fell back to repository data")
	}
	m = drain(m, m.refreshSession())
	if len(m.fileList.Items()) != 0 {
		t.Fatal("refresh escaped piped mode")
	}
}

func TestNavigationDuringAsyncLoadAndSmallViewport(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"one", "two"}})
	m = navigationKey(m, key("/"))
	m = navigationKey(m, key("two"))
	m = drain(m, m.loadDiffCmd())
	if m.promptMode != promptFiles || !strings.Contains(m.diffContent, "+two") {
		t.Fatal("async result was lost while prompt was active")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	for _, width := range []int{120, 60, 40, 20} {
		m, _ = apply(m, tea.WindowSizeMsg{Width: width, Height: 12})
		view := m.View()
		if lipgloss.Width(view) > width || lipgloss.Height(view) > 12 {
			t.Errorf("view at width %d is %dx%d", width, lipgloss.Width(view), lipgloss.Height(view))
		}
	}
	// Results for a previously selected file must remain ignored during input.
	m, _ = apply(m, diffResultMsg{Request: m.diffRequest - 1, Content: "stale"})
	if strings.Contains(m.diffContent, "stale") {
		t.Fatal("stale result replaced the filtered selection")
	}
}

func TestBatchedTypingAndPastedText(t *testing.T) {
	m := testModel(&fakeVCS{files: []string{"work.go", "other.go"}})
	m = navigationKey(m, key("/work"))
	if m.promptMode != promptFiles || m.fileFilter != "work" || m.selectedPath != "work.go" {
		t.Fatal("batched /query input was ignored")
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = navigationKey(m, key(":colorscheme dracula"))
	if !m.commandMode || m.commandBuffer != "colorscheme dracula" {
		t.Fatalf("batched command was lost: %q", m.commandBuffer)
	}
	m = navigationKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	pasted := key("q")
	pasted.Paste = true
	_, cmd := apply(m, pasted)
	if cmd != nil {
		t.Fatal("normal-mode paste executed a command")
	}
}
