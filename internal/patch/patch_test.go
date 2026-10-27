package patch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestGitPathFormats(t *testing.T) {
	for _, tt := range []struct{ name, old, next, path string }{
		{"standard", "a/src/main.go", "b/src/main.go", "src/main.go"},
		{"mnemonic", "i/src/main.go", "w/src/main.go", "src/main.go"},
		{"commit", "c/src/main.go", "w/src/main.go", "src/main.go"},
		{"no prefix", "src/main.go", "src/main.go", "src/main.go"},
		{"custom", "old/src/main.go", "new/src/main.go", "src/main.go"},
		{"spaces", "a/path with spaces.go", "b/path with spaces.go", "path with spaces.go"},
		{"bare spaces", "path with spaces.go", "path with spaces.go", "path with spaces.go"},
		{"quoted", `"a/tab\tname.go"`, `"b/tab\tname.go"`, "tab\tname.go"},
		{"unicode", `"a/\303\251.go"`, `"b/\303\251.go"`, "é.go"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			text := fmt.Sprintf("diff --git %s %s\n--- %s\n+++ %s\n@@ -1 +1 @@\n---old\n+++new\n", tt.old, tt.next, tt.old, tt.next)
			files := Parse(text)
			if len(files) != 1 || files[0].Path != tt.path || files[0].Added != 1 || files[0].Deleted != 1 {
				t.Fatalf("unexpected parse: %+v", files)
			}
			if got := Extract(text, tt.path); got != text {
				t.Fatalf("extracted %q, want %q", got, text)
			}
		})
	}
}

func TestRenameDeleteBinaryAndExactMatching(t *testing.T) {
	text := "diff --git a/old name b/new name\nsimilarity index 100%\nrename from old name\nrename to new name\n" +
		"diff --git a/deleted b/deleted\ndeleted file mode 100644\n--- a/deleted\n+++ /dev/null\n@@ -1 +0,0 @@\n-old\n" +
		"diff --git a/image.png b/image.png\nBinary files a/image.png and b/image.png differ\n" +
		"diff --git a/file.go b/file.go\nold mode 100644\nnew mode 100755\n" +
		"diff --git a/file.go.bak b/file.go.bak\nold mode 100644\nnew mode 100755\n"
	want := []string{"new name", "deleted", "image.png", "file.go", "file.go.bak"}
	if got := Paths(text); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
	if got := Extract(text, "file.go"); strings.Contains(got, "file.go.bak") {
		t.Fatalf("matched a different file: %s", got)
	}
	if got := Extract(text, "old name"); got != "" {
		t.Fatalf("old rename path matched: %s", got)
	}
}

func TestMercurialAndColoredDiffs(t *testing.T) {
	for _, header := range []string{"diff -r abc file with spaces.go", "diff -r abc -r def file with spaces.go"} {
		files := Parse(header + "\n--- a/file with spaces.go\n+++ b/file with spaces.go\n@@ -1 +1 @@\n-old\n+new\n")
		if len(files) != 1 || files[0].Path != "file with spaces.go" {
			t.Fatalf("parse %q: %+v", header, files)
		}
	}
	text := "\x1b[1mdiff --git i/file w/file\x1b[0m\n\x1b[36m@@ -1 +1 @@\x1b[0m\n\x1b[31m-old\x1b[0m\n\x1b[32m+new\x1b[0m\n"
	if got := Parse(text); len(got) != 1 || got[0].Path != "file" || got[0].Added != 1 || got[0].Deleted != 1 {
		t.Fatalf("colored parse: %+v", got)
	}
}

func TestUntrackedAddition(t *testing.T) {
	for _, tt := range []struct {
		data   string
		added  int
		marker string
	}{
		{"first\nsecond\n", 2, "+second"},
		{"no newline", 1, "\\ No newline at end of file"},
		{"", 0, "new file mode"},
		{"binary\x00data", 0, "Binary files"},
	} {
		text := Addition("path with spaces", []byte(tt.data))
		files := Parse(text)
		if len(files) != 1 || files[0].Path != "path with spaces" || files[0].Added != tt.added || !strings.Contains(text, tt.marker) {
			t.Fatalf("addition %q: %+v", tt.data, files)
		}
	}
}
