package git

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRe = regexp.MustCompile(`[\x1b\x9b][[\\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?\x07)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PRZcf-ntqry=><~]))`)
var hunkHeaderRe = regexp.MustCompile(`^.*?@@ \-\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func gitCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"--no-pager"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd
}

func GetCurrentBranch() string {
	out, err := gitCmd("rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "HEAD"
	}
	return strings.TrimSpace(string(out))
}

func GetRepoName() string {
	out, err := gitCmd("rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "Repo"
	}
	path := strings.TrimSpace(string(out))
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "Repo"
}

func ListChangedFiles(targetBranch string) ([]string, error) {
	out, err := gitCmd("diff", "--name-only", targetBranch).Output()
	if err != nil {
		return nil, err
	}

	untracked, err := gitCmd("ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var files []string

	allOutput := string(out) + "\n" + string(untracked)
	for _, line := range strings.Split(strings.TrimSpace(allOutput), "\n") {
		f := strings.TrimSpace(line)
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}

	return files, nil
}

func DiffCmd(targetBranch, path string) tea.Cmd {
	return func() tea.Msg {
		out, err := gitCmd("diff", "--color=always", targetBranch, "--", path).Output()
		if err != nil {
			return DiffMsg{Content: "Error fetching diff: " + err.Error()}
		}

		content := string(out)
		if content == "" {
			if _, err := os.Stat(path); err == nil {
				out, _ = exec.Command("git", "diff", "--color=always", "--no-index", "/dev/null", path).Output()
				content = string(out)
			}
		}

		return DiffMsg{Content: content}
	}
}

func OpenEditorCmd(path string, lineNumber int, targetBranch string, editor string) tea.Cmd {
	var args []string
	if lineNumber > 0 {
		args = append(args, fmt.Sprintf("+%d", lineNumber))
	}
	args = append(args, path)

	c := exec.Command(editor, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.Env = append(os.Environ(), fmt.Sprintf("DIFI_TARGET=%s", targetBranch))

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func DiffStats(targetBranch string) (added int, deleted int, err error) {
	cmd := gitCmd("diff", "--numstat", targetBranch)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git diff stats error: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		if parts[0] != "-" {
			if n, err := strconv.Atoi(parts[0]); err == nil {
				added += n
			}
		}
		if parts[1] != "-" {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				deleted += n
			}
		}
	}
	return added, deleted, nil
}

func DiffStatsByFile(targetBranch string) (map[string][2]int, error) {
	cmd := gitCmd("diff", "--numstat", targetBranch)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff numstat error: %w", err)
	}

	result := make(map[string][2]int)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		var a, d int
		if parts[0] != "-" {
			a, _ = strconv.Atoi(parts[0])
		}
		if parts[1] != "-" {
			d, _ = strconv.Atoi(parts[1])
		}
		filePath := strings.Join(parts[2:], " ")
		if idx := strings.LastIndex(filePath, " => "); idx != -1 {
			filePath = filePath[idx+4:]
		}
		result[filePath] = [2]int{a, d}
	}
	return result, nil
}

func CalculateFileLine(diffContent string, visualLineIndex int) int {
	lines := strings.Split(diffContent, "\n")
	if visualLineIndex >= len(lines) {
		return 0
	}

	currentLineNo := 0
	lastWasHunk := false
	inHeader := true

	for i := 0; i <= visualLineIndex; i++ {
		line := lines[i]
		matches := hunkHeaderRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			startLine, _ := strconv.Atoi(matches[1])
			currentLineNo = startLine
			lastWasHunk = true
			inHeader = false
			continue
		}

		lastWasHunk = false
		cleanLine := stripAnsi(line)

		if inHeader {
			continue
		}

		if strings.HasPrefix(cleanLine, " ") || strings.HasPrefix(cleanLine, "+") {
			currentLineNo++
		}
	}

	if currentLineNo == 0 {
		return 1
	}
	if lastWasHunk {
		return currentLineNo - 1
	}
	return currentLineNo - 1
}

func stripAnsi(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}

type DiffMsg struct{ Content string }
type EditorFinishedMsg struct{ Err error }

func ParseFilesFromDiff(diffText string) []string {
	var files []string
	seen := make(map[string]bool)
	lines := strings.Split(diffText, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git a/") {
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				file := strings.TrimPrefix(parts[0], "diff --git a/")
				if !seen[file] {
					seen[file] = true
					files = append(files, file)
				}
			}
		}
	}
	return files
}

func ExtractFileDiff(diffText, targetPath string) string {
	lines := strings.Split(diffText, "\n")
	var out []string
	inTarget := false
	targetHeader := fmt.Sprintf("diff --git a/%s b/%s", targetPath, targetPath)

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			inTarget = strings.HasPrefix(line, targetHeader)
		}
		if inTarget {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
