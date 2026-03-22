package hg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var hgRoot string
var ansiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
var hunkHeaderRe = regexp.MustCompile(`^.*?@@ \-\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func getHgRoot() string {
	if hgRoot != "" {
		return hgRoot
	}
	cmd := exec.Command("hg", "root")
	cmd.Env = append(os.Environ(), "HGRCPATH="+os.DevNull)
	out, err := cmd.Output()
	if err == nil {
		hgRoot = strings.TrimSpace(string(out))
	}
	return hgRoot
}

func hgCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("hg", args...)
	cmd.Env = append(os.Environ(), "HGRCPATH="+os.DevNull)
	if root := getHgRoot(); root != "" {
		cmd.Dir = root
	}
	return cmd
}

func GetCurrentBranch() string {
	out, err := hgCmd("branch").Output()
	if err != nil {
		return "default"
	}
	return strings.TrimSpace(string(out))
}

func GetRepoName() string {
	out, err := hgCmd("root").Output()
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

func ListChangedFiles(target string) ([]string, error) {
	// m: modified, a: added, r: removed, d: deleted
	out, err := hgCmd("status", "--rev", target, "-mard", "--no-status").Output()
	if err != nil {
		return nil, err
	}

	// u: unknown (untracked)
	untracked, err := hgCmd("status", "--unknown", "--no-status").Output()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var files []string

	all := string(out) + "\n" + string(untracked)
	for _, line := range strings.Split(strings.TrimSpace(all), "\n") {
		f := strings.TrimSpace(line)
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	return files, nil
}

func DiffCmd(target, path string) tea.Cmd {
	return func() tea.Msg {
		out, err := hgCmd("diff", "--change", target, path).Output()
		if err != nil {
			return DiffMsg{Content: "Error: " + err.Error()}
		}

		content := string(out)
		if content == "" {
			if _, err := os.Stat(path); err == nil {
				/* diff untracked file as full addition */
				out, _ = exec.Command("hg", "diff", "--git", "/dev/null", path).Output()
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
	if root := getHgRoot(); root != "" {
		c.Dir = root
	}

	c.Env = append(os.Environ(), fmt.Sprintf("DIFI_TARGET=%s", targetBranch))

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func DiffStats(targetBranch string) (added int, deleted int, err error) {
	var cmd *exec.Cmd
	if targetBranch == "tip" || targetBranch == "." || targetBranch == "" {
		cmd = hgCmd("diff", "--stat")
	} else {
		cmd = hgCmd("diff", "--rev", targetBranch, "--stat")
	}

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("hg diff stats error: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "changed") && (strings.Contains(line, "insertion") || strings.Contains(line, "deletion")) {
			reAdded := regexp.MustCompile(`(\d+) insertion[s]?\(\+\)`)
			if matches := reAdded.FindStringSubmatch(line); len(matches) > 1 {
				if n, err := strconv.Atoi(matches[1]); err == nil {
					added = n
				}
			}

			reDeleted := regexp.MustCompile(`(\d+) deletion[s]?\(\-\)`)
			if matches := reDeleted.FindStringSubmatch(line); len(matches) > 1 {
				if n, err := strconv.Atoi(matches[1]); err == nil {
					deleted = n
				}
			}
			break
		}
	}
	return added, deleted, nil
}

func DiffStatsByFile(targetBranch string) (map[string][2]int, error) {
	var cmd *exec.Cmd
	if targetBranch == "tip" || targetBranch == "." || targetBranch == "" {
		cmd = hgCmd("diff", "--stat")
	} else {
		cmd = hgCmd("diff", "--rev", targetBranch, "--stat")
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("hg diff stat error: %w", err)
	}

	result := make(map[string][2]int)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "changed") && (strings.Contains(line, "insertion") || strings.Contains(line, "deletion")) {
			continue
		}
		pipeIdx := strings.LastIndex(line, "|")
		if pipeIdx < 0 {
			continue
		}
		filePath := strings.TrimSpace(line[:pipeIdx])
		changesPart := strings.TrimSpace(line[pipeIdx+1:])
		var a, d int
		for _, ch := range changesPart {
			if ch == '+' {
				a++
			} else if ch == '-' {
				d++
			}
		}
		if filePath != "" {
			result[filePath] = [2]int{a, d}
		}
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
		if strings.HasPrefix(line, "diff -r ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				file := parts[len(parts)-1]
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

	for _, line := range lines {
		if strings.HasPrefix(line, "diff -r ") {
			parts := strings.Fields(line)
			inTarget = len(parts) > 0 && parts[len(parts)-1] == targetPath
		}
		if inTarget {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
