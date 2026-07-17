package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/xguot/difi/internal/config"
)

// Styles holds all lipgloss styles derived from the active theme.
// Styles are rebuilt whenever the theme changes.
type Styles struct {
	PaneStyle         lipgloss.Style
	FocusedPaneStyle  lipgloss.Style
	TopBarStyle       lipgloss.Style
	TopInfoStyle      lipgloss.Style
	TopStatsAddedStyle   lipgloss.Style
	TopStatsDeletedStyle lipgloss.Style
	DirectoryStyle    lipgloss.Style
	FileStyle         lipgloss.Style
	DiffStyle         lipgloss.Style
	LineNumberStyle   lipgloss.Style
	DiffAddGutter     lipgloss.Style
	DiffDelGutter     lipgloss.Style
	DiffCtxGutter     lipgloss.Style
	DiffAddLineStyle  lipgloss.Style
	DiffDelLineStyle  lipgloss.Style
	CursorNormalStyle lipgloss.Style
	CursorAddStyle    lipgloss.Style
	CursorDelStyle    lipgloss.Style
	EmptyLogoStyle    lipgloss.Style
	EmptyDescStyle    lipgloss.Style
	EmptyStatusStyle  lipgloss.Style
	EmptyHeaderStyle  lipgloss.Style
	EmptyCodeStyle    lipgloss.Style
	HelpDrawerStyle   lipgloss.Style
	HelpTextStyle     lipgloss.Style
	StatusBarStyle    lipgloss.Style
	StatusKeyStyle    lipgloss.Style
	StatusRepoStyle   lipgloss.Style
	StatusBranchStyle lipgloss.Style
	StatusAddedStyle  lipgloss.Style
	StatusDeletedStyle lipgloss.Style
	StatusDividerStyle lipgloss.Style
	CmdLineStyle      lipgloss.Style
	CmdLinePrompt     lipgloss.Style
	CmdLineCursor     lipgloss.Style
}

var S *Styles

// InitStyles builds all UI styles from the selected theme and user config overrides.
func InitStyles(cfg config.Config) {
	copyCfg := cfg
	lastCfg = &copyCfg

	t := GetTheme(cfg.UI.Theme)
	if t == nil {
		t = nord
	}

	addBg := cfg.UI.DiffAddBg
	if addBg == "" {
		addBg = t.DefaultAddBg()
	}

	delBg := cfg.UI.DiffDelBg
	if delBg == "" {
		delBg = t.DefaultDelBg()
	}

	S = &Styles{
		PaneStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),

		FocusedPaneStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.FocusedBorder).
			Padding(0, 1),

		TopBarStyle: lipgloss.NewStyle().
			Background(t.TopBarBg).
			Foreground(t.TopBarFg).
			Height(1),

		TopInfoStyle: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),

		TopStatsAddedStyle: lipgloss.NewStyle().
			Foreground(t.AddGutter).
			PaddingLeft(1),

		TopStatsDeletedStyle: lipgloss.NewStyle().
			Foreground(t.DelGutter).
			PaddingLeft(1).
			PaddingRight(1),

		DirectoryStyle: lipgloss.NewStyle().Foreground(t.DirFg),
		FileStyle:      lipgloss.NewStyle().Foreground(t.FileFg),

		DiffStyle: lipgloss.NewStyle().Padding(0, 0),

		LineNumberStyle: lipgloss.NewStyle().
			Foreground(t.DimFg).
			Width(4).
			Align(lipgloss.Right).
			MarginRight(1),

		DiffAddGutter: lipgloss.NewStyle().Foreground(t.AddGutter).Bold(true),
		DiffDelGutter: lipgloss.NewStyle().Foreground(t.DelGutter).Bold(true),
		DiffCtxGutter: lipgloss.NewStyle().Foreground(t.CtxGutter),

		DiffAddLineStyle: lipgloss.NewStyle().Background(lipgloss.Color(addBg)),
		DiffDelLineStyle: lipgloss.NewStyle().Background(lipgloss.Color(delBg)),

		CursorNormalStyle: lipgloss.NewStyle().
			Background(t.CursorNormalBg).
			Foreground(t.CursorNormalFg),

		CursorAddStyle: lipgloss.NewStyle().
			Background(t.CursorAddBg).
			Foreground(t.CursorAddFg),

		CursorDelStyle: lipgloss.NewStyle().
			Background(t.CursorDelBg).
			Foreground(t.CursorDelFg),

		EmptyLogoStyle: lipgloss.NewStyle().
			Foreground(t.FocusedBorder).
			Bold(true),

		EmptyDescStyle: lipgloss.NewStyle().
			Foreground(t.DimFg),

		EmptyStatusStyle: lipgloss.NewStyle().
			Foreground(t.DimFg),

		EmptyHeaderStyle: lipgloss.NewStyle().
			Foreground(t.DimFg).
			Bold(true).
			MarginBottom(1),

		EmptyCodeStyle: lipgloss.NewStyle().Foreground(t.DimFg),

		HelpDrawerStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(t.Border).
			Padding(1, 2),

		HelpTextStyle: lipgloss.NewStyle().
			Foreground(t.DimFg).
			MarginRight(2),

		StatusBarStyle: lipgloss.NewStyle().
			Background(t.StatusBarBg).
			Foreground(t.StatusBarFg).
			Height(1),

		StatusKeyStyle: lipgloss.NewStyle().
			Foreground(t.DimFg).
			Padding(0, 1),

		StatusRepoStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.StatusRepoFg).
			Padding(0, 1),

		StatusBranchStyle: lipgloss.NewStyle().
			Foreground(t.StatusBranchFg).
			Padding(0, 1),

		StatusAddedStyle: lipgloss.NewStyle().
			Foreground(t.StatusAddFg).
			Padding(0, 1),

		StatusDeletedStyle: lipgloss.NewStyle().
			Foreground(t.StatusDelFg).
			Padding(0, 1),

		StatusDividerStyle: lipgloss.NewStyle().
			Foreground(t.StatusDividerFg).
			Padding(0, 1),

		CmdLineStyle: lipgloss.NewStyle().
			Background(t.CmdLineBg).
			Foreground(t.CmdLineFg).
			Height(1),

		CmdLinePrompt: lipgloss.NewStyle().
			Background(t.CmdLineBg).
			Foreground(t.FocusedBorder).
			Bold(true),

		CmdLineCursor: lipgloss.NewStyle().
			Background(t.CmdLineBg).
			Foreground(t.CmdLineFg).
			Reverse(true),
	}
}

// CurrentTheme returns the currently active ThemeColors.
func CurrentTheme() *ThemeColors {
	return GetTheme("") // fallback will be nord if S was never initialized
}

// ActiveTheme returns the ThemeColors used to build S.
// This is needed by update.go for Chroma theme lookup.
func ActiveTheme() *ThemeColors {
	if cfg, ok := lastConfig(); ok {
		return GetTheme(cfg.UI.Theme)
	}
	return nord
}

// lastConfig is a package-level helper to retain the last config used in InitStyles.
var lastCfg *config.Config

func lastConfig() (config.Config, bool) {
	if lastCfg != nil {
		return *lastCfg, true
	}
	return config.Config{}, false
}

// ReinitStyles rebuilds styles from a new config (used after :colorscheme).
func ReinitStyles(cfg config.Config) {
	InitStyles(cfg)
}
