package diffsplit

import "testing"

func TestBuild_gitHunkHeaderWithContext(t *testing.T) {
	diffLines := []string{
		"@@ -10,4 +10,5 @@ func main() {",
		" func main() {",
		`-	fmt.Println("old")`,
		`+	fmt.Println("new")`,
		" }",
	}
	highlighted := make([]string, len(diffLines))

	rows := Build(diffLines, highlighted)
	if len(rows) == 0 {
		t.Fatal("expected rows from git-style hunk header with trailing context")
	}
}
