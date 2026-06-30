package ui

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/xguot/difi/internal/tree"
	"github.com/xguot/difi/internal/vcs"
)

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
		// ── Command-line mode ──────────────────────────────────
		if m.commandMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.commandMode = false
				m.commandBuffer = ""
				m.resetTabCycle()
				return m, nil

			case "enter":
				quit, cmd := m.executeCommand()
				m.commandMode = false
				m.commandBuffer = ""
				if quit {
					return m, tea.Quit
				}
				if cmd != nil {
					return m, cmd
				}
				return m, nil

			case "backspace":
				if len(m.commandBuffer) > 0 {
					m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
				}
				m.resetTabCycle()
				return m, nil

			case "tab":
				m.tabComplete()
				return m, nil

			default:
				// Only accept printable single characters
				if len(msg.String()) == 1 && msg.String() >= " " {
					m.commandBuffer += msg.String()
					m.resetTabCycle()
				}
				return m, nil
			}
		}

		// ── Help buffer mode ─────────────────────────────────
		if m.helpMode {
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				m.exitHelpMode()
				return m, nil
			case "j", "down":
				m.diffViewport.LineDown(1)
			case "k", "up":
				m.diffViewport.LineUp(1)
			case "h", "left":
				if m.helpXOffset > 0 {
					m.helpXOffset -= 4
				}
			case "l", "right":
				m.helpXOffset += 4
			case "H":
				m.diffViewport.GotoTop()
			case "M":
				mid := len(m.helpLines)/2 - m.diffViewport.Height/2
				if mid < 0 {
					mid = 0
				}
				m.diffViewport.SetYOffset(mid)
			case "L":
				m.diffViewport.GotoBottom()
			case "ctrl+d":
				m.diffViewport.HalfViewDown()
			case "ctrl+u":
				m.diffViewport.HalfViewUp()
			case "ctrl+f", "pgdown":
				m.diffViewport.ViewDown()
			case "ctrl+b", "pgup":
				m.diffViewport.ViewUp()
			case "g":
				if m.inputBuffer == "g" {
					m.diffViewport.GotoTop()
					m.inputBuffer = ""
				} else {
					m.inputBuffer = "g"
				}
			case "G":
				m.diffViewport.GotoBottom()
				m.inputBuffer = ""
			case "?":
				m.showHelp = !m.showHelp
				m.updateSizes()
			default:
				m.inputBuffer = ""
			}
			return m, nil
		}

		// ── Normal mode keys ──────────────────────────────────
		if msg.String() == ":" {
			m.commandMode = true
			m.commandBuffer = ""
			m.pendingZZ = false
			m.resetTabCycle()
			return m, nil
		}

		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// ZZ — press Z twice in normal mode to quit
		if msg.String() == "Z" {
			if m.pendingZZ {
				return m, tea.Quit
			}
			m.pendingZZ = true
			return m, nil
		}
		m.pendingZZ = false

		if msg.String() == "?" {
			m.showHelp = !m.showHelp
			m.updateSizes()
			return m, nil
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

		case "s":
			if item, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && !item.IsDir {
				m.toggleSplitMode()
				m.handleScrolling()
			}
			m.inputBuffer = ""

		case "f":
			if m.focus == FocusTree {
				m.flatMode = !m.flatMode
				m.fileList.SetItems(m.treeState.Items(m.flatMode))
				for i, item := range m.fileList.Items() {
					if ti, ok := item.(tree.TreeItem); ok && ti.FullPath == m.selectedPath {
						m.fileList.Select(i)
						break
					}
				}
				return m, nil
			}

		case "enter", "e":
			m.visualMode = false
			if m.focus == FocusTree && msg.String() == "enter" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					m.treeState.ToggleExpand(i.FullPath)
					m.fileList.SetItems(m.treeState.Items(m.flatMode))
					return m, nil
				}
			}

			if m.selectedPath != "" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					return m, nil
				}

				line := 0
				if m.focus == FocusDiff {
					line = m.vcs.CalculateFileLine(m.diffLines, m.cursorRawIdx())
				} else {
					line = m.vcs.CalculateFileLine(m.diffLines, 0)
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
					m.diffCursor = m.snapCursor(m.diffLineCount()-1, -1)
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

		// Determine Chroma theme name from active ThemeColors
		t := ActiveTheme()
		chromaStyle := "nord"
		if t != nil && !isGitTheme {
			chromaStyle = t.ChromaStyle
		}

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
				err := quick.Highlight(&buf, codeContent, ext, "terminal16m", chromaStyle)
				if err == nil && buf.String() != "" {
					hlLines = append(hlLines, strings.TrimSuffix(buf.String(), "\n"))
				} else {
					hlLines = append(hlLines, codeContent)
				}
			}
		}

		for len(cleanLines) > 0 {
			lastLine := strings.TrimRight(stripAnsi(cleanLines[len(cleanLines)-1]), "\r")
			if lastLine != "" {
				break
			}
			cleanLines = cleanLines[:len(cleanLines)-1]
			hlLines = hlLines[:len(hlLines)-1]
		}

		m.diffLines = cleanLines
		m.diffHighlighted = hlLines
		m.currentFileAdded = added
		m.currentFileDeleted = deleted
		m.rebuildSplitRows()
		m.diffCursor = m.snapCursor(0, 1)

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

// executeCommand parses and executes a vim-style command from the command buffer.
// Returns (shouldQuit, asyncCmd). asyncCmd is non-nil when the command triggers
// an async operation like re-reading a diff.
func (m *Model) executeCommand() (bool, tea.Cmd) {
	cmd := strings.TrimSpace(m.commandBuffer)
	if cmd == "" {
		return false, nil
	}

	// :q / :quit — exit
	if cmd == "q" || cmd == "quit" {
		return true, nil
	}

	// :w / :write — refresh the diff view
	if cmd == "w" || cmd == "write" {
		if m.selectedPath == "" {
			return false, nil
		}
		var refreshCmd tea.Cmd
		if m.pipedDiff != "" {
			refreshCmd = func() tea.Msg {
				return vcs.DiffMsg{Content: m.vcs.ExtractFileDiff(m.pipedDiff, m.selectedPath)}
			}
		} else {
			refreshCmd = m.vcs.DiffCmd(m.targetBranch, m.selectedPath)
		}
		return false, refreshCmd
	}

	// :help / :h — open the full help buffer
	if cmd == "help" || cmd == "h" {
		m.enterHelpMode()
		return false, nil
	}

	// :noh / :nohlsearch — clear visual selection
	if cmd == "noh" || cmd == "nohlsearch" {
		m.visualMode = false
		m.visualStart = 0
		return false, nil
	}

	// :colorscheme <name> — switch theme
	if strings.HasPrefix(cmd, "colorscheme ") {
		name := strings.TrimSpace(strings.TrimPrefix(cmd, "colorscheme "))
		if name == "" {
			return false, nil
		}
		cfg := m.treeDelegate.Config
		cfg.UI.Theme = name
		m.treeDelegate.Config = cfg
		ReinitStyles(cfg)
		return false, nil
	}

	// :theme <name> — alias for :colorscheme
	if strings.HasPrefix(cmd, "theme ") {
		name := strings.TrimSpace(strings.TrimPrefix(cmd, "theme "))
		if name == "" {
			return false, nil
		}
		cfg := m.treeDelegate.Config
		cfg.UI.Theme = name
		m.treeDelegate.Config = cfg
		ReinitStyles(cfg)
		return false, nil
	}

	// :set <option> — live config changes
	if strings.HasPrefix(cmd, "set ") {
		m.executeSet(strings.TrimSpace(strings.TrimPrefix(cmd, "set ")))
		return false, nil
	}

	// :$ — jump to last diff line
	if cmd == "$" {
		m.diffCursor = m.snapCursor(m.diffLineCount()-1, -1)
		m.centerDiffCursor()
		return false, nil
	}

	// :<number> — jump to line N in the diff
	if num, err := strconv.Atoi(cmd); err == nil && num > 0 {
		m.diffCursor = m.snapCursor(num-1, 1)
		m.centerDiffCursor()
		return false, nil
	}

	return false, nil
}

// executeSet handles :set <option> commands for live configuration changes.
func (m *Model) executeSet(opt string) {
	switch opt {
	case "number", "nu":
		m.treeDelegate.Config.UI.LineNumbers = "absolute"
	case "nonumber", "nonu":
		m.treeDelegate.Config.UI.LineNumbers = "hidden"
	case "relativenumber", "rnu":
		m.treeDelegate.Config.UI.LineNumbers = "relative"
	case "norelativenumber", "nornu":
		m.treeDelegate.Config.UI.LineNumbers = "absolute"
	case "hybrid":
		m.treeDelegate.Config.UI.LineNumbers = "hybrid"
	case "relative":
		m.treeDelegate.Config.UI.LineNumbers = "relative"
	case "absolute":
		m.treeDelegate.Config.UI.LineNumbers = "absolute"
	case "hidden":
		m.treeDelegate.Config.UI.LineNumbers = "hidden"
	case "split":
		m.treeDelegate.Config.UI.DiffMode = "split"
		m.applyDiffModeChange(true)
	case "unified":
		m.treeDelegate.Config.UI.DiffMode = "unified"
		m.applyDiffModeChange(false)
	}
}

// commandNames is the set of completable command names (without the leading colon).
var commandNames = []string{"colorscheme", "theme", "q", "quit", "w", "write", "help", "h", "noh", "nohlsearch", "set"}

// setOptionNames is the set of completable :set option values.
var setOptionNames = []string{
	"number", "nonumber", "nu", "nonu",
	"relativenumber", "norelativenumber", "rnu", "nornu",
	"hybrid", "relative", "absolute", "hidden",
	"split", "unified",
}

// tabComplete handles vim-style tab cycling through matching completions.
// Supports command names (e.g. :col → :colorscheme) and theme values
// (e.g. :colorscheme gr → :colorscheme gruvbox). Repeated Tab cycles through
// all matches; typing resets the cycle.
func (m *Model) tabComplete() {
	// Detect which prefix we're completing under
	var prefix, partial string
	switch {
	case strings.HasPrefix(m.commandBuffer, "colorscheme "):
		prefix = "colorscheme "
		partial = strings.TrimPrefix(m.commandBuffer, "colorscheme ")
		m.cycleThemeNames(prefix, partial)
		return
	case strings.HasPrefix(m.commandBuffer, "theme "):
		prefix = "theme "
		partial = strings.TrimPrefix(m.commandBuffer, "theme ")
		m.cycleThemeNames(prefix, partial)
		return
	case strings.HasPrefix(m.commandBuffer, "set "):
		prefix = "set "
		partial = strings.TrimPrefix(m.commandBuffer, "set ")
		m.cycleSetOptions(prefix, partial)
		return
	default:
		// Completing command name itself (e.g. :col → :colorscheme)
		m.cycleCommandNames()
		return
	}
}

// cycleThemeNames cycles through matching theme names for tab completion.
func (m *Model) cycleThemeNames(prefix, partial string) {

	names := ThemeNames()

	// Collect all matching names
	var matches []string
	for _, name := range names {
		if strings.HasPrefix(name, partial) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return
	}

	// If the partial changed (user typed), reset cycle
	if partial != m.tabCyclePartial {
		m.tabCycleIndex = 0
		m.tabCyclePartial = partial
	}

	// Cycle to next match
	idx := m.tabCycleIndex % len(matches)
	m.commandBuffer = prefix + matches[idx]
	m.tabCycleIndex = idx + 1
}

// cycleCommandNames cycles through matching command names (e.g. :col → :colorscheme).
func (m *Model) cycleCommandNames() {
	partial := m.commandBuffer

	var matches []string
	for _, name := range commandNames {
		if strings.HasPrefix(name, partial) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return
	}

	if partial != m.tabCyclePartial {
		m.tabCycleIndex = 0
		m.tabCyclePartial = partial
	}

	idx := m.tabCycleIndex % len(matches)
	m.commandBuffer = matches[idx]
	m.tabCycleIndex = idx + 1
}

// cycleSetOptions cycles through matching :set option names.
func (m *Model) cycleSetOptions(prefix, partial string) {
	var matches []string
	for _, name := range setOptionNames {
		if strings.HasPrefix(name, partial) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return
	}

	if partial != m.tabCyclePartial {
		m.tabCycleIndex = 0
		m.tabCyclePartial = partial
	}

	idx := m.tabCycleIndex % len(matches)
	m.commandBuffer = prefix + matches[idx]
	m.tabCycleIndex = idx + 1
}

// resetTabCycle resets tab-completion state (called when user edits the buffer).
func (m *Model) resetTabCycle() {
	m.tabCycleIndex = 0
	m.tabCyclePartial = ""
}
