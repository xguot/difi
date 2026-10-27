package git

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

// ansiRe matches ANSI escape sequences to strip or identify terminal styling.
var ansiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
var hunkHeaderRe = regexp.MustCompile(`^@@ \-\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func gitCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"--no-pager", "--literal-pathspecs"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Dir = repoRoot()
	return cmd
}

func repoRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
	out, err := gitCmd("diff", "--no-relative", "--name-only", "-z", targetBranch, "--").Output()
	if err != nil {
		return nil, err
	}

	untracked, err := gitCmd("ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var files []string

	allOutput := string(out) + string(untracked)
	for _, f := range strings.Split(allOutput, "\x00") {
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}

	return files, nil
}

func DiffCmd(targetBranch, path string) tea.Cmd {
	return func() tea.Msg {
		out, err := gitCmd("diff", "--no-relative", "--no-ext-diff", "--no-textconv", "--color=never", "--src-prefix=a/", "--dst-prefix=b/", targetBranch, "--", path).CombinedOutput()
		if err != nil {
			return DiffMsg{Err: fmt.Errorf("git diff: %w: %s", err, strings.TrimSpace(string(out)))}
		}

		content := string(out)
		if content == "" {
			// Only untracked files need a synthetic addition. A tracked file
			// restored during refresh must not appear as a full-file addition.
			if err := gitCmd("ls-files", "--error-unmatch", "--", path).Run(); err != nil {
				out, err = gitCmd("diff", "--no-ext-diff", "--no-textconv", "--color=never", "--no-index", "--", os.DevNull, path).CombinedOutput()
				if err != nil {
					if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
						return DiffMsg{Err: fmt.Errorf("git diff untracked file: %w: %s", err, strings.TrimSpace(string(out)))}
					}
				}
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
	c.Dir = repoRoot()
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.Env = append(os.Environ(), fmt.Sprintf("DIFI_TARGET=%s", targetBranch))

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func DiffStats(targetBranch string) (added int, deleted int, err error) {
	byFile, err := DiffStatsByFile(targetBranch)
	if err != nil {
		return 0, 0, fmt.Errorf("git diff stats error: %w", err)
	}

	for _, stats := range byFile {
		added += stats[0]
		deleted += stats[1]
	}
	return added, deleted, nil
}

func DiffStatsByFile(targetBranch string) (map[string][2]int, error) {
	cmd := gitCmd("diff", "--no-relative", "--numstat", "-z", targetBranch, "--")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff numstat error: %w", err)
	}

	result := make(map[string][2]int)
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		parts := strings.SplitN(entries[i], "\t", 3)
		if len(parts) != 3 {
			continue
		}
		var a, d int
		if parts[0] != "-" {
			a, _ = strconv.Atoi(parts[0])
		}
		if parts[1] != "-" {
			d, _ = strconv.Atoi(parts[1])
		}
		filePath := parts[2]
		if filePath == "" && i+2 < len(entries) {
			// In -z mode, renames encode old and new paths as separate fields.
			filePath = entries[i+2]
			i += 2
		}
		if filePath != "" {
			result[filepath.ToSlash(filePath)] = [2]int{a, d}
		}
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
