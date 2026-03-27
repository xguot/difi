package vcs

import (
	"os"
	"testing"
)

// TestVCSInterfaceConsistency tests that both Git and Mercurial VCS implementations
// provide consistent interfaces and handle edge cases similarly.
func TestVCSInterfaceConsistency(t *testing.T) {
	implementations := []struct {
		name string
		vcs  VCS
	}{
		{"Git", GitVCS{}},
		{"Mercurial", HgVCS{}},
	}

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			vcs := impl.vcs

			t.Run("GetCurrentBranch", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s GetCurrentBranch() panicked: %v", impl.name, r)
					}
				}()
				branch := vcs.GetCurrentBranch()
				if branch == "" {
					t.Logf("%s GetCurrentBranch() returned empty string", impl.name)
				}
			})

			t.Run("GetRepoName", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s GetRepoName() panicked: %v", impl.name, r)
					}
				}()
				repoName := vcs.GetRepoName()
				if repoName == "" {
					t.Logf("%s GetRepoName() returned empty string", impl.name)
				}
			})

			t.Run("ListChangedFiles", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s ListChangedFiles() panicked: %v", impl.name, r)
					}
				}()
				testBranches := []string{"main", "master", "default", "HEAD"}
				for _, branch := range testBranches {
					files, err := vcs.ListChangedFiles(branch)
					_ = files
					_ = err
				}
			})

			t.Run("DiffStats", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s DiffStats() panicked: %v", impl.name, r)
					}
				}()
				added, deleted, err := vcs.DiffStats("main")
				_ = added
				_ = deleted
				_ = err
			})

			t.Run("DiffStatsByFile", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s DiffStatsByFile() panicked: %v", impl.name, r)
					}
				}()
				byFile, err := vcs.DiffStatsByFile("main")
				_ = byFile
				_ = err
			})

			t.Run("ParseFilesFromDiff", func(t *testing.T) {
				files := vcs.ParseFilesFromDiff("")
				if len(files) != 0 {
					t.Errorf("%s ParseFilesFromDiff('') should return empty slice, got %d files", impl.name, len(files))
				}

				files = vcs.ParseFilesFromDiff("not a diff")
				if len(files) != 0 {
					t.Errorf("%s ParseFilesFromDiff('not a diff') should return empty slice, got %d files", impl.name, len(files))
				}
			})

			t.Run("ExtractFileDiff", func(t *testing.T) {
				result := vcs.ExtractFileDiff("", "file.txt")
				if result != "" {
					t.Errorf("%s ExtractFileDiff('', 'file.txt') should return empty string, got %q", impl.name, result)
				}

				result = vcs.ExtractFileDiff("some diff", "")
				if result != "" {
					t.Errorf("%s ExtractFileDiff('some diff', '') should return empty string, got %q", impl.name, result)
				}
			})

			t.Run("CalculateFileLine", func(t *testing.T) {
				line := vcs.CalculateFileLine("", 0)
				if line != 1 && line != 0 {
					t.Errorf("%s CalculateFileLine('', 0) should return 1 or 0, got %d", impl.name, line)
				}

				line = vcs.CalculateFileLine("single line", 10)
				if line < 0 {
					t.Errorf("%s CalculateFileLine with out-of-bounds index should not return negative, got %d", impl.name, line)
				}
			})
		})
	}
}

// TestDiffMsgType ensures both implementations use compatible message types.
func TestDiffMsgType(t *testing.T) {
	diffMsg := DiffMsg{Content: "test"}
	if diffMsg.Content != "test" {
		t.Error("DiffMsg.Content not set correctly")
	}

	editorMsg := EditorFinishedMsg{Err: nil}
	if editorMsg.Err != nil {
		t.Error("EditorFinishedMsg.Err not set correctly")
	}
}

// BenchmarkVCSDetection benchmarks the VCS detection performance.
func BenchmarkVCSDetection(b *testing.B) {
	tempDir := b.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			b.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		b.Fatalf("Failed to change to temp dir: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectVCS()
	}
}
