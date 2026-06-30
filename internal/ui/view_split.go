package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/xguot/difi/internal/diffsplit"
)

// splitLineNumWidth matches LineNumberStyle: Width(4) + MarginRight(1).
const splitLineNumWidth = 5

// splitGutterWidth matches unified diff gutter: "+ │ " / "- │ " / "  │ ".
const splitGutterWidth = 4

type splitLayout struct {
	totalW     int
	leftColW   int
	rightColW  int
	leftCodeW  int
	rightCodeW int
}

func computeSplitLayout(totalW int, showLineNums bool) splitLayout {
	if totalW < 3 {
		totalW = 3
	}
	sepW := 1
	leftColW := (totalW - sepW) / 2
	rightColW := totalW - sepW - leftColW

	numW := 0
	if showLineNums {
		numW = splitLineNumWidth
	}

	metaW := numW + splitGutterWidth

	return splitLayout{
		totalW:     totalW,
		leftColW:   leftColW,
		rightColW:  rightColW,
		leftCodeW:  max1(leftColW - metaW),
		rightCodeW: max1(rightColW - metaW),
	}
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func splitGutterStr(kind diffsplit.LineKind, isCursor bool) string {
	sep := "│"
	if isCursor {
		sep = "┃"
	}
	switch kind {
	case diffsplit.KindAdd:
		return "+ " + sep + " "
	case diffsplit.KindDel:
		return "- " + sep + " "
	default:
		return "  " + sep + " "
	}
}

func (m Model) renderSplitGutter(kind diffsplit.LineKind, isGitTheme bool) string {
	s := splitGutterStr(kind, false)
	if isGitTheme {
		switch kind {
		case diffsplit.KindAdd:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(s)
		case diffsplit.KindDel:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(s)
		}
		return S.DiffCtxGutter.Render(s)
	}
	switch kind {
	case diffsplit.KindAdd:
		return S.DiffAddGutter.Render(s)
	case diffsplit.KindDel:
		return S.DiffDelGutter.Render(s)
	default:
		return S.DiffCtxGutter.Render(s)
	}
}

func splitCursorStyle(kind diffsplit.LineKind) lipgloss.Style {
	switch kind {
	case diffsplit.KindAdd:
		return S.CursorAddStyle
	case diffsplit.KindDel:
		return S.CursorDelStyle
	default:
		return S.CursorNormalStyle
	}
}

func padVisible(s string, width int) string {
	if width <= 0 {
		return ""
	}
	n := lipgloss.Width(s)
	if n > width {
		return ansi.Truncate(s, width, "")
	}
	if n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func expandTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "    ")
}

func stripHighlightBg(s string) string {
	return bgAnsiRe.ReplaceAllString(s, "")
}

// renderSplitCode renders the code portion matching unified view: syntax
// highlighting by default (git theme uses colored plain text for +/- lines).
func (m Model) renderSplitCode(side diffsplit.Side, codeW int, isGitTheme bool) string {
	content := expandTabs(side.Content)

	if isGitTheme && (side.Kind == diffsplit.KindAdd || side.Kind == diffsplit.KindDel) {
		truncated := padVisible(ansi.Truncate(content, codeW, ""), codeW)
		if side.Kind == diffsplit.KindAdd {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(truncated)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(truncated)
	}

	hl := side.Highlighted
	if hl == "" {
		hl = side.Content
	}
	hl = stripHighlightBg(expandTabs(hl))
	return padVisible(ansi.Truncate(hl, codeW, ""), codeW)
}

func (m Model) renderSplitDiff(contentHeight int) string {
	layout := computeSplitLayout(m.diffViewport.Width, m.treeDelegate.Config.UI.LineNumbers != "hidden")
	isGitTheme := m.treeDelegate.Config.UI.Theme == "git"
	showLineNums := m.treeDelegate.Config.UI.LineNumbers != "hidden"
	t := ActiveTheme()

	sepStyle := lipgloss.NewStyle().Foreground(t.Border)
	sepCursorStyle := lipgloss.NewStyle().Foreground(t.FocusedBorder).Bold(true)

	var rendered strings.Builder

	start := m.diffViewport.YOffset
	end := start + contentHeight
	if end > len(m.splitRows) {
		end = len(m.splitRows)
	}

	for i := start; i < end; i++ {
		row := m.splitRows[i]
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

		sep := sepStyle.Render("│")
		if isCursor {
			sep = sepCursorStyle.Render("│")
		}

		left := m.renderSplitCell(row.Left, layout.leftColW, layout.leftCodeW, showLineNums, isCursor, isGitTheme)
		right := m.renderSplitCell(row.Right, layout.rightColW, layout.rightCodeW, showLineNums, isCursor, isGitTheme)

		line := left + sep + right
		line = padVisible(line, layout.totalW)
		rendered.WriteString(line + "\n")
	}

	return rendered.String()
}

func (m Model) renderSplitCell(side diffsplit.Side, colW, codeW int, showLineNums, isCursor, isGitTheme bool) string {
	if colW <= 0 {
		return ""
	}
	if codeW < 0 {
		codeW = 0
	}

	lineNum := ""
	if showLineNums {
		numStr := ""
		if side.LineNo > 0 {
			numStr = fmt.Sprintf("%d", side.LineNo)
		}
		lineNum = S.LineNumberStyle.Render(numStr)
	}

	gutterKind := side.Kind
	if gutterKind == diffsplit.KindEmpty {
		gutterKind = diffsplit.KindContext
	}

	gutterAndCodeW := splitGutterWidth + codeW

	// Cursor row: gutter + plain code highlighted together (same as unified view).
	if isCursor && side.Kind != diffsplit.KindEmpty {
		content := padVisible(ansi.Truncate(expandTabs(side.Content), codeW, ""), codeW)
		combined := splitGutterStr(gutterKind, true) + content
		combined = padVisible(combined, gutterAndCodeW)
		combined = splitCursorStyle(gutterKind).Copy().Width(gutterAndCodeW).Render(combined)
		return padVisible(lineNum+combined, colW)
	}

	var codeCell string
	if side.Kind == diffsplit.KindEmpty {
		codeCell = strings.Repeat(" ", codeW)
	} else {
		codeCell = m.renderSplitCode(side, codeW, isGitTheme)
	}

	gutter := m.renderSplitGutter(gutterKind, isGitTheme)
	return padVisible(lineNum+gutter+codeCell, colW)
}
