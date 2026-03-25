package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/oug-t/difi/internal/config"
)

var (
	nord0  = lipgloss.Color("#2E3440")
	nord3  = lipgloss.Color("#4C566A")
	nord4  = lipgloss.Color("#D8DEE9")
	nord11 = lipgloss.Color("#BF616A")
	nord14 = lipgloss.Color("#A3BE8C")
	nord9  = lipgloss.Color("#81A1C1")

	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(nord3).
			Padding(0, 1)

	FocusedPaneStyle = PaneStyle.Copy().
				BorderForeground(nord9)

	TopBarStyle = lipgloss.NewStyle().
			Background(nord0).
			Foreground(nord4).
			Height(1)

	TopInfoStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	TopStatsAddedStyle = lipgloss.NewStyle().
				Foreground(nord14).
				PaddingLeft(1)

	TopStatsDeletedStyle = lipgloss.NewStyle().
				Foreground(nord11).
				PaddingLeft(1).
				PaddingRight(1)

	DirectoryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	FileStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	DiffStyle       = lipgloss.NewStyle().Padding(0, 0)
	LineNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(4).Align(lipgloss.Right).MarginRight(1)

	DiffAddGutter = lipgloss.NewStyle().Foreground(nord14).Bold(true)
	DiffDelGutter = lipgloss.NewStyle().Foreground(nord11).Bold(true)
	DiffCtxGutter = lipgloss.NewStyle().Foreground(nord3)

	DiffAddLineStyle lipgloss.Style
	DiffDelLineStyle lipgloss.Style

	// Dynamic full-line cursor styles
	CursorNormalStyle lipgloss.Style
	CursorAddStyle    lipgloss.Style
	CursorDelStyle    lipgloss.Style

	EmptyLogoStyle   = lipgloss.NewStyle().Foreground(nord9).Bold(true).MarginBottom(1)
	EmptyDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginBottom(1)
	EmptyStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginBottom(2)
	EmptyHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true).MarginBottom(1)
	EmptyCodeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	HelpDrawerStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(nord3).Padding(1, 2)
	HelpTextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginRight(2)

	StatusBarStyle     = lipgloss.NewStyle().Background(nord0).Foreground(nord4).Height(1)
	StatusKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	StatusRepoStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")).Padding(0, 1)
	StatusBranchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Padding(0, 1)
	StatusAddedStyle   = lipgloss.NewStyle().Foreground(nord14).Padding(0, 1)
	StatusDeletedStyle = lipgloss.NewStyle().Foreground(nord11).Padding(0, 1)
	StatusDividerStyle = lipgloss.NewStyle().Foreground(nord3).Padding(0, 1)

	ColorText = lipgloss.Color("252")
)

func InitStyles(cfg config.Config) {
	addBg := cfg.UI.DiffAddBg
	if addBg == "" {
		addBg = "#1A251E"
	}

	delBg := cfg.UI.DiffDelBg
	if delBg == "" {
		delBg = "#2D1A1A"
	}

	DiffAddLineStyle = lipgloss.NewStyle().Background(lipgloss.Color(addBg))
	DiffDelLineStyle = lipgloss.NewStyle().Background(lipgloss.Color(delBg))

	// Light pastel backgrounds for the selected cursor line with forced dark text
	CursorNormalStyle = lipgloss.NewStyle().Background(lipgloss.Color("#434C5E")).Foreground(lipgloss.Color("#ECEFF4")) // Gray
	CursorAddStyle = lipgloss.NewStyle().Background(lipgloss.Color("#A3E4D7")).Foreground(lipgloss.Color("#1A251E"))    // Mint Green
	CursorDelStyle = lipgloss.NewStyle().Background(lipgloss.Color("#F5B7B1")).Foreground(lipgloss.Color("#2D1A1A"))    // Pinky Red
}
