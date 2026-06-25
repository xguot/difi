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
		bottomBar = ""
	}

	contentHeight := m.height - lipgloss.Height(topBar) - lipgloss.Height(bottomBar)
	if contentHeight < 0 {
		contentHeight = 0
	}

	var mainContent string

	// Help buffer takes priority over everything
	if m.helpMode {
		treeView := S.PaneStyle.
			Width(m.fileList.Width()).
			Height(contentHeight).
			MaxHeight(contentHeight).
			Render(m.fileList.View())
		rightPaneView := m.renderHelpBuffer(contentHeight)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treeView, rightPaneView)
	} else if len(m.fileList.Items()) == 0 {
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
	t := ActiveTheme()
	pad := lipgloss.NewStyle().Width(3).Render("")

	header := lipgloss.NewStyle().
		Foreground(t.Fg).
		Bold(true).
		Render(" difi help")

	div := lipgloss.NewStyle().
		Foreground(t.DimFg).
		Render("│")

	nav := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("↑/k  ↓/j     Move up / down"),
		S.HelpTextStyle.Render("←/h  →/l     Focus tree / diff"),
		S.HelpTextStyle.Render("Tab          Toggle focus"),
		S.HelpTextStyle.Render("gg  G        First / last line"),
		S.HelpTextStyle.Render("C-d  C-u     Page down / up"),
		S.HelpTextStyle.Render("H  M  L      Cursor top / mid / bot"),
		S.HelpTextStyle.Render("zz  zt  zb   Center / top / bottom scroll"),
	)

	edit := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("e / Enter    Edit file at cursor"),
		S.HelpTextStyle.Render("V            Visual selection mode"),
		S.HelpTextStyle.Render("f            Toggle flat tree mode"),
		S.HelpTextStyle.Render("esc          Cancel visual mode"),
	)

	cmds := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render(":colorscheme Switch theme"),
		S.HelpTextStyle.Render(":set <opt>   Change setting"),
		S.HelpTextStyle.Render(":w           Refresh diff"),
		S.HelpTextStyle.Render(":noh         Clear highlight"),
		S.HelpTextStyle.Render(":<num>  :$   Jump to line"),
		S.HelpTextStyle.Render(":help :h     This help"),
	)

	quit := lipgloss.JoinVertical(lipgloss.Left,
		S.HelpTextStyle.Render("q  ZZ        Quit"),
		S.HelpTextStyle.Render(":q :quit     Quit (cmd)"),
		S.HelpTextStyle.Render("Ctrl+C       Force quit"),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		nav, pad,
		div, pad,
		edit, pad,
		div, pad,
		cmds, pad,
		div, pad,
		quit,
	)

	return S.HelpDrawerStyle.Copy().
		Width(m.width).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
}

// renderHelpBuffer renders the embedded help.txt in the right pane with
// basic vimdoc-style highlighting: section dividers, tags, and subsections.
func (m Model) renderHelpBuffer(contentHeight int) string {
	if len(m.helpLines) == 0 {
		return ""
	}

	t := ActiveTheme()

	start := m.diffViewport.YOffset
	end := start + contentHeight
	if end > len(m.helpLines) {
		end = len(m.helpLines)
	}

	maxLineWidth := m.diffViewport.Width - 2
	if maxLineWidth < 1 {
		maxLineWidth = 1
	}

	divStyle := lipgloss.NewStyle().Foreground(t.DimFg).Bold(true)
	tagStyle := lipgloss.NewStyle().Foreground(t.FocusedBorder)
	subStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(t.Fg)

	var rendered strings.Builder
	for i := start; i < end; i++ {
		line := m.helpLines[i]

		// Section divider: =======================================
		if strings.HasPrefix(line, "===") {
			rendered.WriteString(divStyle.Render(ansi.Truncate(line, maxLineWidth, "")) + "\n")
			continue
		}

		// Subsection header: > Text
		if strings.HasPrefix(line, "> ") {
			rest := ansi.Truncate(line, maxLineWidth, "")
			rendered.WriteString(subStyle.Render(rest) + "\n")
			continue
		}

		// Inline tags: *tag-name*
		var styledLine strings.Builder
		remaining := line
		for {
			star := strings.Index(remaining, "*")
			if star == -1 {
				styledLine.WriteString(bodyStyle.Render(ansi.Truncate(remaining, maxLineWidth-lipgloss.Width(styledLine.String()), "")))
				break
			}
			// Text before the tag
			styledLine.WriteString(bodyStyle.Render(remaining[:star]))
			remaining = remaining[star+1:]
			endStar := strings.Index(remaining, "*")
			if endStar == -1 {
				styledLine.WriteString(bodyStyle.Render("*" + ansi.Truncate(remaining, maxLineWidth-lipgloss.Width(styledLine.String()), "")))
				break
			}
			tag := remaining[:endStar]
			styledLine.WriteString(tagStyle.Render("*" + tag + "*"))
			remaining = remaining[endStar+1:]
		}
		rendered.WriteString(styledLine.String() + "\n")
	}

	return S.DiffStyle.Copy().
		Width(m.diffViewport.Width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render("\n" + strings.TrimRight(rendered.String(), "\n"))
}

func (m Model) renderEmptyState(w, h int, statusMsg string) string {
	t := ActiveTheme()

	logo := S.EmptyLogoStyle.Render("difi")
	gap := lipgloss.NewStyle().Height(1).Render("")

	versionLine := S.EmptyDescStyle.Render("version " + m.version)
	creditLine := S.EmptyStatusStyle.Render("by Xiyuan Guo et al.")
	ossLine := S.EmptyStatusStyle.Render("difi is open source and freely distributable")

	cmdStyle := S.EmptyCodeStyle
	keyStyle := lipgloss.NewStyle().Foreground(t.Fg)

	cmdHelp := cmdStyle.Render("type  ") + keyStyle.Render(":help") + cmdStyle.Render("       for help")
	cmdQuit := cmdStyle.Render("type  ") + keyStyle.Render(":q") + cmdStyle.Render("          to exit")
	cmdTheme := cmdStyle.Render("type  ") + keyStyle.Render(":colorscheme") + cmdStyle.Render(" to change theme")

	var blocks []string
	blocks = append(blocks, logo, gap, versionLine, creditLine, ossLine)

	if statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
		statusLine := statusStyle.Render(statusMsg)
		blocks = append(blocks, gap, statusLine)
	}

	blocks = append(blocks, gap, cmdHelp, cmdQuit, cmdTheme)

	content := lipgloss.JoinVertical(lipgloss.Center, blocks...)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
