package hg

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestWorkingTreeComparisonAndUntrackedFiles(t *testing.T) {
	if _, err := exec.LookPath("hg"); err != nil {
		t.Skip("Mercurial not installed")
	}
	t.Chdir(t.TempDir())
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("hg", args...)
		cmd.Env = append(os.Environ(), "HGRCPATH="+os.DevNull)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hg %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	run("init")
	write("file.go", "committed\n")
	run("add", "file.go")
	run("commit", "-m", "initial", "-u", "test")
	write("file.go", "working tree\n")
	write("new file.go", "untracked\n")
	diff := DiffCmd("tip", "file.go")().(DiffMsg)
	if diff.Err != nil || !strings.Contains(diff.Content, "+working tree") {
		t.Fatalf("wrong comparison: %+v", diff)
	}
	unknown := DiffCmd("tip", "new file.go")().(DiffMsg)
	if unknown.Err != nil || !strings.Contains(unknown.Content, "+untracked") {
		t.Fatalf("missing untracked diff: %+v", unknown)
	}
	stats, err := DiffStatsByFile("tip")
	if err != nil || stats["file.go"] != [2]int{1, 1} {
		t.Fatalf("wrong stats: %v, %v", stats, err)
	}
}
