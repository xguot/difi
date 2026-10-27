package ui

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xguot/difi/internal/tree"
	"github.com/xguot/difi/internal/vcs"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.promptMode == promptNone && !m.commandMode {
		// Bracketed paste is text, not a sequence of normal-mode commands.
		if key.Paste {
			return m, nil
		}
		if key.Type == tea.KeyRunes && len(key.Runes) > 1 {
			var commands []tea.Cmd
			for _, r := range key.Runes {
				quitting := m.promptMode == promptNone && !m.commandMode && !m.helpMode && (r == 'q' || (r == 'Z' && m.pendingZZ))
				part := key
				part.Runes = []rune{r}
				next, cmd := m.Update(part)
				m = next.(Model)
				commands = append(commands, cmd)
				if quitting {
					break
				}
			}
			return m, tea.Batch(commands...)
		}
	}
	var cmd tea.Cmd
	var cmds []tea.Cmd

	keyHandled := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case StatsMsg:
		if msg.Request != m.refreshRequest {
			return m, nil
		}
		if msg.Err != nil {
			m.errorText = msg.Err.Error()
			m.updateSizes()
			return m, nil
		}
		m.statsAdded = msg.Added
		m.statsDeleted = msg.Deleted
		if msg.ByFile != nil {
			m.fileStats = msg.ByFile
		}

	case sessionMsg:
		if msg.Request != m.refreshRequest {
			return m, nil
		}
		if msg.Err != nil {
			m.diffLoading = false
			m.errorText = msg.Err.Error()
			m.updateSizes()
			return m, nil
		}
		m.errorText = ""
		m.statsAdded, m.statsDeleted, m.fileStats = msg.Stats.Added, msg.Stats.Deleted, msg.Stats.ByFile
		m.replaceFiles(msg.Files)
		return m, m.requestDiff()

	case tea.KeyMsg:
		if m.promptMode != promptNone {
			return m, m.updatePrompt(msg)
		}
		// ── Command-line mode ──────────────────────────────────
		if m.commandMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.commandMode = false
				m.updateSizes()
				m.commandBuffer = ""
				m.resetTabCycle()
				return m, nil

			case "enter":
				quit, cmd := m.executeCommand()
				m.commandMode = false
				m.updateSizes()
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
					_, size := utf8.DecodeLastRuneInString(m.commandBuffer)
					m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-size]
				}
				m.resetTabCycle()
				return m, nil

			case "tab":
				m.tabComplete()
				return m, nil

			default:
				if msg.Type == tea.KeyRunes {
					for _, r := range msg.Runes {
						if unicode.IsPrint(r) {
							m.commandBuffer += string(r)
						}
					}
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
			m.pendingHunk = ""
			m.commandMode = true
			m.updateSizes()
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
			m.pendingHunk = ""
			m.showHelp = !m.showHelp
			m.updateSizes()
			return m, nil
		}

		if msg.String() == "/" {
			return m, m.beginPrompt()
		}
		if msg.String() == "esc" && m.focus == FocusTree && m.fileFilter != "" {
			m.fileFilter = ""
			old := m.selectedPath
			m.rebuildFileList(old)
			m.updateSizes()
			if old != m.selectedPath {
				return m, m.requestDiff()
			}
			return m, nil
		}
		pending := m.pendingHunk
		m.pendingHunk = ""
		if (pending != "" && msg.String() == "c") || msg.String() == "]c" || msg.String() == "[c" {
			if pending != "" {
				m.inputBuffer = strconv.Itoa(m.pendingHunkCount)
			}
			direction := 1
			if pending == "[" || msg.String() == "[c" {
				direction = -1
			}
			m.jumpHunk(direction)
			return m, nil
		}
		if msg.String() == "[" || msg.String() == "]" {
			m.pendingHunk = msg.String()
			m.pendingHunkCount = m.consumeCount()
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
			m.searchQuery = ""
			m.rebuildSearch()
			m.updateSizes()
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

		case "n", "N":
			if m.focus == FocusDiff {
				direction := 1
				if msg.String() == "N" {
					direction = -1
				}
				m.jumpSearch(direction)
			} else {
				m.inputBuffer = ""
			}

		case "}", "{":
			direction := 1
			if msg.String() == "{" {
				direction = -1
			}
			m.jumpHunk(direction)

		case "f":
			if m.focus == FocusTree {
				m.flatMode = !m.flatMode
				m.rebuildFileList(m.selectedPath)
				return m, nil
			}

		case "enter", "e":
			m.visualMode = false
			if m.focus == FocusTree && msg.String() == "enter" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					m.treeState.ToggleExpand(i.FullPath)
					m.fileList.SetItems(m.fileItems())
					return m, nil
				}
			}

			if m.selectedPath != "" {
				if i, ok := m.fileList.SelectedItem().(tree.TreeItem); ok && i.IsDir {
					return m, nil
				}

				line := 0
				if m.focus == FocusDiff {
					line = m.vcs.CalculateFileLine(m.diffLines, m.diffCursor)
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
			count := m.getRepeatCount()
			for i := 0; i < count; i++ {
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
			count := m.getRepeatCount()
			for i := 0; i < count; i++ {
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
				cmds = append(cmds, m.requestDiff())
			}
		}
	}

	switch msg := msg.(type) {
	case diffResultMsg:
		if msg.Request != m.diffRequest {
			return m, nil
		}
		m.diffLoading = false
		if msg.Err != nil {
			m.diffError = msg.Err.Error()
			return m, nil
		}
		m.diffError = ""
		m.diffContent = msg.Content
		m.diffLines, m.diffHighlighted, m.diffFileLines = msg.Lines, msg.Highlighted, msg.FileLines
		m.currentFileAdded, m.currentFileDeleted = msg.Added, msg.Deleted
		m.diffNotice = msg.Notice
		m.diffHunks = hunkStarts(m.diffLines)
		m.rebuildSearch()
		m.updateSizes()
		m.diffCursor = m.snapCursor(m.diffCursor, 1)
		m.handleScrolling()

	case vcs.EditorFinishedMsg:
		if msg.Err != nil {
			m.errorText = "Editor: " + msg.Err.Error()
			m.updateSizes()
			return m, nil
		}
		return m, m.refreshSession()

	}

	if m.promptMode != promptNone {
		m.promptInput, cmd = m.promptInput.Update(msg)
		cmds = append(cmds, cmd)
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
		return false, m.refreshSession()
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
		m.searchQuery = ""
		m.rebuildSearch()
		m.updateSizes()
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
		m.updateTreeFocus()
		return false, m.requestDiff()
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
		m.updateTreeFocus()
		return false, m.requestDiff()
	}

	// :set <option> — live config changes
	if strings.HasPrefix(cmd, "set ") {
		m.executeSet(strings.TrimSpace(strings.TrimPrefix(cmd, "set ")))
		return false, nil
	}

	// :$ — jump to last diff line
	if cmd == "$" {
		m.diffCursor = m.snapCursor(len(m.diffLines)-1, -1)
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
	}
}

// commandNames is the set of completable command names (without the leading colon).
var commandNames = []string{"colorscheme", "theme", "q", "quit", "w", "write", "help", "h", "noh", "nohlsearch", "set"}

// setOptionNames is the set of completable :set option values.
var setOptionNames = []string{
	"number", "nonumber", "nu", "nonu",
	"relativenumber", "norelativenumber", "rnu", "nornu",
	"hybrid", "relative", "absolute", "hidden",
}

// tabComplete handles vim-style tab cycling through matching completions.
// Supports command names (e.g. :col → :colorscheme) and theme values
// (e.g. :colorscheme gr → :colorscheme gruvbox). Repeated Tab cycles through
// all matches; typing resets the cycle.
func (m *Model) tabComplete() {
	if len(m.tabCycleMatches) == 0 {
		partial := m.commandBuffer
		names := commandNames
		prefix := ""
		switch {
		case strings.HasPrefix(partial, "colorscheme "):
			prefix, names = "colorscheme ", ThemeNames()
		case strings.HasPrefix(partial, "theme "):
			prefix, names = "theme ", ThemeNames()
		case strings.HasPrefix(partial, "set "):
			prefix, names = "set ", setOptionNames
		}
		partial = strings.TrimPrefix(partial, prefix)
		for _, name := range names {
			if strings.HasPrefix(name, partial) {
				m.tabCycleMatches = append(m.tabCycleMatches, name)
			}
		}
		m.tabCyclePrefix = prefix
	}
	if len(m.tabCycleMatches) == 0 {
		return
	}
	m.commandBuffer = m.tabCyclePrefix + m.tabCycleMatches[m.tabCycleIndex%len(m.tabCycleMatches)]
	m.tabCycleIndex++
}

func (m *Model) resetTabCycle() {
	m.tabCycleIndex = 0
	m.tabCycleMatches = nil
	m.tabCyclePrefix = ""
}
