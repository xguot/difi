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
	"github.com/xguot/difi/internal/review"
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
	return ListChangedFilesWithOptions(targetBranch, review.Options{})
}

func diffArgs(target string, options review.Options, flags ...string) []string {
	args := append([]string{"diff", "--no-relative", "--no-ext-diff", "--no-textconv"}, flags...)
	if options.Scope == review.Staged {
		args = append(args, "--cached")
	}
	if options.Scope != review.Unstaged && !(options.Scope == review.Staged && target == "HEAD") {
		args = append(args, target)
	}
	return append(args, "--")
}

func ListChangedFilesWithOptions(target string, options review.Options) ([]string, error) {
	out, err := gitCmd(diffArgs(target, options, "--name-only", "-z")...).Output()
	if err != nil {
		return nil, err
	}

	var untracked []byte
	if options.IncludeUntracked() {
		untracked, err = gitCmd("ls-files", "--others", "--exclude-standard", "-z").Output()
		if err != nil {
			return nil, err
		}
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
	return DiffCmdWithOptions(targetBranch, path, review.Options{})
}

func DiffCmdWithOptions(targetBranch, path string, options review.Options) tea.Cmd {
	return func() tea.Msg {
		args := append(diffArgs(targetBranch, options, "--color=never", "--src-prefix=a/", "--dst-prefix=b/"), path)
		out, err := gitCmd(args...).CombinedOutput()
		if err != nil {
			return DiffMsg{Err: fmt.Errorf("git diff: %w: %s", err, strings.TrimSpace(string(out)))}
		}

		content := string(out)
		// Restricting a diff to the destination alone can hide a rename's
		// source and render the destination as an entirely new file.
		if strings.Contains(content, "\nnew file mode ") {
			if source, err := renameSource(targetBranch, path, options); err != nil {
				return DiffMsg{Err: err}
			} else if source != "" {
				args = append(diffArgs(targetBranch, options, "--color=never", "--src-prefix=a/", "--dst-prefix=b/"), source, path)
				out, err = gitCmd(args...).CombinedOutput()
				if err != nil {
					return DiffMsg{Err: fmt.Errorf("git rename diff: %w: %s", err, strings.TrimSpace(string(out)))}
				}
				content = string(out)
			}
		}
		if content == "" && options.IncludeUntracked() {
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
	return OpenEditorCmdWithOptions(path, lineNumber, targetBranch, editor, review.Options{})
}

func OpenEditorCmdWithOptions(path string, lineNumber int, targetBranch string, editor string, options review.Options) tea.Cmd {
	if options.Scope == review.Staged && lineNumber > 0 {
		lineNumber = workingTreeLine(path, lineNumber)
	}
	var args []string
	if lineNumber > 0 {
		args = append(args, fmt.Sprintf("+%d", lineNumber))
	}
	args = append(args, path)

	c := exec.Command(editor, args...)
	c.Dir = repoRoot()
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.Env = editorEnvironment(targetBranch, options)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func editorEnvironment(target string, options review.Options) []string {
	scope := string(options.Scope)
	if scope == "" {
		scope = "working"
	}
	if options.Scope == review.Unstaged {
		target = ""
	}
	return append(os.Environ(), "DIFI_TARGET="+target, "DIFI_SCOPE="+scope)
}

func renameSource(target, path string, options review.Options) (string, error) {
	out, err := gitCmd(diffArgs(target, options, "--name-status", "-z")...).Output()
	if err != nil {
		return "", fmt.Errorf("git rename status: %w", err)
	}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i+1 < len(entries); {
		status := entries[i]
		if status != "" && (status[0] == 'R' || status[0] == 'C') {
			if i+2 >= len(entries) {
				break
			}
			if entries[i+2] == path {
				return entries[i+1], nil
			}
			i += 3
		} else {
			i += 2
		}
	}
	return "", nil
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
	return DiffStatsByFileWithOptions(targetBranch, review.Options{})
}

func DiffStatsByFileWithOptions(targetBranch string, options review.Options) (map[string][2]int, error) {
	cmd := gitCmd(diffArgs(targetBranch, options, "--numstat", "-z")...)
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
