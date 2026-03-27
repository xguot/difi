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
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

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
