package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopeFlagValidation(t *testing.T) {
	for _, tt := range []struct {
		args             []string
		piped, hg, valid bool
	}{
		{args: []string{"-u", "-p"}, valid: true},
		{args: []string{"-s", "main"}, valid: true},
		{args: []string{"--no-untracked"}, hg: true, valid: true},
		{args: []string{"--staged", "--unstaged"}},
		{args: []string{"-u", "main"}},
		{args: []string{"-s"}, hg: true},
		{args: []string{"-u"}, hg: true},
		{args: []string{"--no-untracked"}, piped: true},
		{args: []string{"--staged"}, piped: true},
		{args: []string{"main"}, piped: true},
		{args: []string{"-p"}, piped: true, valid: true},
		{args: []string{"one", "two"}},
		{args: []string{"--", "--ext-diff"}},
	} {
		options, err := parseArgs(tt.args, io.Discard)
		if err == nil {
			err = options.validate(tt.piped, tt.hg)
		}
		if (err == nil) != tt.valid {
			t.Errorf("%v (pipe=%t hg=%t): %v", tt.args, tt.piped, tt.hg, err)
		}
	}
}

func TestPlainPatchesIncludingEmptyInput(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, patch := range []string{"", "diff --git i/file w/file\n@@ -1 +1 @@\n-old\n+new\n"} {
		path := filepath.Join(t.TempDir(), "input.patch")
		if err := os.WriteFile(path, []byte(patch), 0644); err != nil {
			t.Fatal(err)
		}
		input, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		code := run([]string{"--plain"}, input, &stdout, &stderr)
		input.Close()
		want := ""
		if patch != "" {
			want = "file\n"
		}
		if code != 0 || stdout.String() != want {
			t.Fatalf("plain patch exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	}
}

func TestPlainNativeScopes(t *testing.T) {
	t.Chdir(t.TempDir())
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	git("config", "commit.gpgsign", "false")
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("index", "old\n")
	write("work", "old\n")
	git("add", ".")
	git("commit", "-qm", "base")
	write("index", "new\n")
	git("add", "index")
	write("work", "new\n")
	write("unknown", "new\n")
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"-p", "-s"}, "index\n"},
		{[]string{"-p", "-u", "--no-untracked"}, "work\n"},
		{[]string{"-p", "--no-untracked"}, "index\nwork\n"},
	} {
		input, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		code := run(tt.args, input, &stdout, &stderr)
		input.Close()
		if code != 0 || stdout.String() != tt.want {
			t.Fatalf("%v exit=%d stdout=%q stderr=%q", tt.args, code, stdout.String(), stderr.String())
		}
	}
}

func TestHelpAndUsageExitCodes(t *testing.T) {
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	for _, tt := range []struct {
		args     []string
		code     int
		contains string
	}{
		{[]string{"--help"}, 0, "--unstaged"},
		{[]string{"--version"}, 0, "difi version"},
		{[]string{"--staged", "--unstaged"}, 2, "cannot be combined"},
		{[]string{"-u", "HEAD"}, 2, "does not accept a target"},
		{[]string{"--vcs", "hg", "-s"}, 2, "require Git"},
	} {
		var output bytes.Buffer
		code := run(tt.args, input, &output, &output)
		if code != tt.code || !strings.Contains(output.String(), tt.contains) {
			t.Errorf("%v exit=%d output=%q", tt.args, code, output.String())
		}
	}
}
