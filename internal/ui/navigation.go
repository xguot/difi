package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/xguot/difi/internal/review"
	"github.com/xguot/difi/internal/tree"
)

type promptKind int

const (
	promptNone promptKind = iota
	promptFiles
	promptSearch
)

func (m *Model) beginPrompt() tea.Cmd {
	m.promptMode = promptSearch
	value := m.searchQuery
	if m.focus == FocusTree {
		m.promptMode, value = promptFiles, m.fileFilter
	}
	m.promptOriginal = value
	m.promptPath, m.promptCursor, m.promptOffset = m.selectedPath, m.diffCursor, m.diffViewport.YOffset
	m.promptInput = textinput.New()
	m.promptInput.Prompt = ""
	m.promptInput.CharLimit = 1024
	m.promptInput.Width = max(1, m.width-12)
	m.promptInput.TextStyle = S.CmdLineStyle
	m.promptInput.Cursor.Style = S.CmdLineCursor
	m.promptInput.SetValue(value)
	m.promptInput.CursorEnd()
	m.visualMode, m.pendingZZ, m.pendingZ = false, false, false
	m.pendingHunk, m.inputBuffer = "", ""
	m.updateSizes()
	return m.promptInput.Focus()
}

func (m *Model) updatePrompt(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		var cmd tea.Cmd
		if m.promptMode == promptFiles {
			m.fileFilter = m.promptOriginal
			old := m.selectedPath
			m.rebuildFileList(m.promptPath)
			if old != m.selectedPath {
				cmd = m.requestDiff()
			}
		} else {
			m.searchQuery = m.promptOriginal
			m.rebuildSearch()
		}
		m.diffCursor = m.promptCursor
		if !m.diffLoading {
			m.diffCursor = m.snapCursor(m.diffCursor, 1)
		}
		m.setYOffset(m.promptOffset)
		m.promptMode = promptNone
		m.promptInput.Blur()
		m.updateSizes()
		return cmd
	case "enter":
		if m.promptMode == promptSearch {
			m.searchQuery = m.promptInput.Value()
			m.rebuildSearch()
			m.jumpSearch(1)
		}
		m.promptMode = promptNone
		m.promptInput.Blur()
		m.updateSizes()
		return nil
	}
	previous := m.promptInput.Value()
	var inputCmd tea.Cmd
	m.promptInput, inputCmd = m.promptInput.Update(msg)
	if previous == m.promptInput.Value() {
		return inputCmd
	}
	if m.promptMode == promptFiles {
		m.fileFilter = m.promptInput.Value()
		old := m.selectedPath
		m.rebuildFileList(m.promptPath)
		if old != m.selectedPath {
			return tea.Batch(inputCmd, m.requestDiff())
		}
	} else {
		m.searchQuery = m.promptInput.Value()
		m.rebuildSearch()
	}
	return inputCmd
}

func (m Model) fileItems() []list.Item {
	if m.fileFilter == "" {
		return m.treeState.Items(m.flatMode)
	}
	query := strings.ToLower(m.fileFilter)
	var items []list.Item
	for _, item := range m.treeState.Items(true) {
		if strings.Contains(strings.ToLower(item.FilterValue()), query) {
			items = append(items, item)
		}
	}
	return items
}

func (m *Model) rebuildFileList(preferred string) {
	previous := m.selectedPath
	if m.fileFilter == "" && !m.flatMode && preferred != "" {
		node := m.treeState.Root
		parts := strings.Split(preferred, "/")
		for _, name := range parts[:len(parts)-1] {
			if node = node.Children[name]; node == nil {
				break
			}
			node.Expanded = true
		}
	}
	items := m.fileItems()
	m.fileList.SetItems(items)
	m.selectedPath = ""
	first := -1
	for i, item := range items {
		if ti, ok := item.(tree.TreeItem); ok && !ti.IsDir {
			if first < 0 {
				first = i
			}
			if ti.FullPath == preferred {
				first = i
				break
			}
		}
	}
	if first >= 0 {
		m.fileList.Select(first)
		m.selectedPath = items[first].(tree.TreeItem).FullPath
	}
	if previous != m.selectedPath {
		m.diffCursor, m.visualStart = 0, 0
		m.visualMode = false
		m.diffViewport.GotoTop()
	}
}

func (m *Model) rebuildSearch() {
	m.searchMatches = nil
	if m.searchQuery == "" {
		return
	}
	query := strings.ToLower(m.searchQuery)
	for i, raw := range m.diffLines {
		line := stripAnsi(raw)
		if isDiffMetadata(line) || !isDiffContentLine(line) {
			continue
		}
		if strings.Contains(strings.ToLower(line[1:]), query) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
}

func (m Model) isSearchMatch(line int) bool {
	i := sort.SearchInts(m.searchMatches, line)
	return i < len(m.searchMatches) && m.searchMatches[i] == line
}

func (m *Model) jumpSearch(direction int) {
	if len(m.searchMatches) == 0 {
		m.inputBuffer = ""
		return
	}
	count := m.consumeCount()
	index := sort.SearchInts(m.searchMatches, m.diffCursor)
	if direction > 0 {
		if index < len(m.searchMatches) && m.searchMatches[index] == m.diffCursor {
			index++
		}
		index = (index + (count-1)%len(m.searchMatches)) % len(m.searchMatches)
	} else {
		index = (index - 1 - (count-1)%len(m.searchMatches) + len(m.searchMatches)*2) % len(m.searchMatches)
	}
	m.diffCursor = m.searchMatches[index]
	m.centerDiffCursor()
}

func hunkStarts(lines []string) []int {
	var starts []int
	inHunk := false
	for i, raw := range lines {
		line := stripAnsi(raw)
		if isDiffMetadata(line) {
			inHunk = true
			continue
		}
		if inHunk && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) {
			starts = append(starts, i)
			inHunk = false
		}
	}
	return starts
}

func (m *Model) jumpHunk(direction int) {
	if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && item.IsDir {
		m.inputBuffer = ""
		return
	}
	if len(m.diffHunks) == 0 {
		m.inputBuffer = ""
		return
	}
	m.focus = FocusDiff
	m.visualMode = false
	m.updateTreeFocus()
	count := min(max(1, m.getRepeatCount()), len(m.diffHunks))
	for i := 0; i < count; i++ {
		index := sort.SearchInts(m.diffHunks, m.diffCursor)
		if direction > 0 {
			if index < len(m.diffHunks) && m.diffHunks[index] == m.diffCursor {
				index++
			}
			if index == len(m.diffHunks) {
				break
			}
		} else {
			index--
			if index < 0 {
				break
			}
		}
		m.diffCursor = m.diffHunks[index]
	}
	m.centerDiffCursor()
}

func (m Model) navigationStatus() string {
	var parts []string
	if m.fileFilter != "" {
		parts = append(parts, fmt.Sprintf("Files /%s · %d matches · Esc clears", m.fileFilter, len(m.fileList.Items())))
	}
	if m.searchQuery != "" {
		current := 0
		if m.isSearchMatch(m.diffCursor) {
			current = sort.SearchInts(m.searchMatches, m.diffCursor) + 1
		}
		if len(m.searchMatches) == 0 {
			parts = append(parts, fmt.Sprintf("/%s · no matches", m.searchQuery))
		} else {
			parts = append(parts, fmt.Sprintf("/%s · %d/%d lines · n/N next/previous", m.searchQuery, current, len(m.searchMatches)))
		}
	}
	return strings.Join(parts, "  │  ")
}

func (m Model) renderNavigationBar() string {
	text := m.navigationStatus()
	if m.promptMode != promptNone {
		label := "/ "
		if m.promptMode == promptFiles {
			label = "/files "
		}
		text = S.CmdLinePrompt.Render(label) + m.promptInput.View()
	}
	return S.CmdLineStyle.Width(m.width).Render(ansi.Truncate(text, m.width, "…"))
}

func (m Model) comparisonLabel() string {
	label := m.currentBranch + " ➜ " + m.targetBranch
	switch m.reviewOptions.Scope {
	case review.Staged:
		label = m.targetBranch + " ➜ index (staged)"
	case review.Unstaged:
		label = "index ➜ working tree (unstaged)"
	}
	if m.fromStdin {
		label = "piped patch"
	}
	if m.reviewOptions.NoUntracked {
		label += " · tracked only"
	}
	return label
}

func (m Model) emptyMessage() string {
	if m.fromStdin {
		return "No changes in piped patch"
	}
	switch m.reviewOptions.Scope {
	case review.Staged:
		return "No staged changes against " + m.targetBranch
	case review.Unstaged:
		return "No unstaged changes"
	}
	return "No changes found against " + m.targetBranch
}
