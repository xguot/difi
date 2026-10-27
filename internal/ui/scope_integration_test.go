package ui

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/review"
	"github.com/xguot/difi/internal/vcs"
)

func TestNativeScopesRemainStableThroughRefresh(t *testing.T) {
	t.Chdir(t.TempDir())
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	git("config", "commit.gpgsign", "false")
	write("file", "base\n")
	git("add", ".")
	git("commit", "-qm", "base")
	write("file", "index\n")
	git("add", "file")
	write("file", "working\n")
	write("unknown", "untracked\n")
	for _, scope := range []review.Scope{review.Staged, review.Unstaged} {
		t.Run(string(scope), func(t *testing.T) {
			options := review.Options{Scope: scope, NoUntracked: true}
			m := NewModel(config.Config{UI: config.UIConfig{Theme: "git"}}, "HEAD", "", vcs.GitVCS{Options: options}, true, "test", Options{Review: options})
			m = drain(m, m.Init())
			if len(m.fileList.Items()) != 1 {
				t.Fatal("untracked file escaped the selected scope")
			}
			m.fileFilter, m.searchQuery = "fil", "index"
			write("file", "edited working\n")
			m, cmd := apply(m, vcs.EditorFinishedMsg{})
			m = drain(m, cmd)
			want := "+index"
			if scope == review.Unstaged {
				want = "+edited working"
			}
			if !strings.Contains(m.diffContent, want) || m.fileFilter != "fil" || m.searchQuery != "index" || len(m.fileList.Items()) != 1 {
				t.Fatalf("refresh changed scope: %q", m.diffContent)
			}
			if scope == review.Staged && strings.Contains(m.diffContent, "+edited working") {
				t.Fatal("working edit leaked into staged review")
			}
		})
	}
}
