// Package patch reads Git and Mercurial unified diffs independently of the
// repository that produced them, including Git's configurable path prefixes.
package patch

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type File struct {
	Path    string
	Content string
	Added   int
	Deleted int
}

// Addition displays an untracked file without requiring a second VCS tool.
func Addition(path string, data []byte) string {
	old, next := strconv.Quote("a/"+path), strconv.Quote("b/"+path)
	header := fmt.Sprintf("diff --git %s %s\nnew file mode 100644\n", old, next)
	if bytes.ContainsRune(data, 0) {
		return header + fmt.Sprintf("Binary files /dev/null and %s differ\n", next)
	}
	if len(data) == 0 {
		return header
	}
	text := string(data)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	var out strings.Builder
	out.WriteString(header)
	fmt.Fprintf(&out, "--- /dev/null\n+++ %s\n@@ -0,0 +1,%d @@\n", next, len(lines))
	for _, line := range lines {
		out.WriteString("+" + line + "\n")
	}
	if !strings.HasSuffix(text, "\n") {
		out.WriteString("\\ No newline at end of file\n")
	}
	return out.String()
}

func Parse(text string) []File {
	var files []File
	var lines []string
	flush := func() {
		if len(lines) == 0 {
			return
		}
		file := parseFile(lines)
		if file.Path != "" {
			files = append(files, file)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		clean := ansi.Strip(line)
		if strings.HasPrefix(clean, "diff --git ") || strings.HasPrefix(clean, "diff -r ") {
			flush()
			lines = nil
		}
		if len(lines) > 0 || strings.HasPrefix(clean, "diff --git ") || strings.HasPrefix(clean, "diff -r ") {
			lines = append(lines, line)
		}
	}
	flush()
	return files
}

func Paths(text string) []string {
	var paths []string
	seen := make(map[string]bool)
	for _, file := range Parse(text) {
		if !seen[file.Path] {
			paths = append(paths, file.Path)
			seen[file.Path] = true
		}
	}
	return paths
}

func Extract(text, path string) string {
	var sections []string
	for _, file := range Parse(text) {
		if file.Path == path {
			sections = append(sections, file.Content)
		}
	}
	return strings.Join(sections, "\n")
}

func parseFile(lines []string) File {
	file := File{Content: strings.Join(lines, "\n")}
	header := ansi.Strip(lines[0])
	oldPath, newPath := "", ""
	if strings.HasPrefix(header, "diff --git ") {
		oldPath, newPath = splitHeader(strings.TrimPrefix(header, "diff --git "))
	} else {
		// Mercurial headers contain one or two revision arguments.
		parts := strings.SplitN(header, " ", 4)
		if len(parts) == 4 {
			file.Path = parts[3]
			if strings.HasPrefix(file.Path, "-r ") {
				if _, path, ok := strings.Cut(strings.TrimPrefix(file.Path, "-r "), " "); ok {
					file.Path = path
				}
			}
		}
	}
	var markerOld, markerNew, renamed string
	inHunk := false
	for _, raw := range lines[1:] {
		line := strings.TrimSuffix(ansi.Strip(raw), "\r")
		if strings.HasPrefix(line, "@@ ") {
			inHunk = true
			continue
		}
		if inHunk {
			if strings.HasPrefix(line, "+") {
				file.Added++
			} else if strings.HasPrefix(line, "-") {
				file.Deleted++
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "rename to "):
			renamed = decodePath(strings.TrimPrefix(line, "rename to "))
		case strings.HasPrefix(line, "copy to "):
			renamed = decodePath(strings.TrimPrefix(line, "copy to "))
		case strings.HasPrefix(line, "--- "):
			markerOld = markerPath(line[4:])
		case strings.HasPrefix(line, "+++ "):
			markerNew = markerPath(line[4:])
		}
	}
	if file.Path != "" { // Native Mercurial paths do not have Git prefixes.
		return file
	}
	if renamed != "" {
		file.Path = renamed
		return file
	}
	// Equal header paths indicate --no-prefix. Otherwise derive the prefix
	// from the pair, rather than removing the first directory unconditionally.
	stripPrefix := oldPath != newPath && pairedPrefixes(oldPath, newPath)
	if markerNew != "" && markerNew != "/dev/null" {
		newPath = markerNew
	} else if markerOld != "" {
		newPath = markerOld
	}
	if stripPrefix {
		_, newPath, _ = strings.Cut(newPath, "/")
	}
	if newPath != "/dev/null" {
		file.Path = newPath
	}
	return file
}

func markerPath(value string) string {
	value, _, _ = strings.Cut(value, "\t")
	return decodePath(value)
}

func decodePath(value string) string {
	if strings.HasPrefix(value, "\"") {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	return value
}

func pairedPrefixes(old, next string) bool {
	a, oldTail, oldOK := strings.Cut(old, "/")
	b, nextTail, nextOK := strings.Cut(next, "/")
	if !oldOK || !nextOK {
		return false
	}
	if oldTail == nextTail && a != b {
		return true
	}
	return strings.Contains(" a i c o 1 ", " "+a+" ") && strings.Contains(" b w i 2 ", " "+b+" ")
}

func splitHeader(value string) (string, string) {
	// Git quotes paths with tabs, newlines, quotes, and non-ASCII bytes.
	if strings.HasPrefix(value, "\"") {
		escaped := false
		for i := 1; i < len(value); i++ {
			if value[i] == '"' && !escaped {
				return decodePath(value[:i+1]), decodePath(strings.TrimPrefix(value[i+1:], " "))
			}
			if value[i] == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
	}
	// Spaces alone do not cause Git to quote a path. Find the split whose
	// paths agree (with or without prefixes) before trying renamed paths.
	for i := range value {
		if value[i] == ' ' {
			a, b := value[:i], decodePath(value[i+1:])
			_, at, _ := strings.Cut(a, "/")
			_, bt, _ := strings.Cut(b, "/")
			if a == b || (at != "" && at == bt) {
				return a, b
			}
		}
	}
	for i := range value {
		if value[i] == ' ' && pairedPrefixes(value[:i], value[i+1:]) {
			return value[:i], decodePath(value[i+1:])
		}
	}
	a, b, _ := strings.Cut(value, " ")
	return a, decodePath(b)
}
