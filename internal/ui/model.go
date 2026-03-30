package ui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/tree"
	"github.com/xguot/difi/internal/vcs"
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

func isDiffContentLine(cleanLine string) bool {
	cleanLine = strings.TrimRight(cleanLine, "\r")
	return strings.HasPrefix(cleanLine, " ") ||
		strings.HasPrefix(cleanLine, "+") ||
		strings.HasPrefix(cleanLine, "-")
}

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

	m.diffViewport.YOffset = offset
}

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
		if isDiffContentLine(cleanLine) {
			return curr
		}
		curr += dir
	}

	curr = idx
	for curr >= 0 && curr < len(m.diffLines) {
		cleanLine := stripAnsi(m.diffLines[curr])
		if isDiffContentLine(cleanLine) {
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
