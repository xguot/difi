// Package review defines the comparison being reviewed across the CLI, VCS,
// and terminal UI.
package review

type Scope string

const (
	WorkingTree Scope = ""
	Staged      Scope = "staged"
	Unstaged    Scope = "unstaged"
)

type Options struct {
	Scope       Scope
	NoUntracked bool
}

func (o Options) IncludeUntracked() bool {
	return o.Scope != Staged && !o.NoUntracked
}
