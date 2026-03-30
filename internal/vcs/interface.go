package vcs

import tea "github.com/charmbracelet/bubbletea"

type VCS interface {
	GetCurrentBranch() string
	GetRepoName() string
	ListChangedFiles(targetBranch string) ([]string, error)
	DiffCmd(targetBranch, path string) tea.Cmd
	OpenEditorCmd(path string, lineNumber int, targetBranch string, editor string) tea.Cmd
	DiffStats(targetBranch string) (added int, deleted int, err error)
	DiffStatsByFile(targetBranch string) (map[string][2]int, error)
	CalculateFileLine(diffContent []string, visualLineIndex int) int
	ParseFilesFromDiff(diffText string) []string
	ExtractFileDiff(diffText, targetPath string) string
}

type DiffMsg struct{ Content string }
type EditorFinishedMsg struct{ Err error }
