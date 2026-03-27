package vcs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectVCS_GitPriority(t *testing.T) {
	tempDir := t.TempDir()

	gitDir := filepath.Join(tempDir, ".git")
	hgDir := filepath.Join(tempDir, ".hg")

	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	if err := os.Mkdir(hgDir, 0755); err != nil {
		t.Fatalf("Failed to create .hg dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(GitVCS{}) {
		t.Errorf("Expected GitVCS, got %T", vcs)
	}
}

func TestDetectVCS_GitOnly(t *testing.T) {
	tempDir := t.TempDir()

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(GitVCS{}) {
		t.Errorf("Expected GitVCS, got %T", vcs)
	}
}

func TestDetectVCS_HgOnly(t *testing.T) {
	tempDir := t.TempDir()

	hgDir := filepath.Join(tempDir, ".hg")
	if err := os.Mkdir(hgDir, 0755); err != nil {
		t.Fatalf("Failed to create .hg dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(HgVCS{}) {
		t.Errorf("Expected HgVCS, got %T", vcs)
	}
}

func TestDetectVCS_NoVCS(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(GitVCS{}) {
		t.Errorf("Expected GitVCS (default), got %T", vcs)
	}
}

func TestDetectVCS_NestedDirectories(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir", "nested")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to nested dir: %v", err)
	}

	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(GitVCS{}) {
		t.Errorf("Expected GitVCS (from parent), got %T", vcs)
	}
}

func TestDetectVCS_GitInParentHgInChild(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	hgDir := filepath.Join(subDir, ".hg")
	if err := os.Mkdir(hgDir, 0755); err != nil {
		t.Fatalf("Failed to create .hg dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subdir: %v", err)
	}

	// We expect the detector to find the .git directory in the parent
	// rather than the .hg directory in the current working directory.
	vcs := DetectVCS()
	if reflect.TypeOf(vcs) != reflect.TypeOf(GitVCS{}) {
		t.Errorf("Expected GitVCS (priority), got %T", vcs)
	}
}

func TestVCSInterface_GitVCS(t *testing.T) {
	var vcs VCS = GitVCS{}

	_ = vcs.GetCurrentBranch()
	_ = vcs.GetRepoName()

	files, _ := vcs.ListChangedFiles("main")
	if files == nil {
		files = []string{}
	}
}

func TestVCSInterface_HgVCS(t *testing.T) {
	var vcs VCS = HgVCS{}

	_ = vcs.GetCurrentBranch()
	_ = vcs.GetRepoName()

	files, _ := vcs.ListChangedFiles("default")
	if files == nil {
		files = []string{}
	}
}

func TestDetectVCS_ErrorHandling(t *testing.T) {
	vcs := DetectVCS()

	if vcs == nil {
		t.Error("DetectVCS() returned nil, should always return a VCS implementation")
	}

	if _, ok := vcs.(GitVCS); !ok {
		if _, ok := vcs.(HgVCS); !ok {
			t.Errorf("DetectVCS() returned unexpected type %T", vcs)
		}
	}
}

