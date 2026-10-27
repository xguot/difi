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

*-s, --staged*
	Review the Git index against HEAD, or against the supplied target.
	Untracked files are excluded. Works before the first commit.

*-u, --unstaged*
	Review Git working files against the index. Takes no target.
	Untracked files are included unless *--no-untracked* is set.

*--no-untracked*
	Exclude untracked files. Supported by Git and Mercurial.

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

## Review Scope

The default comparison remains working files against HEAD, including untracked
files. *--staged* and *--unstaged* are Git-only and cannot be combined. Put flags
before the optional target. For example:

```
difi --staged main
difi --unstaged --no-untracked
difi --plain --staged
```

File lists, diffs, statistics, and refreshes use the selected scope. Editors
always open working files; editing in staged mode does not stage the result.
Difi does not stage or discard changes.

A piped patch defines its own comparison and cannot be combined with a target
or scope flags. *--plain* also accepts piped patches. Empty patches stay empty;
refresh uses the supplied patch in memory.

## Panes

The screen has two panes: the file tree (left) and the diff view (right).
Press _Tab_ to toggle focus between them, or _h_ and _l_ to focus the
left or right pane directly. Terminals narrower than 60 columns show only
the focused pane.

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
|  _/_
:  Filter paths in the tree or search text in the diff
|  _n_, _N_
:  Next / previous matching diff line, wrapping at the ends
|  _]c_, _[c_, _}_, _{_
:  Next / previous hunk; stops at the first and last hunk
|  _Esc_
:  Clear the focused pane's filter or search
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

## Filtering and Search

Press _/_ in the tree to filter full file paths. Matching is literal and case
insensitive, including files in collapsed directories. Results show full paths.
Press _Enter_ to keep the filter, or _Esc_ during input to restore the previous
filter and selection. _Esc_ in the tree clears an applied filter. Filters survive
refreshes.

Press _/_ in the diff to search content lines. Matching is literal and case
insensitive; hunk headers are excluded. Matching lines are underlined and marked
with a diamond. The footer counts matching lines. _n_ and _N_ wrap forward and
backward. _Esc_ cancels input or clears search; _:noh_ also clears it. _Ctrl-u_
clears prompt input.

Hunk navigation (_]c_ / _[c_, also _}_ / _{_) moves to the first changed line of
each hunk. Vim counts are supported, such as _3]c_ or _5n_.

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
:  Refresh files, statistics, and diff from disk
|  _:help_, _:h_
:  Open the full help buffer
|  _:noh_, _:nohlsearch_
:  Clear search and visual selection
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

Git editor sessions also receive:

_DIFI_SCOPE_
	_working_, _staged_, or _unstaged_.

_DIFI_TARGET_
	The comparison target. Empty in unstaged mode, whose baseline is the index.

Staged line positions are mapped through unstaged edits before opening the
working file.

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
