package ui

import "github.com/charmbracelet/lipgloss"

// ThemeColors holds the semantic color slots that define a complete UI theme.
// Each slot maps to a specific UI element. Preset themes fill these from
// well-known vim/neovim colorscheme palettes.
type ThemeColors struct {
	Name        string // Display name, e.g. "Nord"
	ChromaStyle string // Chroma style name for syntax highlighting, e.g. "nord"
	IsDark      bool   // Whether this is a dark-background theme

	// Core background / foreground
	Bg lipgloss.Color
	Fg lipgloss.Color

	// Dimmed foreground for secondary text (line numbers, help text, empty states)
	DimFg lipgloss.Color

	// Pane borders
	Border         lipgloss.Color
	FocusedBorder  lipgloss.Color

	// Diff gutter symbols (+, -, space)
	AddGutter lipgloss.Color
	DelGutter lipgloss.Color
	CtxGutter lipgloss.Color

	// Diff line backgrounds (the full-width highlight behind added/deleted lines)
	AddLineBg lipgloss.Color
	DelLineBg lipgloss.Color

	// Cursor line (the currently selected line) — bg + fg per line type
	CursorNormalBg lipgloss.Color
	CursorNormalFg lipgloss.Color
	CursorAddBg    lipgloss.Color
	CursorAddFg    lipgloss.Color
	CursorDelBg    lipgloss.Color
	CursorDelFg    lipgloss.Color

	// Top bar (repo info, file path, stats)
	TopBarBg lipgloss.Color
	TopBarFg lipgloss.Color

	// Status bar (bottom shortcuts)
	StatusBarBg lipgloss.Color
	StatusBarFg lipgloss.Color

	// Status bar accent colors for repo name, branch, stats
	StatusRepoFg   lipgloss.Color
	StatusBranchFg lipgloss.Color
	StatusAddFg    lipgloss.Color
	StatusDelFg    lipgloss.Color
	StatusDividerFg lipgloss.Color

	// File tree
	DirFg          lipgloss.Color
	FileFg         lipgloss.Color
	SelectedItemBg lipgloss.Color
	SelectedItemFg lipgloss.Color

	// Command line (vim-style : prompt at bottom)
	CmdLineBg lipgloss.Color
	CmdLineFg lipgloss.Color
}

// DefaultAddBg returns the theme's AddLineBg, or a fallback dark-green.
func (t *ThemeColors) DefaultAddBg() string {
	if t.AddLineBg != "" {
		return string(t.AddLineBg)
	}
	return "#1A251E"
}

// DefaultDelBg returns the theme's DelLineBg, or a fallback dark-red.
func (t *ThemeColors) DefaultDelBg() string {
	if t.DelLineBg != "" {
		return string(t.DelLineBg)
	}
	return "#2D1A1A"
}

// themeRegistry maps user-facing theme names (from config.yaml) to ThemeColors.
var themeRegistry = map[string]*ThemeColors{
	"nord":               nord,
	"gruvbox":            gruvbox,
	"catppuccin-mocha":   catppuccinMocha,
	"catppuccin-latte":   catppuccinLatte,
	"dracula":            dracula,
	"monokai":            monokai,
	"onedark":            onedark,
	"github":             githubLight,
	"github-dark":        githubDark,
	"rose-pine":          rosePine,
	"rose-pine-dawn":     rosePineDawn,
	"solarized-dark":     solarizedDark,
	"tokyonight-night":   tokyonightNight,
	"tokyonight-storm":   tokyonightStorm,
	"evergarden":         evergarden,
	"doom-one":           doomOne,
	"quiet":              quiet,
}

// GetTheme returns the ThemeColors for a given name.
// Falls back to "nord" if the name is unknown.
func GetTheme(name string) *ThemeColors {
	if t, ok := themeRegistry[name]; ok {
		return t
	}
	return nord
}

// ThemeNames returns a sorted list of all available theme names.
func ThemeNames() []string {
	names := make([]string, 0, len(themeRegistry))
	for name := range themeRegistry {
		names = append(names, name)
	}
	return names
}

// ──────────────────────────────────────────────────────────────
// Preset theme palettes
// ──────────────────────────────────────────────────────────────

func c(hex string) lipgloss.Color { return lipgloss.Color(hex) }

var nord = &ThemeColors{
	Name: "Nord", ChromaStyle: "nord", IsDark: true,
	Bg: c("#2E3440"), Fg: c("#D8DEE9"), DimFg: c("#4C566A"),
	Border: c("#4C566A"), FocusedBorder: c("#81A1C1"),
	AddGutter: c("#A3BE8C"), DelGutter: c("#BF616A"), CtxGutter: c("#4C566A"),
	AddLineBg: c("#1A251E"), DelLineBg: c("#2D1A1A"),
	CursorNormalBg: c("#434C5E"), CursorNormalFg: c("#ECEFF4"),
	CursorAddBg: c("#A3E4D7"), CursorAddFg: c("#1A251E"),
	CursorDelBg: c("#F5B7B1"), CursorDelFg: c("#2D1A1A"),
	TopBarBg: c("#2E3440"), TopBarFg: c("#D8DEE9"),
	StatusBarBg: c("#2E3440"), StatusBarFg: c("#D8DEE9"),
	StatusRepoFg: c("#7aa2f7"), StatusBranchFg: c("#bb9af7"),
	StatusAddFg: c("#A3BE8C"), StatusDelFg: c("#BF616A"), StatusDividerFg: c("#4C566A"),
	DirFg: c("99"), FileFg: c("252"),
	SelectedItemBg: c("#3B4252"), SelectedItemFg: c("#ECEFF4"),
	CmdLineBg: c("#3B4252"), CmdLineFg: c("#D8DEE9"),
}

var gruvbox = &ThemeColors{
	Name: "Gruvbox", ChromaStyle: "gruvbox", IsDark: true,
	Bg: c("#282828"), Fg: c("#ebdbb2"), DimFg: c("#665c54"),
	Border: c("#504945"), FocusedBorder: c("#83a598"),
	AddGutter: c("#b8bb26"), DelGutter: c("#fb4934"), CtxGutter: c("#665c54"),
	AddLineBg: c("#1d3022"), DelLineBg: c("#3c1f1e"),
	CursorNormalBg: c("#3c3836"), CursorNormalFg: c("#ebdbb2"),
	CursorAddBg: c("#b8bb26"), CursorAddFg: c("#282828"),
	CursorDelBg: c("#fb4934"), CursorDelFg: c("#282828"),
	TopBarBg: c("#1d2021"), TopBarFg: c("#ebdbb2"),
	StatusBarBg: c("#1d2021"), StatusBarFg: c("#ebdbb2"),
	StatusRepoFg: c("#83a598"), StatusBranchFg: c("#d3869b"),
	StatusAddFg: c("#b8bb26"), StatusDelFg: c("#fb4934"), StatusDividerFg: c("#504945"),
	DirFg: c("#83a598"), FileFg: c("#ebdbb2"),
	SelectedItemBg: c("#3c3836"), SelectedItemFg: c("#ebdbb2"),
	CmdLineBg: c("#3c3836"), CmdLineFg: c("#ebdbb2"),
}

var catppuccinMocha = &ThemeColors{
	Name: "Catppuccin Mocha", ChromaStyle: "catppuccin-mocha", IsDark: true,
	Bg: c("#1e1e2e"), Fg: c("#cdd6f4"), DimFg: c("#585b70"),
	Border: c("#45475a"), FocusedBorder: c("#89b4fa"),
	AddGutter: c("#a6e3a1"), DelGutter: c("#f38ba8"), CtxGutter: c("#585b70"),
	AddLineBg: c("#162818"), DelLineBg: c("#2e1520"),
	CursorNormalBg: c("#313244"), CursorNormalFg: c("#cdd6f4"),
	CursorAddBg: c("#a6e3a1"), CursorAddFg: c("#1e1e2e"),
	CursorDelBg: c("#f38ba8"), CursorDelFg: c("#1e1e2e"),
	TopBarBg: c("#181825"), TopBarFg: c("#cdd6f4"),
	StatusBarBg: c("#181825"), StatusBarFg: c("#cdd6f4"),
	StatusRepoFg: c("#89b4fa"), StatusBranchFg: c("#cba6f7"),
	StatusAddFg: c("#a6e3a1"), StatusDelFg: c("#f38ba8"), StatusDividerFg: c("#45475a"),
	DirFg: c("#89b4fa"), FileFg: c("#cdd6f4"),
	SelectedItemBg: c("#313244"), SelectedItemFg: c("#cdd6f4"),
	CmdLineBg: c("#313244"), CmdLineFg: c("#cdd6f4"),
}

var catppuccinLatte = &ThemeColors{
	Name: "Catppuccin Latte", ChromaStyle: "catppuccin-latte", IsDark: false,
	Bg: c("#eff1f5"), Fg: c("#4c4f69"), DimFg: c("#bcc0cc"),
	Border: c("#ccd0da"), FocusedBorder: c("#1e66f5"),
	AddGutter: c("#40a02b"), DelGutter: c("#d20f39"), CtxGutter: c("#9ca0b0"),
	AddLineBg: c("#d9efd3"), DelLineBg: c("#f5d6dc"),
	CursorNormalBg: c("#ccd0da"), CursorNormalFg: c("#4c4f69"),
	CursorAddBg: c("#40a02b"), CursorAddFg: c("#eff1f5"),
	CursorDelBg: c("#d20f39"), CursorDelFg: c("#eff1f5"),
	TopBarBg: c("#e6e9ef"), TopBarFg: c("#4c4f69"),
	StatusBarBg: c("#e6e9ef"), StatusBarFg: c("#4c4f69"),
	StatusRepoFg: c("#1e66f5"), StatusBranchFg: c("#8839ef"),
	StatusAddFg: c("#40a02b"), StatusDelFg: c("#d20f39"), StatusDividerFg: c("#ccd0da"),
	DirFg: c("#1e66f5"), FileFg: c("#4c4f69"),
	SelectedItemBg: c("#ccd0da"), SelectedItemFg: c("#4c4f69"),
	CmdLineBg: c("#ccd0da"), CmdLineFg: c("#4c4f69"),
}

var dracula = &ThemeColors{
	Name: "Dracula", ChromaStyle: "dracula", IsDark: true,
	Bg: c("#282a36"), Fg: c("#f8f8f2"), DimFg: c("#6272a4"),
	Border: c("#44475a"), FocusedBorder: c("#bd93f9"),
	AddGutter: c("#50fa7b"), DelGutter: c("#ff5555"), CtxGutter: c("#6272a4"),
	AddLineBg: c("#1a3320"), DelLineBg: c("#331c1c"),
	CursorNormalBg: c("#44475a"), CursorNormalFg: c("#f8f8f2"),
	CursorAddBg: c("#50fa7b"), CursorAddFg: c("#282a36"),
	CursorDelBg: c("#ff5555"), CursorDelFg: c("#282a36"),
	TopBarBg: c("#21222c"), TopBarFg: c("#f8f8f2"),
	StatusBarBg: c("#21222c"), StatusBarFg: c("#f8f8f2"),
	StatusRepoFg: c("#bd93f9"), StatusBranchFg: c("#ff79c6"),
	StatusAddFg: c("#50fa7b"), StatusDelFg: c("#ff5555"), StatusDividerFg: c("#44475a"),
	DirFg: c("#8be9fd"), FileFg: c("#f8f8f2"),
	SelectedItemBg: c("#44475a"), SelectedItemFg: c("#f8f8f2"),
	CmdLineBg: c("#44475a"), CmdLineFg: c("#f8f8f2"),
}

var monokai = &ThemeColors{
	Name: "Monokai", ChromaStyle: "monokai", IsDark: true,
	Bg: c("#272822"), Fg: c("#f8f8f2"), DimFg: c("#75715e"),
	Border: c("#49483e"), FocusedBorder: c("#a6e22e"),
	AddGutter: c("#a6e22e"), DelGutter: c("#f92672"), CtxGutter: c("#75715e"),
	AddLineBg: c("#1e2e1a"), DelLineBg: c("#3a1a1a"),
	CursorNormalBg: c("#3e3d32"), CursorNormalFg: c("#f8f8f2"),
	CursorAddBg: c("#a6e22e"), CursorAddFg: c("#272822"),
	CursorDelBg: c("#f92672"), CursorDelFg: c("#272822"),
	TopBarBg: c("#1e1f1a"), TopBarFg: c("#f8f8f2"),
	StatusBarBg: c("#1e1f1a"), StatusBarFg: c("#f8f8f2"),
	StatusRepoFg: c("#66d9ef"), StatusBranchFg: c("#ae81ff"),
	StatusAddFg: c("#a6e22e"), StatusDelFg: c("#f92672"), StatusDividerFg: c("#49483e"),
	DirFg: c("#66d9ef"), FileFg: c("#f8f8f2"),
	SelectedItemBg: c("#3e3d32"), SelectedItemFg: c("#f8f8f2"),
	CmdLineBg: c("#3e3d32"), CmdLineFg: c("#f8f8f2"),
}

var onedark = &ThemeColors{
	Name: "OneDark", ChromaStyle: "onedark", IsDark: true,
	Bg: c("#282c34"), Fg: c("#abb2bf"), DimFg: c("#5c6370"),
	Border: c("#3b4048"), FocusedBorder: c("#61afef"),
	AddGutter: c("#98c379"), DelGutter: c("#e06c75"), CtxGutter: c("#5c6370"),
	AddLineBg: c("#1c2a1e"), DelLineBg: c("#2d1c1e"),
	CursorNormalBg: c("#3b4048"), CursorNormalFg: c("#abb2bf"),
	CursorAddBg: c("#98c379"), CursorAddFg: c("#282c34"),
	CursorDelBg: c("#e06c75"), CursorDelFg: c("#282c34"),
	TopBarBg: c("#21252b"), TopBarFg: c("#abb2bf"),
	StatusBarBg: c("#21252b"), StatusBarFg: c("#abb2bf"),
	StatusRepoFg: c("#61afef"), StatusBranchFg: c("#c678dd"),
	StatusAddFg: c("#98c379"), StatusDelFg: c("#e06c75"), StatusDividerFg: c("#3b4048"),
	DirFg: c("#61afef"), FileFg: c("#abb2bf"),
	SelectedItemBg: c("#3b4048"), SelectedItemFg: c("#abb2bf"),
	CmdLineBg: c("#3b4048"), CmdLineFg: c("#abb2bf"),
}

var githubLight = &ThemeColors{
	Name: "GitHub Light", ChromaStyle: "github", IsDark: false,
	Bg: c("#ffffff"), Fg: c("#24292e"), DimFg: c("#959da5"),
	Border: c("#d1d5da"), FocusedBorder: c("#0366d6"),
	AddGutter: c("#28a745"), DelGutter: c("#cb2431"), CtxGutter: c("#959da5"),
	AddLineBg: c("#e6ffed"), DelLineBg: c("#ffeef0"),
	CursorNormalBg: c("#f6f8fa"), CursorNormalFg: c("#24292e"),
	CursorAddBg: c("#dcffe4"), CursorAddFg: c("#24292e"),
	CursorDelBg: c("#ffdce0"), CursorDelFg: c("#24292e"),
	TopBarBg: c("#f6f8fa"), TopBarFg: c("#24292e"),
	StatusBarBg: c("#f6f8fa"), StatusBarFg: c("#24292e"),
	StatusRepoFg: c("#0366d6"), StatusBranchFg: c("#6f42c1"),
	StatusAddFg: c("#28a745"), StatusDelFg: c("#cb2431"), StatusDividerFg: c("#d1d5da"),
	DirFg: c("#0366d6"), FileFg: c("#24292e"),
	SelectedItemBg: c("#f6f8fa"), SelectedItemFg: c("#24292e"),
	CmdLineBg: c("#f6f8fa"), CmdLineFg: c("#24292e"),
}

var githubDark = &ThemeColors{
	Name: "GitHub Dark", ChromaStyle: "github-dark", IsDark: true,
	Bg: c("#0d1117"), Fg: c("#c9d1d9"), DimFg: c("#484f58"),
	Border: c("#30363d"), FocusedBorder: c("#58a6ff"),
	AddGutter: c("#3fb950"), DelGutter: c("#f85149"), CtxGutter: c("#484f58"),
	AddLineBg: c("#0d2b15"), DelLineBg: c("#2d1214"),
	CursorNormalBg: c("#161b22"), CursorNormalFg: c("#c9d1d9"),
	CursorAddBg: c("#3fb950"), CursorAddFg: c("#0d1117"),
	CursorDelBg: c("#f85149"), CursorDelFg: c("#0d1117"),
	TopBarBg: c("#161b22"), TopBarFg: c("#c9d1d9"),
	StatusBarBg: c("#161b22"), StatusBarFg: c("#c9d1d9"),
	StatusRepoFg: c("#58a6ff"), StatusBranchFg: c("#bc8cff"),
	StatusAddFg: c("#3fb950"), StatusDelFg: c("#f85149"), StatusDividerFg: c("#30363d"),
	DirFg: c("#58a6ff"), FileFg: c("#c9d1d9"),
	SelectedItemBg: c("#161b22"), SelectedItemFg: c("#c9d1d9"),
	CmdLineBg: c("#161b22"), CmdLineFg: c("#c9d1d9"),
}

var rosePine = &ThemeColors{
	Name: "Rosé Pine", ChromaStyle: "rose-pine", IsDark: true,
	Bg: c("#191724"), Fg: c("#e0def4"), DimFg: c("#6e6a86"),
	Border: c("#26233a"), FocusedBorder: c("#c4a7e7"),
	AddGutter: c("#31748f"), DelGutter: c("#eb6f92"), CtxGutter: c("#6e6a86"),
	AddLineBg: c("#1a2733"), DelLineBg: c("#2d1a22"),
	CursorNormalBg: c("#26233a"), CursorNormalFg: c("#e0def4"),
	CursorAddBg: c("#31748f"), CursorAddFg: c("#191724"),
	CursorDelBg: c("#eb6f92"), CursorDelFg: c("#191724"),
	TopBarBg: c("#1f1d2e"), TopBarFg: c("#e0def4"),
	StatusBarBg: c("#1f1d2e"), StatusBarFg: c("#e0def4"),
	StatusRepoFg: c("#c4a7e7"), StatusBranchFg: c("#ebbcba"),
	StatusAddFg: c("#31748f"), StatusDelFg: c("#eb6f92"), StatusDividerFg: c("#26233a"),
	DirFg: c("#9ccfd8"), FileFg: c("#e0def4"),
	SelectedItemBg: c("#26233a"), SelectedItemFg: c("#e0def4"),
	CmdLineBg: c("#26233a"), CmdLineFg: c("#e0def4"),
}

var rosePineDawn = &ThemeColors{
	Name: "Rosé Pine Dawn", ChromaStyle: "rose-pine-dawn", IsDark: false,
	Bg: c("#faf4ed"), Fg: c("#575279"), DimFg: c("#9893a5"),
	Border: c("#dfdad9"), FocusedBorder: c("#907aa9"),
	AddGutter: c("#286983"), DelGutter: c("#b4637a"), CtxGutter: c("#9893a5"),
	AddLineBg: c("#e2eef3"), DelLineBg: c("#f4e4e8"),
	CursorNormalBg: c("#f2e9e1"), CursorNormalFg: c("#575279"),
	CursorAddBg: c("#286983"), CursorAddFg: c("#faf4ed"),
	CursorDelBg: c("#b4637a"), CursorDelFg: c("#faf4ed"),
	TopBarBg: c("#f2e9e1"), TopBarFg: c("#575279"),
	StatusBarBg: c("#f2e9e1"), StatusBarFg: c("#575279"),
	StatusRepoFg: c("#907aa9"), StatusBranchFg: c("#d7827e"),
	StatusAddFg: c("#286983"), StatusDelFg: c("#b4637a"), StatusDividerFg: c("#dfdad9"),
	DirFg: c("#56949f"), FileFg: c("#575279"),
	SelectedItemBg: c("#f2e9e1"), SelectedItemFg: c("#575279"),
	CmdLineBg: c("#f2e9e1"), CmdLineFg: c("#575279"),
}

var solarizedDark = &ThemeColors{
	Name: "Solarized Dark", ChromaStyle: "solarized-dark", IsDark: true,
	Bg: c("#002b36"), Fg: c("#839496"), DimFg: c("#586e75"),
	Border: c("#073642"), FocusedBorder: c("#268bd2"),
	AddGutter: c("#859900"), DelGutter: c("#dc322f"), CtxGutter: c("#586e75"),
	AddLineBg: c("#003e26"), DelLineBg: c("#3a1414"),
	CursorNormalBg: c("#073642"), CursorNormalFg: c("#93a1a1"),
	CursorAddBg: c("#859900"), CursorAddFg: c("#002b36"),
	CursorDelBg: c("#dc322f"), CursorDelFg: c("#002b36"),
	TopBarBg: c("#00212b"), TopBarFg: c("#839496"),
	StatusBarBg: c("#00212b"), StatusBarFg: c("#839496"),
	StatusRepoFg: c("#268bd2"), StatusBranchFg: c("#6c71c4"),
	StatusAddFg: c("#859900"), StatusDelFg: c("#dc322f"), StatusDividerFg: c("#073642"),
	DirFg: c("#268bd2"), FileFg: c("#839496"),
	SelectedItemBg: c("#073642"), SelectedItemFg: c("#93a1a1"),
	CmdLineBg: c("#073642"), CmdLineFg: c("#839496"),
}

var tokyonightNight = &ThemeColors{
	Name: "Tokyo Night", ChromaStyle: "tokyonight-night", IsDark: true,
	Bg: c("#1a1b26"), Fg: c("#c0caf5"), DimFg: c("#565f89"),
	Border: c("#292e42"), FocusedBorder: c("#7aa2f7"),
	AddGutter: c("#9ece6a"), DelGutter: c("#f7768e"), CtxGutter: c("#565f89"),
	AddLineBg: c("#1b2d1e"), DelLineBg: c("#2d1b22"),
	CursorNormalBg: c("#292e42"), CursorNormalFg: c("#c0caf5"),
	CursorAddBg: c("#9ece6a"), CursorAddFg: c("#1a1b26"),
	CursorDelBg: c("#f7768e"), CursorDelFg: c("#1a1b26"),
	TopBarBg: c("#16161e"), TopBarFg: c("#c0caf5"),
	StatusBarBg: c("#16161e"), StatusBarFg: c("#c0caf5"),
	StatusRepoFg: c("#7aa2f7"), StatusBranchFg: c("#bb9af7"),
	StatusAddFg: c("#9ece6a"), StatusDelFg: c("#f7768e"), StatusDividerFg: c("#292e42"),
	DirFg: c("#7dcfff"), FileFg: c("#c0caf5"),
	SelectedItemBg: c("#292e42"), SelectedItemFg: c("#c0caf5"),
	CmdLineBg: c("#292e42"), CmdLineFg: c("#c0caf5"),
}

var tokyonightStorm = &ThemeColors{
	Name: "Tokyo Night Storm", ChromaStyle: "tokyonight-storm", IsDark: true,
	Bg: c("#24283b"), Fg: c("#c0caf5"), DimFg: c("#565f89"),
	Border: c("#292e42"), FocusedBorder: c("#7aa2f7"),
	AddGutter: c("#9ece6a"), DelGutter: c("#f7768e"), CtxGutter: c("#565f89"),
	AddLineBg: c("#1e2e22"), DelLineBg: c("#2e1e24"),
	CursorNormalBg: c("#292e42"), CursorNormalFg: c("#c0caf5"),
	CursorAddBg: c("#9ece6a"), CursorAddFg: c("#24283b"),
	CursorDelBg: c("#f7768e"), CursorDelFg: c("#24283b"),
	TopBarBg: c("#1f2335"), TopBarFg: c("#c0caf5"),
	StatusBarBg: c("#1f2335"), StatusBarFg: c("#c0caf5"),
	StatusRepoFg: c("#7aa2f7"), StatusBranchFg: c("#bb9af7"),
	StatusAddFg: c("#9ece6a"), StatusDelFg: c("#f7768e"), StatusDividerFg: c("#292e42"),
	DirFg: c("#7dcfff"), FileFg: c("#c0caf5"),
	SelectedItemBg: c("#292e42"), SelectedItemFg: c("#c0caf5"),
	CmdLineBg: c("#292e42"), CmdLineFg: c("#c0caf5"),
}

var evergarden = &ThemeColors{
	Name: "Evergarden", ChromaStyle: "evergarden", IsDark: true,
	Bg: c("#1e2326"), Fg: c("#d3c6aa"), DimFg: c("#4c555b"),
	Border: c("#3a4248"), FocusedBorder: c("#a7c080"),
	AddGutter: c("#a7c080"), DelGutter: c("#e67e80"), CtxGutter: c("#4c555b"),
	AddLineBg: c("#1e2d1e"), DelLineBg: c("#2d1e20"),
	CursorNormalBg: c("#2e353b"), CursorNormalFg: c("#d3c6aa"),
	CursorAddBg: c("#a7c080"), CursorAddFg: c("#1e2326"),
	CursorDelBg: c("#e67e80"), CursorDelFg: c("#1e2326"),
	TopBarBg: c("#1a1f21"), TopBarFg: c("#d3c6aa"),
	StatusBarBg: c("#1a1f21"), StatusBarFg: c("#d3c6aa"),
	StatusRepoFg: c("#7fbbb3"), StatusBranchFg: c("#d699b6"),
	StatusAddFg: c("#a7c080"), StatusDelFg: c("#e67e80"), StatusDividerFg: c("#3a4248"),
	DirFg: c("#7fbbb3"), FileFg: c("#d3c6aa"),
	SelectedItemBg: c("#2e353b"), SelectedItemFg: c("#d3c6aa"),
	CmdLineBg: c("#2e353b"), CmdLineFg: c("#d3c6aa"),
}

var doomOne = &ThemeColors{
	Name: "Doom One", ChromaStyle: "doom-one", IsDark: true,
	Bg: c("#282c34"), Fg: c("#bbc2cf"), DimFg: c("#5c6370"),
	Border: c("#3b4048"), FocusedBorder: c("#51afef"),
	AddGutter: c("#98be65"), DelGutter: c("#ff6c6b"), CtxGutter: c("#5c6370"),
	AddLineBg: c("#1c2a1e"), DelLineBg: c("#2d1c1e"),
	CursorNormalBg: c("#3b4048"), CursorNormalFg: c("#bbc2cf"),
	CursorAddBg: c("#98be65"), CursorAddFg: c("#282c34"),
	CursorDelBg: c("#ff6c6b"), CursorDelFg: c("#282c34"),
	TopBarBg: c("#21242b"), TopBarFg: c("#bbc2cf"),
	StatusBarBg: c("#21242b"), StatusBarFg: c("#bbc2cf"),
	StatusRepoFg: c("#51afef"), StatusBranchFg: c("#c678dd"),
	StatusAddFg: c("#98be65"), StatusDelFg: c("#ff6c6b"), StatusDividerFg: c("#3b4048"),
	DirFg: c("#51afef"), FileFg: c("#bbc2cf"),
	SelectedItemBg: c("#3b4048"), SelectedItemFg: c("#bbc2cf"),
	CmdLineBg: c("#3b4048"), CmdLineFg: c("#bbc2cf"),
}

var quiet = &ThemeColors{
	Name: "Quiet", ChromaStyle: "native", IsDark: true,
	Bg: c("#000000"), Fg: c("#cccccc"), DimFg: c("#555555"),
	Border: c("#333333"), FocusedBorder: c("#6CBDF5"),
	AddGutter: c("#5EA980"), DelGutter: c("#A9555E"), CtxGutter: c("#555555"),
	AddLineBg: c("#0a1a0a"), DelLineBg: c("#1a0a0a"),
	CursorNormalBg: c("#1a1a1a"), CursorNormalFg: c("#cccccc"),
	CursorAddBg: c("#5EA980"), CursorAddFg: c("#000000"),
	CursorDelBg: c("#A9555E"), CursorDelFg: c("#000000"),
	TopBarBg: c("#080808"), TopBarFg: c("#cccccc"),
	StatusBarBg: c("#080808"), StatusBarFg: c("#cccccc"),
	StatusRepoFg: c("#6CBDF5"), StatusBranchFg: c("#9A779D"),
	StatusAddFg: c("#5EA980"), StatusDelFg: c("#A9555E"), StatusDividerFg: c("#333333"),
	DirFg: c("#548CB9"), FileFg: c("#cccccc"),
	SelectedItemBg: c("#1a1a1a"), SelectedItemFg: c("#ffffff"),
	CmdLineBg: c("#1a1a1a"), CmdLineFg: c("#cccccc"),
}
