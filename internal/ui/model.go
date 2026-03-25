package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/oug-t/difi/internal/config"
	"github.com/oug-t/difi/internal/tree"
	"github.com/oug-t/difi/internal/vcs"
)

type Focus int

const (
	FocusTree Focus = iota
	FocusDiff
)

var ansiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
var bgAnsiRe = regexp.MustCompile(`\x1b\[48;2;\d+;\d+;\d+m|\x1b\[4[0-9]m`)

type StatsMsg struct {
	Added   int
	Deleted int
	ByFile  map[string][2]int
}

type Model struct {
	fileList     list.Model
	treeState    *tree.FileTree
	treeDelegate TreeDelegate
	diffViewport viewport.Model

	selectedPath  string
	currentBranch string
	targetBranch  string
	repoName      string

	statsAdded   int
	statsDeleted int

	currentFileAdded   int
	currentFileDeleted int

	fileStats map[string][2]int

	diffContent     string
	diffLines       []string
	diffHighlighted []string
	diffCursor      int
	visualMode      bool // Visual selection mode
	visualStart     int  // Anchor for visual selection

	inputBuffer string
	pendingZ    bool

	focus    Focus
	showHelp bool

	width, height int

	pipedDiff string
	vcs       vcs.VCS
}

func NewModel(cfg config.Config, targetBranch string, pipedDiff string, vcsClient vcs.VCS) Model {
	InitStyles(cfg)

	var files []string
	if pipedDiff != "" {
		files = vcsClient.ParseFilesFromDiff(pipedDiff)
	} else {
		files, _ = vcsClient.ListChangedFiles(targetBranch)
	}
	t := tree.New(files)
	items := t.Items()

	delegate := TreeDelegate{
		Config:  cfg,
		Focused: true,
	}

	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	m := Model{
		fileList:      l,
		treeState:     t,
		treeDelegate:  delegate,
		diffViewport:  viewport.New(0, 0),
		focus:         FocusTree,
		currentBranch: vcsClient.GetCurrentBranch(),
		targetBranch:  targetBranch,
		repoName:      vcsClient.GetRepoName(),
		showHelp:      false,
		inputBuffer:   "",
		pendingZ:      false,
		pipedDiff:     pipedDiff,
		vcs:           vcsClient,
		visualMode:    false,
		visualStart:   0,
	}

	for idx, item := range items {
		if ti, ok := item.(tree.TreeItem); ok && !ti.IsDir {
			m.selectedPath = ti.FullPath
			m.fileList.Select(idx)
			break
		}
	}
	return m
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.selectedPath != "" {
		if m.pipedDiff != "" {
			cmds = append(cmds, func() tea.Msg {
				return vcs.DiffMsg{Content: m.vcs.ExtractFileDiff(m.pipedDiff, m.selectedPath)}
			})
		} else {
			cmds = append(cmds, m.vcs.DiffCmd(m.targetBranch, m.selectedPath))
		}
	}

	if m.pipedDiff == "" {
		cmds = append(cmds, m.fetchStatsCmd(m.targetBranch))
	} else {
		cmds = append(cmds, m.computePipedStatsCmd())
	}

	return tea.Batch(cmds...)
}

func (m Model) fetchStatsCmd(target string) tea.Cmd {
	return func() tea.Msg {
		added, deleted, err := m.vcs.DiffStats(target)
		if err != nil {
			return nil
		}
		byFile, _ := m.vcs.DiffStatsByFile(target)
		return StatsMsg{Added: added, Deleted: deleted, ByFile: byFile}
	}
}

func (m Model) computePipedStatsCmd() tea.Cmd {
	return func() tea.Msg {
		byFile := make(map[string][2]int)
		var totalAdded, totalDeleted int
		var currentFile string

		for _, line := range strings.Split(m.pipedDiff, "\n") {
			clean := stripAnsi(line)
			if strings.HasPrefix(clean, "diff --git ") {
				parts := strings.Fields(clean)
				if len(parts) >= 4 {
					currentFile = strings.TrimPrefix(parts[3], "b/")
				}
			} else if strings.HasPrefix(clean, "diff -r ") {
				parts := strings.Fields(clean)
				if len(parts) >= 3 {
					currentFile = parts[len(parts)-1]
				}
			} else if currentFile != "" {
				if strings.HasPrefix(clean, "+") && !strings.HasPrefix(clean, "+++") {
					s := byFile[currentFile]
					s[0]++
					byFile[currentFile] = s
					totalAdded++
				} else if strings.HasPrefix(clean, "-") && !strings.HasPrefix(clean, "---") {
					s := byFile[currentFile]
					s[1]++
					byFile[currentFile] = s
					totalDeleted++
				}
			}
		}
		return StatsMsg{Added: totalAdded, Deleted: totalDeleted, ByFile: byFile}
	}
}

func (m *Model) getRepeatCount() int {
	if m.inputBuffer == "" {
		return 1
	}
	count, err := strconv.Atoi(m.inputBuffer)
	if err != nil {
		return 1
	}
	m.inputBuffer = ""
	return count
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	keyHandled := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case StatsMsg:
		m.statsAdded = msg.Added
		m.statsDeleted = msg.Deleted
		if msg.ByFile != nil {
			m.fileStats = msg.ByFile
		}

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if len(m.fileList.Items()) == 0 {
			return m, nil
		}

		if m.pendingZ {
			m.pendingZ = false
			if m.focus == FocusDiff {
				switch msg.String() {
				case "z", ".":
					m.centerDiffCursor()
				case "t":
					m.setYOffset(m.diffCursor)
				case "b":
					m.setYOffset(m.diffCursor - m.diffViewport.Height + 1)
				}
			}
			return m, nil
		}

		if len(msg.String()) == 1 && strings.ContainsAny(msg.String(), "0123456789") {
			m.inputBuffer += msg.String()
			return m, nil
		}

		if msg.String() == "?" {
			m.showHelp = !m.showHelp
			m.updateSizes()
			return m, nil
		}

		switch msg.String() {
		case "V":
			if m.focus == FocusDiff {
				m.visualMode = !m.visualMode
				if m.visualMode {
					m.visualStart = m.diffCursor
				}
			}
			m.inputBuffer = ""

		case "esc":
			m.visualMode = false
			m.inputBuffer = ""

		case "tab":
			m.visualMode = false
			if m.focus == FocusTree {
				if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && item.IsDir {
					return m, nil
				}
				m.focus = FocusDiff
			} else {
				m.focus = FocusTree
			}
			m.updateTreeFocus()
			m.inputBuffer = ""

		case "ctrl+h", "[":
			m.visualMode = false
			m.focus = FocusTree
			m.updateTreeFocus()
			m.inputBuffer = ""

		case "ctrl+l", "]":
			m.visualMode = false
			if m.focus == FocusTree {
				if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && item.IsDir {
					return m, nil
				}
			}
			m.focus = FocusDiff
			m.updateTreeFocus()
			m.inputBuffer = ""

		case "h", "left":
			m.visualMode = false
			keyHandled = true
			m.focus = FocusTree
			m.updateTreeFocus()
			m.inputBuffer = ""

		case "l", "right":
			m.visualMode = false
			keyHandled = true
			if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && item.IsDir {
				return m, nil
			}
			m.focus = FocusDiff
			m.updateTreeFocus()
			m.inputBuffer = ""

		case "enter", "e":
			m.visualMode = false
			if m.focus == FocusTree && msg.String() == "enter" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					m.treeState.ToggleExpand(i.FullPath)
					m.fileList.SetItems(m.treeState.Items())
					return m, nil
				}
			}

			if m.selectedPath != "" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					return m, nil
				}

				line := 0
				if m.focus == FocusDiff {
					line = m.vcs.CalculateFileLine(m.diffContent, m.diffCursor)
				} else {
					line = m.vcs.CalculateFileLine(m.diffContent, 0)
				}
				m.inputBuffer = ""
				return m, m.vcs.OpenEditorCmd(m.selectedPath, line, m.targetBranch, m.treeDelegate.Config.Editor)
			}

		case "z":
			if m.focus == FocusDiff {
				m.pendingZ = true
				return m, nil
			}

		case "H":
			if m.focus == FocusDiff {
				m.diffCursor = m.snapCursor(m.diffViewport.YOffset, 1)
			}

		case "M":
			if m.focus == FocusDiff {
				half := m.diffViewport.Height / 2
				m.diffCursor = m.snapCursor(m.diffViewport.YOffset+half, 1)
			}

		case "L":
			if m.focus == FocusDiff {
				end := m.diffViewport.YOffset + m.diffViewport.Height - 1
				m.diffCursor = m.snapCursor(end, -1)
			}

		case "ctrl+d":
			if m.focus == FocusDiff {
				target := m.diffCursor + m.diffViewport.Height/2
				m.diffCursor = m.snapCursor(target, 1)
				m.centerDiffCursor()
			}
			m.inputBuffer = ""

		case "ctrl+u":
			if m.focus == FocusDiff {
				target := m.diffCursor - m.diffViewport.Height/2
				m.diffCursor = m.snapCursor(target, -1)
				m.centerDiffCursor()
			}
			m.inputBuffer = ""

		case "j", "down":
			keyHandled = true
			for i := 0; i < m.getRepeatCount(); i++ {
				if m.focus == FocusDiff {
					m.diffCursor = m.snapCursor(m.diffCursor+1, 1)
					m.handleScrolling()
				} else {
					m.fileList.CursorDown()
				}
			}
			m.inputBuffer = ""

		case "k", "up":
			keyHandled = true
			for i := 0; i < m.getRepeatCount(); i++ {
				if m.focus == FocusDiff {
					m.diffCursor = m.snapCursor(m.diffCursor-1, -1)
					m.handleScrolling()
				} else {
					m.fileList.CursorUp()
				}
			}
			m.inputBuffer = ""

		case "g":
			if m.focus == FocusDiff {
				if m.inputBuffer == "g" {
					m.diffCursor = m.snapCursor(0, 1)
					m.setYOffset(m.diffCursor)
					m.inputBuffer = ""
				} else {
					m.inputBuffer = "g"
				}
			}

		case "G":
			if m.focus == FocusDiff {
				count, err := strconv.Atoi(m.inputBuffer)
				if err == nil && count > 0 {
					target := count - 1
					m.diffCursor = m.snapCursor(target, 1)
				} else {
					m.diffCursor = m.snapCursor(len(m.diffLines)-1, -1)
				}
				m.setYOffset(m.diffCursor - m.diffViewport.Height + 1)
				m.inputBuffer = ""
			}

		default:
			m.inputBuffer = ""
		}
	}

	if len(m.fileList.Items()) > 0 && m.focus == FocusTree {
		if !keyHandled {
			m.fileList, cmd = m.fileList.Update(msg)
			cmds = append(cmds, cmd)
		}

		if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok {
			if !item.IsDir && item.FullPath != m.selectedPath {
				m.selectedPath = item.FullPath
				m.diffCursor = 0
				m.visualMode = false
				m.diffViewport.GotoTop()
				if m.pipedDiff != "" {
					cmds = append(cmds, func() tea.Msg {
						return vcs.DiffMsg{Content: m.vcs.ExtractFileDiff(m.pipedDiff, m.selectedPath)}
					})
				} else {
					cmds = append(cmds, m.vcs.DiffCmd(m.targetBranch, m.selectedPath))
				}
			}
		}
	}

	switch msg := msg.(type) {
	case vcs.DiffMsg:
		fullLines := strings.Split(msg.Content, "\n")
		var cleanLines, hlLines []string
		var added, deleted int
		foundHunk := false

		ext := filepath.Ext(m.selectedPath)
		if len(ext) > 0 {
			ext = ext[1:]
		} else {
			ext = "txt"
		}

		isGitTheme := m.treeDelegate.Config.UI.Theme == "git"

		for _, line := range fullLines {
			cleanLine := stripAnsi(line)

			if strings.HasPrefix(cleanLine, "@@") {
				foundHunk = true
			}

			if !foundHunk {
				continue
			}

			cleanLines = append(cleanLines, line)

			isAdd := strings.HasPrefix(cleanLine, "+") && !strings.HasPrefix(cleanLine, "+++")
			isDel := strings.HasPrefix(cleanLine, "-") && !strings.HasPrefix(cleanLine, "---")

			if isAdd {
				added++
			} else if isDel {
				deleted++
			}

			codeContent := cleanLine
			if len(codeContent) > 0 && (isAdd || isDel || strings.HasPrefix(codeContent, " ")) {
				codeContent = codeContent[1:]
			}

			if isGitTheme {
				hlLines = append(hlLines, codeContent)
			} else {
				var buf strings.Builder
				err := quick.Highlight(&buf, codeContent, ext, "terminal16m", "nord")
				if err == nil && buf.String() != "" {
					hlLines = append(hlLines, strings.TrimSuffix(buf.String(), "\n"))
				} else {
					hlLines = append(hlLines, codeContent)
				}
			}
		}

		m.diffLines = cleanLines
		m.diffHighlighted = hlLines
		m.currentFileAdded = added
		m.currentFileDeleted = deleted
		m.diffCursor = m.snapCursor(0, 1) // Ensure we start on a visible line

	case vcs.EditorFinishedMsg:
		if m.pipedDiff != "" {
			return m, func() tea.Msg {
				return vcs.DiffMsg{Content: m.vcs.ExtractFileDiff(m.pipedDiff, m.selectedPath)}
			}
		}
		return m, m.vcs.DiffCmd(m.targetBranch, m.selectedPath)
	}

	return m, tea.Batch(cmds...)
}

// snapCursor prevents highlight vanishing by snapping to non-metadata lines.
func (m *Model) snapCursor(idx int, dir int) int {
	if len(m.diffLines) == 0 {
		return 0
	}

	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.diffLines) {
		idx = len(m.diffLines) - 1
	}

	curr := idx
	for curr >= 0 && curr < len(m.diffLines) {
		cleanLine := stripAnsi(m.diffLines[curr])
		if !isDiffMetadata(cleanLine) {
			return curr
		}
		curr += dir
	}

	// 3. Fallback: if we hit a boundary, scan in the reverse direction
	curr = idx
	for curr >= 0 && curr < len(m.diffLines) {
		cleanLine := stripAnsi(m.diffLines[curr])
		if !isDiffMetadata(cleanLine) {
			return curr
		}
		curr -= dir
	}

	return m.diffCursor
}

func (m *Model) handleScrolling() {
	if m.diffCursor < m.diffViewport.YOffset {
		m.setYOffset(m.diffCursor)
	} else if m.diffCursor >= m.diffViewport.YOffset+m.diffViewport.Height {
		m.setYOffset(m.diffCursor - m.diffViewport.Height + 1)
	}
}

func (m *Model) centerDiffCursor() {
	targetOffset := m.diffCursor - (m.diffViewport.Height / 2)
	m.setYOffset(targetOffset)
}

func (m *Model) updateSizes() {
	reservedHeight := 2
	if m.showHelp {
		reservedHeight += 6
	}

	contentHeight := m.height - reservedHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	treeWidth := int(float64(m.width) * 0.20)
	if treeWidth < 20 {
		treeWidth = 20
	}

	treePaneOverhead := 4
	treeInnerWidth := treeWidth - treePaneOverhead
	if treeInnerWidth < 10 {
		treeInnerWidth = 10
	}

	listHeight := contentHeight - 2
	if listHeight < 1 {
		listHeight = 1
	}
	m.fileList.SetSize(treeInnerWidth, listHeight)

	m.diffViewport.Width = m.width - treeWidth
	m.diffViewport.Height = listHeight
}

func (m *Model) updateTreeFocus() {
	m.treeDelegate.Focused = (m.focus == FocusTree)
	m.fileList.SetDelegate(m.treeDelegate)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	topBar := m.renderTopBar()

	var mainContent string
	contentHeight := m.height - 2
	if m.showHelp {
		contentHeight -= 6
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	if len(m.fileList.Items()) == 0 {
		mainContent = m.renderEmptyState(m.width, contentHeight, "No changes found against "+m.targetBranch)
	} else {
		treeStyle := PaneStyle
		if m.focus == FocusTree {
			treeStyle = FocusedPaneStyle
		}

		treeView := treeStyle.Copy().
			Width(m.fileList.Width()).
			Height(m.fileList.Height()).
			MaxHeight(m.fileList.Height() + 2).
			Render(m.fileList.View())

		var rightPaneView string
		selectedItem, ok := m.fileList.SelectedItem().(tree.TreeItem)

		if ok && selectedItem.IsDir {
			rightPaneView = m.renderEmptyState(m.diffViewport.Width, m.diffViewport.Height, "Directory: "+selectedItem.Name)
		} else {
			var renderedDiff strings.Builder

			viewportHeight := m.diffViewport.Height
			start := m.diffViewport.YOffset
			end := start + viewportHeight
			if end > len(m.diffLines) {
				end = len(m.diffLines)
			}

			maxLineWidth := m.diffViewport.Width - 7
			if maxLineWidth < 1 {
				maxLineWidth = 1
			}

			isGitTheme := m.treeDelegate.Config.UI.Theme == "git"

			for i := start; i < end; i++ {
				rawLine := m.diffLines[i]
				cleanLine := stripAnsi(rawLine)

				if isDiffMetadata(cleanLine) {
					if end < len(m.diffLines) {
						end++
					}
					continue
				}

				isAdd := strings.HasPrefix(cleanLine, "+")
				isDel := strings.HasPrefix(cleanLine, "-")

				codeContent := cleanLine
				if len(codeContent) > 0 && (isAdd || isDel || strings.HasPrefix(codeContent, " ")) {
					codeContent = codeContent[1:]
				}

				// Active line evaluation handles both single cursor and Visual Mode
				isCursor := false
				if m.focus == FocusDiff {
					if m.visualMode {
						minIdx, maxIdx := m.visualStart, m.diffCursor
						if minIdx > maxIdx {
							minIdx, maxIdx = maxIdx, minIdx
						}
						isCursor = (i >= minIdx && i <= maxIdx)
					} else {
						isCursor = (i == m.diffCursor)
					}
				}

				separator := "│"
				if isCursor {
					separator = "┃"
				}

				var gutterStr string
				if isAdd {
					gutterStr = "+ " + separator + " "
				} else if isDel {
					gutterStr = "- " + separator + " "
				} else {
					gutterStr = "  " + separator + " "
				}

				var numStr string
				mode := "relative"

				if mode != "hidden" {
					if isCursor && mode == "hybrid" {
						realLine := m.vcs.CalculateFileLine(m.diffContent, m.diffCursor) // always anchor to actual cursor
						numStr = fmt.Sprintf("%d", realLine)
					} else if isCursor && mode == "relative" {
						numStr = "0"
					} else if mode == "absolute" {
						numStr = fmt.Sprintf("%d", i+1)
					} else {
						dist := int(math.Abs(float64(i - m.diffCursor)))
						numStr = fmt.Sprintf("%d", dist)
					}
				}

				lineNumRendered := ""
				if numStr != "" {
					lineNumRendered = LineNumberStyle.Render(numStr)
				}

				var line string

				if isCursor {
					fullStr := gutterStr + ansi.Truncate(codeContent, maxLineWidth-4, "")

					visibleLen := lipgloss.Width(fullStr)
					padLen := maxLineWidth - visibleLen
					if padLen > 0 {
						fullStr += strings.Repeat(" ", padLen)
					}

					if isAdd {
						line = CursorAddStyle.Copy().Width(maxLineWidth).Render(fullStr)
					} else if isDel {
						line = CursorDelStyle.Copy().Width(maxLineWidth).Render(fullStr)
					} else {
						line = CursorNormalStyle.Copy().Width(maxLineWidth).Render(fullStr)
					}
				} else {
					var hlCode string
					var gutter string

					if isGitTheme {
						if isAdd {
							hlCode = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(codeContent)
							gutter = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(gutterStr)
						} else if isDel {
							hlCode = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(codeContent)
							gutter = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(gutterStr)
						} else {
							hlCode = codeContent
							gutter = DiffCtxGutter.Render(gutterStr)
						}
					} else {
						if i < len(m.diffHighlighted) {
							hlCode = m.diffHighlighted[i]
							hlCode = bgAnsiRe.ReplaceAllString(hlCode, "")
						}

						if isAdd {
							gutter = DiffAddGutter.Render(gutterStr)
						} else if isDel {
							gutter = DiffDelGutter.Render(gutterStr)
						} else {
							gutter = DiffCtxGutter.Render(gutterStr)
						}
					}

					hlCode = ansi.Truncate(hlCode, maxLineWidth-4, "")

					line = gutter + hlCode
				}

				renderedDiff.WriteString(lineNumRendered + line + "\n")
			}

			diffContentStr := "\n" + strings.TrimRight(renderedDiff.String(), "\n")

			rightPaneView = DiffStyle.Copy().
				Width(m.diffViewport.Width).
				Height(viewportHeight).
				Render(diffContentStr)
		}

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treeView, rightPaneView)
	}

	var bottomBar string
	if m.showHelp {
		bottomBar = m.renderHelpDrawer()
	} else {
		bottomBar = m.viewStatusBar()
	}

	return lipgloss.JoinVertical(lipgloss.Top, topBar, mainContent, bottomBar)
}

func (m Model) renderTopBar() string {
	vcsType := "git"
	if m.vcs != nil {
		if _, isHg := m.vcs.(vcs.HgVCS); isHg {
			vcsType = "hg"
		}
	}

	repoStats := ""
	if m.statsAdded > 0 || m.statsDeleted > 0 {
		repoStats = fmt.Sprintf(" +%d -%d", m.statsAdded, m.statsDeleted)
	}

	info := fmt.Sprintf(" %s:%s  %s ➜ %s%s", m.repoName, vcsType, m.currentBranch, m.targetBranch, repoStats)
	leftSide := TopInfoStyle.Render(info)

	rightSide := ""
	if selectedItem, ok := m.fileList.SelectedItem().(tree.TreeItem); ok {
		var displayPath string
		var statsAdded, statsDeleted int

		if selectedItem.IsDir {
			displayPath = selectedItem.FullPath + "/"
			prefix := selectedItem.FullPath + "/"
			for filePath, stats := range m.fileStats {
				if strings.HasPrefix(filePath, prefix) {
					statsAdded += stats[0]
					statsDeleted += stats[1]
				}
			}
		} else {
			displayPath = selectedItem.FullPath
			if fs, ok := m.fileStats[selectedItem.FullPath]; ok {
				statsAdded = fs[0]
				statsDeleted = fs[1]
			} else {
				statsAdded = m.currentFileAdded
				statsDeleted = m.currentFileDeleted
			}
		}

		fileStats := ""
		if statsAdded > 0 || statsDeleted > 0 {
			added := TopStatsAddedStyle.Render(fmt.Sprintf("+%d", statsAdded))
			deleted := TopStatsDeletedStyle.Render(fmt.Sprintf("-%d", statsDeleted))
			fileStats = lipgloss.JoinHorizontal(lipgloss.Center, added, deleted)
		}

		fileStatsWidth := lipgloss.Width(fileStats)
		maxPathWidth := m.width - lipgloss.Width(leftSide) - fileStatsWidth - 4
		if maxPathWidth < 10 {
			maxPathWidth = 10
		}

		truncPath := ansi.Truncate(displayPath, maxPathWidth, "…")
		if fileStats != "" {
			rightSide = truncPath + " " + fileStats
		} else {
			rightSide = truncPath
		}
	}

	availWidth := m.width - lipgloss.Width(leftSide) - lipgloss.Width(rightSide)
	if availWidth < 0 {
		availWidth = 0
	}

	padding := strings.Repeat(" ", availWidth)
	finalBar := lipgloss.JoinHorizontal(lipgloss.Top, leftSide, padding, rightSide)

	return TopBarStyle.Width(m.width).Render(finalBar)
}

func (m Model) viewStatusBar() string {
	shortcuts := StatusKeyStyle.Render("? Help  q Quit  Tab Switch  V Visual")
	return StatusBarStyle.Width(m.width).Render(shortcuts)
}

func (m Model) renderHelpDrawer() string {
	col1 := lipgloss.JoinVertical(lipgloss.Left,
		HelpTextStyle.Render("↑/k   Move Up"),
		HelpTextStyle.Render("↓/j   Move Down"),
	)
	col2 := lipgloss.JoinVertical(lipgloss.Left,
		HelpTextStyle.Render("←/h   Left Panel"),
		HelpTextStyle.Render("→/l   Right Panel"),
	)
	col3 := lipgloss.JoinVertical(lipgloss.Left,
		HelpTextStyle.Render("C-d/u Page Dn/Up"),
		HelpTextStyle.Render("zz/zt Scroll View"),
	)
	col4 := lipgloss.JoinVertical(lipgloss.Left,
		HelpTextStyle.Render("H/M/L Move Cursor"),
		HelpTextStyle.Render("e     Edit File"),
	)
	col5 := lipgloss.JoinVertical(lipgloss.Left,
		HelpTextStyle.Render("V     Visual Mode"),
		HelpTextStyle.Render("esc   Cancel Visual"),
	)

	return HelpDrawerStyle.Copy().
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			col1, lipgloss.NewStyle().Width(4).Render(""),
			col2, lipgloss.NewStyle().Width(4).Render(""),
			col3, lipgloss.NewStyle().Width(4).Render(""),
			col4, lipgloss.NewStyle().Width(4).Render(""),
			col5,
		))
}

func (m Model) renderEmptyState(w, h int, statusMsg string) string {
	logo := EmptyLogoStyle.Render("difi")
	desc := EmptyDescStyle.Render("A calm, focused way to review Git & Mercurial diffs.")
	status := EmptyStatusStyle.Render(statusMsg)

	usageHeader := EmptyHeaderStyle.Render("Usage Patterns")
	cmd1 := lipgloss.NewStyle().Foreground(ColorText).Render("difi")
	desc1 := EmptyCodeStyle.Render("Auto-detect VCS, diff against main/tip")
	cmd2 := lipgloss.NewStyle().Foreground(ColorText).Render("difi --vcs git")
	desc2 := EmptyCodeStyle.Render("Force Git mode")
	cmd3 := lipgloss.NewStyle().Foreground(ColorText).Render("difi --vcs hg")
	desc3 := EmptyCodeStyle.Render("Force Mercurial mode")

	usageBlock := lipgloss.JoinVertical(lipgloss.Left,
		usageHeader,
		lipgloss.JoinHorizontal(lipgloss.Left, cmd1, "    ", desc1),
		lipgloss.JoinHorizontal(lipgloss.Left, cmd2, "    ", desc2),
		lipgloss.JoinHorizontal(lipgloss.Left, cmd3, "    ", desc3),
	)

	navHeader := EmptyHeaderStyle.Render("Navigation")
	key1 := lipgloss.NewStyle().Foreground(ColorText).Render("Tab")
	key2 := lipgloss.NewStyle().Foreground(ColorText).Render("j/k")
	keyDesc1 := EmptyCodeStyle.Render("Switch panels")
	keyDesc2 := EmptyCodeStyle.Render("Move cursor")

	navBlock := lipgloss.JoinVertical(lipgloss.Left,
		navHeader,
		lipgloss.JoinHorizontal(lipgloss.Left, key1, "    ", keyDesc1),
		lipgloss.JoinHorizontal(lipgloss.Left, key2, "    ", keyDesc2),
	)

	nvimHeader := EmptyHeaderStyle.Render("Neovim Integration")
	nvim1 := lipgloss.NewStyle().Foreground(ColorText).Render("oug-t/difi.nvim")
	nvimDesc1 := EmptyCodeStyle.Render("Install plugin")
	nvim2 := lipgloss.NewStyle().Foreground(ColorText).Render("Press 'e'")
	nvimDesc2 := EmptyCodeStyle.Render("Edit with context")

	nvimBlock := lipgloss.JoinVertical(lipgloss.Left,
		nvimHeader,
		lipgloss.JoinHorizontal(lipgloss.Left, nvim1, "  ", nvimDesc1),
		lipgloss.JoinHorizontal(lipgloss.Left, nvim2, "          ", nvimDesc2),
	)

	var guides string
	if w > 80 {
		guides = lipgloss.JoinHorizontal(lipgloss.Top,
			usageBlock, lipgloss.NewStyle().Width(6).Render(""),
			navBlock, lipgloss.NewStyle().Width(6).Render(""),
			nvimBlock,
		)
	} else {
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, usageBlock, lipgloss.NewStyle().Width(4).Render(""), navBlock)
		guides = lipgloss.JoinVertical(lipgloss.Left,
			topRow,
			lipgloss.NewStyle().Height(1).Render(""),
			nvimBlock,
		)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		desc,
		status,
		lipgloss.NewStyle().Height(1).Render(""),
		guides,
	)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func stripAnsi(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}

func isDiffMetadata(cleanLine string) bool {
	return strings.HasPrefix(cleanLine, "diff --git") ||
		strings.HasPrefix(cleanLine, "diff -r ") ||
		strings.HasPrefix(cleanLine, "index ") ||
		strings.HasPrefix(cleanLine, "new file mode") ||
		strings.HasPrefix(cleanLine, "old mode") ||
		strings.HasPrefix(cleanLine, "--- a/") ||
		strings.HasPrefix(cleanLine, "--- /dev/") ||
		strings.HasPrefix(cleanLine, "+++ b/") ||
		strings.HasPrefix(cleanLine, "+++ /dev/") ||
		strings.HasPrefix(cleanLine, "@@")
}

// setYOffset safely sets the viewport offset without relying on viewport's internal content.
func (m *Model) setYOffset(offset int) {
	maxOffset := len(m.diffLines) - m.diffViewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	// FIX: Assign directly. Do not use m.diffViewport.SetYOffset()!
	m.diffViewport.YOffset = offset
}
