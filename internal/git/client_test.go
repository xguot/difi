package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func testRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	t.Chdir(root)
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Difi Test"}, {"config", "commit.gpgsign", "false"}, {"config", "core.autocrlf", "false"}} {
		runGit(t, args...)
	}
	return root
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFilesAndStatsFromSubdirectory(t *testing.T) {
	root := testRepo(t)
	paths := []string{"space  name.go", "nested/main.go", "renamed.go", "[file].go"}
	if runtime.GOOS != "windows" {
		paths = append(paths, "tab\tname.go", "line\nname.go", " trailing ")
	}
	if err := os.Mkdir("nested", 0755); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		writeFile(t, path, "same\nold\nsame end\n")
	}
	runGit(t, "add", ".")
	runGit(t, "commit", "-qm", "initial")
	runGit(t, "mv", "renamed.go", "new.go")
	paths[2] = "new.go"
	for _, path := range paths {
		writeFile(t, path, "same\nnew\nsame end\n")
	}
	writeFile(t, "untracked.go", "new file\n")
	t.Chdir(filepath.Join(root, "nested"))
	files, err := ListChangedFiles("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range append(paths, "untracked.go") {
		found := false
		for _, file := range files {
			if file == path {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q from %q", path, files)
		}
	}
	stats, err := DiffStatsByFile("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if got := stats[path]; got != [2]int{1, 1} {
			t.Errorf("stats[%q] = %v", path, got)
		}
	}
	diff := DiffCmd("HEAD", "space  name.go")().(DiffMsg)
	if diff.Err != nil || !strings.Contains(diff.Content, "+new") {
		t.Fatalf("subdirectory diff: %+v", diff)
	}
	unknown := DiffCmd("HEAD", "untracked.go")().(DiffMsg)
	if unknown.Err != nil || !strings.Contains(unknown.Content, "+new file") {
		t.Fatalf("untracked diff: %+v", unknown)
	}
}

func TestPipedGitConfigurations(t *testing.T) {
	testRepo(t)
	writeFile(t, "file with spaces.go", "old\n")
	runGit(t, "add", ".")
	runGit(t, "commit", "-qm", "initial")
	writeFile(t, "file with spaces.go", "new\n")
	for _, option := range []string{"diff.mnemonicPrefix=true", "diff.noprefix=true"} {
		text := runGit(t, "-c", option, "diff", "--color=never")
		if got := ParseFilesFromDiff(text); !reflect.DeepEqual(got, []string{"file with spaces.go"}) {
			t.Errorf("%s paths = %q", option, got)
		}
		if got := ExtractFileDiff(text, "file with spaces.go"); got != text {
			t.Errorf("%s extracted %q", option, got)
		}
	}
}

func TestRestoredFileAndInvalidTarget(t *testing.T) {
	testRepo(t)
	writeFile(t, "file.go", "old\n")
	runGit(t, "add", ".")
	runGit(t, "commit", "-qm", "initial")
	if got := DiffCmd("HEAD", "file.go")().(DiffMsg); got.Err != nil || got.Content != "" {
		t.Fatalf("clean file shown as addition: %+v", got)
	}
	if got := DiffCmd("missing-target", "file.go")().(DiffMsg); got.Err == nil {
		t.Fatal("invalid target did not return an error")
	}
}
