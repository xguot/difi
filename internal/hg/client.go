package hg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/patch"
)

var ansiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
var hunkHeaderRe = regexp.MustCompile(`^.*?@@ \-\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func getHgRoot() string {
	cmd := exec.Command("hg", "root")
	cmd.Env = append(os.Environ(), "HGRCPATH="+os.DevNull)
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
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
		out, err := hgCmd("diff", "--git", "--rev", target, "--", path).CombinedOutput()
		if err != nil {
			return DiffMsg{Err: fmt.Errorf("hg diff: %w: %s", err, strings.TrimSpace(string(out)))}
		}

		content := string(out)
		if content == "" {
			unknown, err := hgCmd("status", "--unknown", "--no-status", "-0", "--", path).Output()
			if err != nil {
				return DiffMsg{Err: fmt.Errorf("hg status: %w", err)}
			}
			if len(unknown) > 0 {
				filePath := filepath.Join(getHgRoot(), filepath.FromSlash(path))
				data, err := os.ReadFile(filePath)
				if err != nil {
					return DiffMsg{Err: err}
				}
				content = patch.Addition(path, data)
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

func DiffStats(target string) (added int, deleted int, err error) {
	byFile, err := DiffStatsByFile(target)
	if err != nil {
		return 0, 0, err
	}
	for _, counts := range byFile {
		added += counts[0]
		deleted += counts[1]
	}
	return added, deleted, nil
}

func DiffStatsByFile(target string) (map[string][2]int, error) {
	out, err := hgCmd("diff", "--git", "--rev", target).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("hg diff stats: %w: %s", err, strings.TrimSpace(string(out)))
	}
	result := make(map[string][2]int)
	for _, file := range patch.Parse(string(out)) {
		result[file.Path] = [2]int{file.Added, file.Deleted}
	}
	return result, nil
}

func CalculateFileLine(diffLines []string, visualLineIndex int) int {
	if len(diffLines) == 0 {
		return 1
	}

	if visualLineIndex < 0 {
		visualLineIndex = 0
	}
	if visualLineIndex >= len(diffLines) {
		visualLineIndex = len(diffLines) - 1
	}

	currentLineNo := 1
	mappedLineNo := 1
	inHunk := false

	for i := 0; i <= visualLineIndex; i++ {
		cleanLine := stripAnsi(diffLines[i])
		cleanLine = strings.TrimRight(cleanLine, "\r")
		matches := hunkHeaderRe.FindStringSubmatch(cleanLine)

		if len(matches) > 1 {
			startLine, _ := strconv.Atoi(matches[1])
			if startLine < 1 {
				startLine = 1
			}
			currentLineNo = startLine
			mappedLineNo = currentLineNo
			inHunk = true
			continue
		}

		if !inHunk {
			continue
		}

		switch {
		case strings.HasPrefix(cleanLine, " "), strings.HasPrefix(cleanLine, "+"):
			mappedLineNo = currentLineNo
			currentLineNo++
		case strings.HasPrefix(cleanLine, "-"):
			mappedLineNo = currentLineNo
		}
	}

	if mappedLineNo < 1 {
		return 1
	}

	return mappedLineNo
}

func stripAnsi(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}

type DiffMsg struct {
	Content string
	Err     error
}
type EditorFinishedMsg struct{ Err error }

func ParseFilesFromDiff(diffText string) []string {
	return patch.Paths(diffText)
}

func ExtractFileDiff(diffText, targetPath string) string {
	return patch.Extract(diffText, targetPath)
}
