package diffsplit

import (
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// LineKind describes the role of a side in a split row.
type LineKind int

const (
	KindEmpty LineKind = iota
	KindContext
	KindAdd
	KindDel
)

// Side is one column cell in a split diff row.
type Side struct {
	RawIdx      int
	Content     string
	Highlighted string
	Kind        LineKind
	LineNo      int // source file line number, 0 when empty
}

// Row is a GitHub-style split diff row with optional left and right content.
type Row struct {
	Left  Side
	Right Side
}

type pendingLine struct {
	rawIdx      int
	content     string
	highlighted string
	kind        LineKind
}

// Build converts unified diff display lines into side-by-side rows.
// diffLines and highlighted must have the same length; only hunk content
// lines (context, additions, deletions) are included.
func Build(diffLines, highlighted []string) []Row {
	var rows []Row
	var dels, adds []pendingLine
	oldLine, newLine := 0, 0
	inHunk := false

	flush := func() {
		n := len(dels)
		if len(adds) > n {
			n = len(adds)
		}
		for i := 0; i < n; i++ {
			var row Row
			if i < len(dels) {
				row.Left = sideFromPending(dels[i], oldLine)
				oldLine++
			}
			if i < len(adds) {
				row.Right = sideFromPending(adds[i], newLine)
				newLine++
			}
			rows = append(rows, row)
		}
		dels = dels[:0]
		adds = adds[:0]
	}

	for i, raw := range diffLines {
		clean := strings.TrimRight(stripAnsi(raw), "\r")
		hl := ""
		if i < len(highlighted) {
			hl = highlighted[i]
		}

		if m := hunkHeaderRe.FindStringSubmatch(clean); len(m) > 2 {
			flush()
			oldStart, _ := strconv.Atoi(m[1])
			newStart, _ := strconv.Atoi(m[2])
			if oldStart < 1 {
				oldStart = 1
			}
			if newStart < 1 {
				newStart = 1
			}
			oldLine = oldStart
			newLine = newStart
			inHunk = true
			continue
		}

		if !inHunk {
			continue
		}

		isAdd := strings.HasPrefix(clean, "+") && !strings.HasPrefix(clean, "+++")
		isDel := strings.HasPrefix(clean, "-") && !strings.HasPrefix(clean, "---")
		isCtx := strings.HasPrefix(clean, " ")

		if !isAdd && !isDel && !isCtx {
			continue
		}

		content := clean
		if len(content) > 0 {
			content = content[1:]
		}

		pl := pendingLine{
			rawIdx:      i,
			content:     content,
			highlighted: hl,
		}

		switch {
		case isCtx:
			flush()
			rows = append(rows, Row{
				Left:  sideFromPending(pendingLine{rawIdx: i, content: content, highlighted: hl, kind: KindContext}, oldLine),
				Right: sideFromPending(pendingLine{rawIdx: i, content: content, highlighted: hl, kind: KindContext}, newLine),
			})
			oldLine++
			newLine++
		case isDel:
			pl.kind = KindDel
			dels = append(dels, pl)
		case isAdd:
			pl.kind = KindAdd
			adds = append(adds, pl)
		}
	}

	flush()
	return rows
}

func sideFromPending(pl pendingLine, lineNo int) Side {
	if pl.kind == KindEmpty && pl.rawIdx < 0 {
		return Side{Kind: KindEmpty}
	}
	return Side{
		RawIdx:      pl.rawIdx,
		Content:     pl.content,
		Highlighted: pl.highlighted,
		Kind:        pl.kind,
		LineNo:      lineNo,
	}
}

// PrimaryRawIdx returns the diff line index used for editor line mapping.
// Prefer the right side when present (context/add), otherwise the left.
func (r Row) PrimaryRawIdx() int {
	if r.Right.Kind != KindEmpty && r.Right.RawIdx >= 0 {
		return r.Right.RawIdx
	}
	if r.Left.RawIdx >= 0 {
		return r.Left.RawIdx
	}
	return 0
}

// IsNavigable reports whether a row should receive cursor focus.
func (r Row) IsNavigable() bool {
	return r.Left.Kind != KindEmpty || r.Right.Kind != KindEmpty
}

var ansiRe = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func stripAnsi(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}
