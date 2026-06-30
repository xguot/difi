package diffsplit

import "testing"

func TestBuild_contextAndChanges(t *testing.T) {
	diffLines := []string{
		"@@ -10,4 +10,5 @@",
		" func main() {",
		`-	fmt.Println("old")`,
		`+	fmt.Println("new")`,
		`+	fmt.Println("added")`,
		" }",
	}
	highlighted := []string{
		"@@ -10,4 +10,5 @@",
		"func main() {",
		`fmt.Println("old")`,
		`fmt.Println("new")`,
		`fmt.Println("added")`,
		"}",
	}

	rows := Build(diffLines, highlighted)
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}

	// Context row
	if rows[0].Left.Kind != KindContext || rows[0].Right.Kind != KindContext {
		t.Errorf("row 0: want context on both sides")
	}
	if rows[0].Left.LineNo != 10 || rows[0].Right.LineNo != 10 {
		t.Errorf("row 0: line numbers = %d/%d, want 10/10", rows[0].Left.LineNo, rows[0].Right.LineNo)
	}

	// Paired change: deletion left, addition right
	if rows[1].Left.Kind != KindDel || rows[1].Right.Kind != KindAdd {
		t.Errorf("row 1: want del/add pair, got %v/%v", rows[1].Left.Kind, rows[1].Right.Kind)
	}
	if rows[1].Left.Content != "\tfmt.Println(\"old\")" {
		t.Errorf("row 1 left content = %q", rows[1].Left.Content)
	}
	if rows[1].Right.Content != "\tfmt.Println(\"new\")" {
		t.Errorf("row 1 right content = %q", rows[1].Right.Content)
	}

	// Pure addition on right
	if rows[2].Left.Kind != KindEmpty || rows[2].Right.Kind != KindAdd {
		t.Errorf("row 2: want empty/add, got %v/%v", rows[2].Left.Kind, rows[2].Right.Kind)
	}
	if rows[2].Right.LineNo != 12 {
		t.Errorf("row 2 right line = %d, want 12", rows[2].Right.LineNo)
	}

	// Trailing context
	if rows[3].Left.LineNo != 12 || rows[3].Right.LineNo != 13 {
		t.Errorf("row 3 line numbers = %d/%d, want 12/13", rows[3].Left.LineNo, rows[3].Right.LineNo)
	}
}

func TestBuild_pureDeletion(t *testing.T) {
	diffLines := []string{
		"@@ -1,2 +1,1 @@",
		"-removed",
		" context",
	}
	highlighted := []string{"@@", "removed", "context"}

	rows := Build(diffLines, highlighted)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Left.Kind != KindDel || rows[0].Right.Kind != KindEmpty {
		t.Errorf("row 0: want del/empty")
	}
	if rows[1].Left.Kind != KindContext {
		t.Errorf("row 1: want left context")
	}
}

func TestBuild_pureAddition(t *testing.T) {
	diffLines := []string{
		"@@ -0,0 +1,1 @@",
		"+added",
	}
	highlighted := []string{"@@", "added"}

	rows := Build(diffLines, highlighted)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Left.Kind != KindEmpty || rows[0].Right.Kind != KindAdd {
		t.Errorf("row 0: want empty/add")
	}
}

func TestRow_PrimaryRawIdx(t *testing.T) {
	row := Row{
		Left:  Side{RawIdx: 5, Kind: KindDel},
		Right: Side{RawIdx: 6, Kind: KindAdd},
	}
	if row.PrimaryRawIdx() != 6 {
		t.Errorf("PrimaryRawIdx() = %d, want 6", row.PrimaryRawIdx())
	}

	row = Row{Left: Side{RawIdx: 5, Kind: KindDel}}
	if row.PrimaryRawIdx() != 5 {
		t.Errorf("PrimaryRawIdx() for del-only = %d, want 5", row.PrimaryRawIdx())
	}
}
