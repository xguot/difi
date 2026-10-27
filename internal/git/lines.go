package git

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/xguot/difi/internal/review"
)

var zeroContextHunk = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Staged diffs use index line numbers. Translate through unstaged edits before
// opening a working file so an insertion above the selected line does not move
// the editor to an unrelated line.
func workingTreeLine(path string, line int) int {
	args := append(diffArgs("", review.Options{Scope: review.Unstaged}, "--color=never", "--unified=0"), path)
	out, err := gitCmd(args...).Output()
	if err != nil {
		return line
	}
	return mapIndexLine(string(out), line)
}

func mapIndexLine(diff string, line int) int {
	offset := 0
	for _, raw := range strings.Split(diff, "\n") {
		match := zeroContextHunk.FindStringSubmatch(raw)
		if match == nil {
			continue
		}
		oldStart, _ := strconv.Atoi(match[1])
		newStart, _ := strconv.Atoi(match[3])
		oldCount, newCount := 1, 1
		if match[2] != "" {
			oldCount, _ = strconv.Atoi(match[2])
		}
		if match[4] != "" {
			newCount, _ = strconv.Atoi(match[4])
		}
		if oldCount == 0 {
			if line <= oldStart {
				break
			}
		} else {
			if line < oldStart {
				break
			}
			if line < oldStart+oldCount {
				return max(1, newStart+min(line-oldStart, max(0, newCount-1)))
			}
		}
		offset += newCount - oldCount
	}
	return max(1, line+offset)
}
