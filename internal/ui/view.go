package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/xguot/difi/internal/tree"
	"github.com/xguot/difi/internal/vcs"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	topBar := m.renderTopBar()

	var bottomBar string
	if m.commandMode {
		bottomBar = m.renderCmdLine()
	} else if m.showHelp {
		bottomBar = m.renderHelpDrawer()
	} else {
		bottomBar = m.renderStatusBar()
	}

	contentHeight := m.height - lipgloss.Height(topBar) - lipgloss.Height(bottomBar)
	if contentHeight < 0 {
		contentHeight = 0
	}

	var mainContent string
	if len(m.fileList.Items()) == 0 {
		mainContent = m.renderEmptyState(m.width, contentHeight, "No changes found against "+m.targetBranch)
	} else {
		treeStyle := S.PaneStyle
		if m.focus == FocusTree {
			treeStyle = S.FocusedPaneStyle
		}

		treeView := treeStyle.Copy().
			Width(m.fileList.Width()).
			Height(contentHeight).
			MaxHeight(contentHeight).
			Render(m.fileList.View())

		var rightPaneView string
		selectedItem, ok := m.fileList.SelectedItem().(tree.TreeItem)

		if ok && selectedItem.IsDir {
			rightPaneView = m.renderEmptyState(m.diffViewport.Width, contentHeight, "Directory: "+selectedItem.Name)
		} else {
			var renderedDiff strings.Builder

			viewportHeight := contentHeight
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
				mode := m.treeDelegate.Config.UI.LineNumbers
				if mode == "" {
					mode = "hybrid"
				}

				if mode != "hidden" {
					if isCursor && mode == "hybrid" {
						realLine := m.vcs.CalculateFileLine(m.diffLines, m.diffCursor)
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
					lineNumRendered = S.LineNumberStyle.Render(numStr)
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
						line = S.CursorAddStyle.Copy().Width(maxLineWidth).Render(fullStr)
					} else if isDel {
						line = S.CursorDelStyle.Copy().Width(maxLineWidth).Render(fullStr)
					} else {
						line = S.CursorNormalStyle.Copy().Width(maxLineWidth).Render(fullStr)
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
							gutter = S.DiffCtxGutter.Render(gutterStr)
						}
					} else {
						if i < len(m.diffHighlighted) {
							hlCode = m.diffHighlighted[i]
							hlCode = bgAnsiRe.ReplaceAllString(hlCode, "")
						}

						if isAdd {
							gutter = S.DiffAddGutter.Render(gutterStr)
						} else if isDel {
							gutter = S.DiffDelGutter.Render(gutterStr)
						} else {
							gutter = S.DiffCtxGutter.Render(gutterStr)
						}
					}

					hlCode = ansi.Truncate(hlCode, maxLineWidth-4, "")
					line = gutter + hlCode
				}

				renderedDiff.WriteString(lineNumRendered + line + "\n")
			}

			diffContentStr := "\n" + strings.TrimRight(renderedDiff.String(), "\n")

			rightPaneView = S.DiffStyle.Copy().
				Width(m.diffViewport.Width).
				Height(contentHeight).
				MaxHeight(contentHeight).
				Render(diffContentStr)
		}

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treeView, rightPaneView)
	}

	return lipgloss.JoinVertical(lipgloss.Top, topBar, mainContent, bottomBar)
}

func (m Model) renderTopBar() string {
	vcsType := "git"
	if _, isHg := m.vcs.(vcs.HgVCS); isHg {
		vcsType = "hg"
	}

	repoStats := ""
	if m.statsAdded > 0 || m.statsDeleted > 0 {
		repoStats = fmt.Sprintf(" +%d -%d", m.statsAdded, m.statsDeleted)
	}

	info := fmt.Sprintf(" %s:%s  %s ➜ %s%s", m.repoName, vcsType, m.currentBranch, m.targetBranch, repoStats)
	leftSide := S.TopInfoStyle.Render(info)

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
			added := S.TopStatsAddedStyle.Render(fmt.Sprintf("+%d", statsAdded))
			deleted := S.TopStatsDeletedStyle.Render(fmt.Sprintf("-%d", statsDeleted))
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

	return S.TopBarStyle.Width(m.width).Render(finalBar)
}

// renderStatusBar renders the default bottom status line with shortcut hints.
func (m Model) renderStatusBar() string {
	t := ActiveTheme()
	shortcutsStyle := S.StatusKeyStyle.Copy().Background(t.StatusBarBg)
	shortcuts := shortcutsStyle.Render("? Help  q Quit  Tab Switch  V Visual  f Flat  : Cmd")

	availWidth := m.width - lipgloss.Width(shortcuts)
	if availWidth < 0 {
		availWidth = 0
	}

	paddingStyle := lipgloss.NewStyle().Background(t.StatusBarBg)
	padding := paddingStyle.Render(strings.Repeat(" ", availWidth))

	return lipgloss.JoinHorizontal(lipgloss.Top, shortcuts, padding)
}

// renderCmdLine renders a vim-style command-line at the bottom.
func (m Model) renderCmdLine() string {
	prompt := S.CmdLinePrompt.Render(":")
	text := S.CmdLineStyle.Render(m.commandBuffer)

	// Blinking cursor
	cursor := S.CmdLineCursor.Render(" ")

	line := prompt + text + cursor

	// Pad to full width
	lineWidth := lipgloss.Width(line)
	if lineWidth < m.width {
		t := ActiveTheme()
		padStyle := lipgloss.NewStyle().Background(t.CmdLineBg)
		line += padStyle.Render(strings.Repeat(" ", m.width-lineWidth))
	}

	return S.CmdLineStyle.Copy().Width(m.width).Render(line)
}

func (m Model) renderHelpDrawer() string {
	col1 := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("↑/k   Move Up"),
		S.HelpTextStyle.Render("↓/j   Move Down"),
	)
	col2 := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("←/h   Left Panel"),
		S.HelpTextStyle.Render("→/l   Right Panel"),
	)
	col3 := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("C-d/u Page Dn/Up"),
		S.HelpTextStyle.Render("zz/zt Scroll View"),
	)
	col4 := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("H/M/L Move Cursor"),
		S.HelpTextStyle.Render("e     Edit File"),
	)
	col5 := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("V     Visual Mode"),
		S.HelpTextStyle.Render("f     Flat Mode"),
		S.HelpTextStyle.Render(":     Command"),
		S.HelpTextStyle.Render("esc   Cancel Visual"),
	)

	return S.HelpDrawerStyle.Copy().
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
	logo := S.EmptyLogoStyle.Render("difi")
	desc := S.EmptyDescStyle.Render("A calm, focused way to review Git & Mercurial diffs.")
	status := S.EmptyStatusStyle.Render(statusMsg)

	t := ActiveTheme()
	usageHeader := S.EmptyHeaderStyle.Render("Usage Patterns")
	cmd1 := lipgloss.NewStyle().Foreground(t.Fg).Render("difi")
	desc1 := S.EmptyCodeStyle.Render("Auto-detect VCS, diff against main/tip")
	cmd2 := lipgloss.NewStyle().Foreground(t.Fg).Render("difi --vcs git")
	desc2 := S.EmptyCodeStyle.Render("Force Git mode")
	cmd3 := lipgloss.NewStyle().Foreground(t.Fg).Render("difi --vcs hg")
	desc3 := S.EmptyCodeStyle.Render("Force Mercurial mode")

	usageBlock := lipgloss.JoinVertical(lipgloss.Left,
		usageHeader,
		lipgloss.JoinHorizontal(lipgloss.Left, cmd1, "    ", desc1),
		lipgloss.JoinHorizontal(lipgloss.Left, cmd2, "    ", desc2),
		lipgloss.JoinHorizontal(lipgloss.Left, cmd3, "    ", desc3),
	)

	navHeader := S.EmptyHeaderStyle.Render("Navigation")
	key1 := lipgloss.NewStyle().Foreground(t.Fg).Render("Tab")
	key2 := lipgloss.NewStyle().Foreground(t.Fg).Render("j/k")
	keyDesc1 := S.EmptyCodeStyle.Render("Switch panels")
	keyDesc2 := S.EmptyCodeStyle.Render("Move cursor")

	navBlock := lipgloss.JoinVertical(lipgloss.Left,
		navHeader,
		lipgloss.JoinHorizontal(lipgloss.Left, key1, "    ", keyDesc1),
		lipgloss.JoinHorizontal(lipgloss.Left, key2, "    ", keyDesc2),
	)

	nvimHeader := S.EmptyHeaderStyle.Render("Neovim Integration")
	nvim1 := lipgloss.NewStyle().Foreground(t.Fg).Render("xguot/difi.nvim")
	nvimDesc1 := S.EmptyCodeStyle.Render("Install plugin")
	nvim2 := lipgloss.NewStyle().Foreground(t.Fg).Render("Press 'e'")
	nvimDesc2 := S.EmptyCodeStyle.Render("Edit with context")

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
