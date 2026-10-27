package git

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xguot/difi/internal/review"
)

func TestReviewScopesWithMixedIndexAndWorkingTree(t *testing.T) {
	testRepo(t)
	for _, path := range []string{"both", "staged", "unstaged", "cancelled"} {
		writeFile(t, path, "base\n")
	}
	runGit(t, "add", ".")
	runGit(t, "commit", "-qm", "base")
	for _, path := range []string{"both", "staged", "cancelled", "new-index"} {
		writeFile(t, path, "index\n")
	}
	runGit(t, "add", "both", "staged", "cancelled", "new-index")
	writeFile(t, "both", "working\n")
	writeFile(t, "unstaged", "working\n")
	writeFile(t, "cancelled", "base\n")
	writeFile(t, "untracked", "unknown\n")
	for _, tt := range []struct {
		name           string
		options        review.Options
		files          []string
		added, removed string
	}{
		{"default", review.Options{}, []string{"both", "staged", "unstaged", "new-index", "untracked"}, "+working", "-base"},
		{"tracked", review.Options{NoUntracked: true}, []string{"both", "staged", "unstaged", "new-index"}, "+working", "-base"},
		{"staged", review.Options{Scope: review.Staged}, []string{"both", "staged", "cancelled", "new-index"}, "+index", "-base"},
		{"unstaged", review.Options{Scope: review.Unstaged}, []string{"both", "unstaged", "cancelled", "untracked"}, "+working", "-index"},
		{"unstaged tracked", review.Options{Scope: review.Unstaged, NoUntracked: true}, []string{"both", "unstaged", "cancelled"}, "+working", "-index"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files, err := ListChangedFilesWithOptions("HEAD", tt.options)
			if err != nil {
				t.Fatal(err)
			}
			sort.Strings(files)
			sort.Strings(tt.files)
			if !reflect.DeepEqual(files, tt.files) {
				t.Fatalf("files = %q, want %q", files, tt.files)
			}
			diff := DiffCmdWithOptions("HEAD", "both", tt.options)().(DiffMsg)
			if diff.Err != nil || !strings.Contains(diff.Content, tt.added) || !strings.Contains(diff.Content, tt.removed) {
				t.Fatalf("wrong comparison: %+v", diff)
			}
			stats, err := DiffStatsByFileWithOptions("HEAD", tt.options)
			if err != nil {
				t.Fatal(err)
			}
			for path := range stats {
				if sort.SearchStrings(files, path) == len(files) || files[sort.SearchStrings(files, path)] != path {
					t.Errorf("stats for file outside scope: %q", path)
				}
			}
			if stats["both"] != [2]int{1, 1} {
				t.Errorf("wrong scoped stats: %v", stats)
			}
		})
	}
	// A staged comparison against a target remains independent of the worktree.
	runGit(t, "commit", "-qm", "index")
	files, err := ListChangedFilesWithOptions("HEAD~1", review.Options{Scope: review.Staged})
	if err != nil || len(files) != 4 {
		t.Fatalf("staged target: %q, %v", files, err)
	}
}

func TestStagedFirstCommit(t *testing.T) {
	testRepo(t)
	writeFile(t, "first", "first commit\n")
	runGit(t, "add", "first")
	options := review.Options{Scope: review.Staged}
	files, err := ListChangedFilesWithOptions("HEAD", options)
	if err != nil || !reflect.DeepEqual(files, []string{"first"}) {
		t.Fatalf("unborn index: %q, %v", files, err)
	}
	diff := DiffCmdWithOptions("HEAD", "first", options)().(DiffMsg)
	if diff.Err != nil || !strings.Contains(diff.Content, "+first commit") {
		t.Fatalf("unborn diff: %+v", diff)
	}
}

func TestStagedEditorLineMapping(t *testing.T) {
	for _, tt := range []struct {
		diff       string
		line, want int
	}{
		{"@@ -0,0 +1,2 @@\n+one\n+two", 5, 7},
		{"@@ -3,0 +4,2 @@", 3, 3},
		{"@@ -3,0 +4,2 @@", 4, 6},
		{"@@ -3,2 +2,0 @@", 4, 2},
		{"@@ -3,2 +3,1 @@", 4, 3},
		{"@@ -3,2 +3,1 @@\n@@ -8,0 +8,2 @@", 10, 11},
	} {
		if got := mapIndexLine(tt.diff, tt.line); got != tt.want {
			t.Errorf("mapIndexLine(%q, %d) = %d, want %d", tt.diff, tt.line, got, tt.want)
		}
	}
	testRepo(t)
	writeFile(t, "file", "one\ntwo\nthree\n")
	runGit(t, "add", "file")
	runGit(t, "commit", "-qm", "base")
	writeFile(t, "file", "inserted\none\ntwo\nthree\n")
	if got := workingTreeLine("file", 3); got != 4 {
		t.Fatalf("editor line = %d, want 4", got)
	}
}

func TestScopedRenamesAndDeletes(t *testing.T) {
	testRepo(t)
	writeFile(t, "old name", "unchanged content\n")
	writeFile(t, "deleted", "delete this\n")
	runGit(t, "add", ".")
	runGit(t, "commit", "-qm", "base")
	runGit(t, "mv", "old name", "new name")
	runGit(t, "rm", "deleted")
	options := review.Options{Scope: review.Staged}
	diff := DiffCmdWithOptions("HEAD", "new name", options)().(DiffMsg)
	if diff.Err != nil || !strings.Contains(diff.Content, "rename from old name") || strings.Contains(diff.Content, "+unchanged content") {
		t.Fatalf("staged rename displayed as addition: %+v", diff)
	}
	diff = DiffCmdWithOptions("HEAD", "deleted", options)().(DiffMsg)
	if diff.Err != nil || !strings.Contains(diff.Content, "-delete this") {
		t.Fatalf("staged deletion: %+v", diff)
	}
	files, err := ListChangedFilesWithOptions("HEAD", review.Options{Scope: review.Unstaged})
	if err != nil || len(files) != 0 {
		t.Fatalf("staged rename leaked into unstaged scope: %q, %v", files, err)
	}
}

func TestEditorComparisonEnvironment(t *testing.T) {
	for _, tt := range []struct {
		scope         review.Scope
		target, label string
	}{
		{review.WorkingTree, "main", "working"},
		{review.Staged, "main", "staged"},
		{review.Unstaged, "", "unstaged"},
	} {
		var target, scope string
		for _, entry := range editorEnvironment("main", review.Options{Scope: tt.scope}) {
			if strings.HasPrefix(entry, "DIFI_TARGET=") {
				target = strings.TrimPrefix(entry, "DIFI_TARGET=")
			}
			if strings.HasPrefix(entry, "DIFI_SCOPE=") {
				scope = strings.TrimPrefix(entry, "DIFI_SCOPE=")
			}
		}
		if target != tt.target || scope != tt.label {
			t.Errorf("editor scope=%q target=%q", scope, target)
		}
	}
}
