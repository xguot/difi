package ui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/vcs"
)

type diffResultMsg struct {
	Request            uint64
	Content            string
	Lines, Highlighted []string
	FileLines          []int
	Added, Deleted     int
	Notice             string
	Err                error
}

// Capture the request and theme before starting a command. Commands run outside
// Update, so slow Git output and syntax highlighting cannot block navigation.
func (m Model) loadDiffCmd() tea.Cmd {
	path, request := m.selectedPath, m.diffRequest
	theme := m.treeDelegate.Config.UI.Theme
	chromaStyle := GetTheme(theme).ChromaStyle
	return func() tea.Msg {
		var diff vcs.DiffMsg
		if m.pipedDiff != "" {
			diff.Content = m.vcs.ExtractFileDiff(m.pipedDiff, path)
		} else {
			msg := m.vcs.DiffCmd(m.targetBranch, path)()
			var ok bool
			diff, ok = msg.(vcs.DiffMsg)
			if !ok {
				diff.Err = fmt.Errorf("unexpected diff response %T", msg)
			}
		}
		if diff.Err != nil {
			return diffResultMsg{Request: request, Err: diff.Err}
		}
		result := prepareDiff(diff.Content, path, theme, chromaStyle)
		result.Request = request
		return result
	}
}

func (m *Model) requestDiff() tea.Cmd {
	m.diffRequest++
	m.diffLoading = m.selectedPath != ""
	m.diffLines, m.diffHighlighted, m.diffFileLines = nil, nil, nil
	m.currentFileAdded, m.currentFileDeleted = 0, 0
	m.diffNotice = ""
	m.diffError = ""
	if m.selectedPath == "" {
		return nil
	}
	return m.loadDiffCmd()
}

func prepareDiff(content, path, theme, chromaStyle string) diffResultMsg {
	fullLines := strings.Split(content, "\n")
	var cleanLines, hlLines []string
	var added, deleted int
	foundHunk := false

	ext := filepath.Ext(path)
	if len(ext) > 0 {
		ext = ext[1:]
	} else {
		ext = "txt"
	}

	isGitTheme := theme == "git"

	cache := make(map[string]string)
	for _, line := range fullLines {
		cleanLine := stripAnsi(line)

		if strings.HasPrefix(cleanLine, "@@") {
			foundHunk = true
		}

		if !foundHunk {
			continue
		}

		cleanLines = append(cleanLines, line)

		isAdd := strings.HasPrefix(cleanLine, "+")
		isDel := strings.HasPrefix(cleanLine, "-")

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
		} else if cached, ok := cache[codeContent]; ok {
			hlLines = append(hlLines, cached)
		} else {
			var buf strings.Builder
			err := quick.Highlight(&buf, codeContent, ext, "terminal16m", chromaStyle)
			if err == nil && buf.String() != "" {
				hlLines = append(hlLines, strings.TrimSuffix(buf.String(), "\n"))
			} else {
				hlLines = append(hlLines, codeContent)
			}
			cache[codeContent] = hlLines[len(hlLines)-1]
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

	result := diffResultMsg{Content: content, Lines: cleanLines, Highlighted: hlLines, Added: added, Deleted: deleted}
	result.FileLines = fileLineNumbers(cleanLines)
	if !foundHunk && strings.TrimSpace(content) != "" {
		result.Notice = "File metadata changed"
		if strings.Contains(content, "Binary files ") || strings.Contains(content, "GIT binary patch") {
			result.Notice = "Binary file changed"
		}
	}
	return result
}

var diffHunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func fileLineNumbers(lines []string) []int {
	numbers := make([]int, len(lines))
	current, mapped := 1, 1
	for i, raw := range lines {
		line := stripAnsi(raw)
		if match := diffHunkRe.FindStringSubmatch(line); len(match) > 1 {
			current, _ = strconv.Atoi(match[1])
			if current < 1 {
				current = 1
			}
			mapped = current
		} else if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "+") {
			mapped = current
			current++
		} else if strings.HasPrefix(line, "-") {
			mapped = current
		}
		numbers[i] = mapped
	}
	return numbers
}
