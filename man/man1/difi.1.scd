difi(1)

# NAME

difi - review and refine Git diffs before you push

# SYNOPSIS

*difi* [options] [_target_]

*difi* < _patch_
*git diff* | *difi*
*jj diff --git* | *difi*

# DESCRIPTION

*difi* is a terminal UI for reviewing diffs from Git, Mercurial, or raw
patch files. It opens a two-pane interface: a file tree on the left and a
syntax-highlighted diff view on the right. Navigation, commands, and
configuration follow vim conventions.

Run *difi* in any repository to review uncommitted changes against a
target branch, commit, or tag. You can also pipe raw diff output directly
into *difi* from *git diff*, *jj diff --git*, or a patch file.

# OPTIONS

*-v, --version*
	Show version and exit.

*-p, --plain*
	Print a plain list of changed files and exit (no TUI).

*-f, --flat*
	Use one-line file navigation in the tree pane instead of
	a collapsible directory tree.

*--vcs* _string_
	Force a specific VCS backend. Supported values: _git_, _hg_.
	By default, *difi* auto-detects the VCS from the current
	directory.

# USAGE

## Invocation

*difi*
	Diff against HEAD (or tip for Mercurial). This is the most
	common invocation.

*difi* _main_
	Diff against the _main_ branch.

*difi* _HEAD~3_
	Diff against 3 commits ago.

*difi* _< changes.patch_
	Review a saved patch file from stdin.

*git diff | difi*
	Pipe raw Git diff output into *difi*.

*jj diff --git | difi*
	Review Jujutsu changes through *difi*.

## Panes

The screen has two panes: the file tree (left) and the diff view (right).
Press _Tab_ to toggle focus between them, or _h_ and _l_ to focus the
left or right pane directly.

# KEY BINDINGS

## Normal Mode

[[ *Key*
:- *Action*
|  _j_, _k_, _Down_, _Up_
:  Move cursor down / up one line
|  _gg_
:  Jump to the first line of the diff
|  _G_
:  Jump to the last line of the diff
|  _H_, _M_, _L_
:  Cursor to top / middle / bottom of screen
|  _Ctrl-d_, _Ctrl-u_
:  Scroll down / up half a page
|  _zz_, _zt_, _zb_
:  Center cursor / top / bottom scroll
|  _e_, _Enter_
:  Edit file at the cursor line
|  _s_
:  Toggle split / unified diff view
|  _V_
:  Toggle visual selection mode
|  _f_
:  Toggle flat tree mode
|  _Tab_
:  Toggle pane focus (tree / diff)
|  _h_, _Left_
:  Focus left pane (file tree)
|  _l_, _Right_
:  Focus right pane (diff view)
|  _?_
:  Toggle quick help drawer
|  _:_
:  Open vim-style command line
|  _q_, _ZZ_
:  Quit

## File Tree Pane

[[ *Key*
:- *Action*
|  _Enter_ on a directory
:  Expand or collapse the directory
|  _Enter_ on a file
:  Edit the file (same as _e_)
|  _j_, _k_
:  Move selection up / down
|  _f_
:  Toggle flat mode (show full file paths, no tree)

## Visual Mode

[[ *Key*
:- *Action*
|  _V_
:  Toggle visual selection mode
|  _j_, _k_
:  Extend selection down / up
|  _Esc_
:  Exit visual mode

The selected range is highlighted with the cursor line background color.

# COMMAND LINE

Press _:_ to open a vim-style command line at the bottom of the screen.
Tab cycles through matching command names, theme names, and _:set_ option
values. Press _Esc_ to cancel.

## Commands

[[ *Command*
:- *Action*
|  _:q_, _:quit_
:  Exit difi
|  _:w_, _:write_
:  Refresh the diff view from disk
|  _:help_, _:h_
:  Open the full help buffer
|  _:noh_, _:nohlsearch_
:  Clear visual selection highlight
|  _:<number>_
:  Jump to line <number> in the diff
|  _:$_
:  Jump to the last line
|  _:colorscheme_ _<name>_
:  Switch the color theme
|  _:theme_ _<name>_
:  Alias for _:colorscheme_
|  _:set_ _<option>_
:  Change a setting live

## :set Options

[[ *Option*
:- *Effect*
|  _number_, _nu_
:  Absolute line numbers
|  _nonumber_, _nonu_
:  Hide line numbers
|  _relativenumber_, _rnu_
:  Relative line numbers
|  _norelativenumber_
:  Disable relative line numbers
|  _hybrid_
:  Hybrid mode (absolute on cursor line)
|  _relative_
:  Relative mode
|  _absolute_
:  Absolute mode
|  _hidden_
:  No line numbers
|  _split_
:  Side-by-side diff layout
|  _unified_
:  Unified diff layout (default)

# EDITING

Pressing _e_ or _Enter_ on a file opens your editor at the exact line
corresponding to the cursor position in the diff. The editor is
determined by, in order:

. _DIFI_EDITOR_ environment variable
. _EDITOR_ environment variable
. _VISUAL_ environment variable
. _vi_ (fallback)

# CONFIGURATION

*difi* reads its configuration from _~/.config/difi/config.yaml_. If the
file does not exist, sensible defaults are used.

## Options

[[ *Key*
:- *Default*
:- *Description*
|  _editor_
:  _$DIFI_EDITOR_, _$EDITOR_, _$VISUAL_, or _vi_
:  Editor command for opening files
|  _ui.theme_
:  _nord_
:  Vim colorscheme for syntax highlighting and UI
|  _ui.line_numbers_
:  _hybrid_
:  Line number style (_hybrid_, _relative_, _absolute_, _hidden_)
|  _ui.diff_mode_
:  _unified_
:  Diff layout (_unified_ or _split_)
|  _ui.diff_add_bg_
:  _""_
:  Hex color for added line backgrounds
|  _ui.diff_del_bg_
:  _""_
:  Hex color for deleted line backgrounds

## Example

```
editor: "nvim"

ui:
  line_numbers: "hybrid"
  theme: "nord"
  diff_add_bg: "#2b3328"
  diff_del_bg: "#4a2323"
```

# THEMES

*difi* ships with 17 built-in themes ported from popular vim
colorschemes. Set the default in _~/.config/difi/config.yaml_ or switch
live with _:colorscheme <name>_.

[[ *Theme*
:< *Light/Dark*
|  _nord_ (default)
:  dark
|  _gruvbox_
:  dark
|  _catppuccin-mocha_
:  dark
|  _catppuccin-latte_
:  light
|  _dracula_
:  dark
|  _monokai_
:  dark
|  _onedark_
:  dark
|  _github_
:  light
|  _github-dark_
:  dark
|  _rose-pine_
:  dark
|  _rose-pine-dawn_
:  light
|  _solarized-dark_
:  dark
|  _tokyonight-night_
:  dark
|  _tokyonight-storm_
:  dark
|  _evergarden_
:  dark
|  _doom-one_
:  dark
|  _quiet_
:  dark

# FILES

_~/.config/difi/config.yaml_
	User configuration file.

# ENVIRONMENT

_DIFI_EDITOR_
	Preferred editor (highest priority).

_EDITOR_, _VISUAL_
	Fallback editor, checked in order after _DIFI_EDITOR_.

# INTEGRATIONS

## Git alias

To use *difi* as a native git command:

```
git config --global alias.difi '!difi'
```

Then run _git difi_ directly from any repository.

## vim-fugitive

*difi* integrates with vim-fugitive for a side-by-side diff editing
experience. See the project README for configuration details.

## difi.nvim

The companion Neovim plugin *difi.nvim* provides auto-open, inline
visual diffs, and interactive review. See
_https://github.com/xguot/difi.nvim_.

# SEE ALSO

*difi.nvim*(1), *git*(1), *hg*(1), *jj*(1)

Homepage: _https://github.com/xguot/difi_
Website: _https://difi.vercel.app_

# AUTHORS

Xiyuan Guo and contributors. Open source under the MIT license.
