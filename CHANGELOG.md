# Changelog

All notable changes to difi. Versions are listed in reverse chronological order.

Each entry includes the tag, date, and a summary of changes. Versions that were
retracted (tags deleted or never formally published) are marked accordingly.

---

## [v0.2.31] — 2026-07-22

- **Fix Homebrew upgrade path for users stuck on ghost version 0.2.30**
- Make git tag the single source of truth for version (source always `"dev"`)
- Inject version at build time via `git describe --tags` in Makefile
- Add CI guard: block release if source version is hardcoded
- Add CI guard: block release on semver mismatch or duplicate tag
- Add CI guard: verify version injection on every PR
- Add CHANGELOG.md with full release history and maintainer docs
- Document release process and branch protection rules

## [v0.2.12] — 2026-07-17

- Fix ActiveTheme not respecting configured theme at startup (PR #52)
- Bump embedded version string to 0.2.12

## [v0.2.11] — 2026-07-08

- Add man page and bump embedded version

> **Note:** v0.2.9 and v0.2.11 point to the same commit (0b9699d). Tags v0.2.10
> and v0.2.12–v0.2.29 were never published. See the [retracted versions](#retracted-versions)
> section below.

## [v0.2.9] — 2026-06-27

- Add man page and bump embedded version

## [v0.2.8] — 2026-06-25

- Fix tag re-point checksum conflict (internal release hygiene)

## [v0.2.7] — 2026-06-25

- Add vim-style command line with tab completion and `ZZ` quit
- Add pluggable theme engine with 17 vim colorscheme presets
- Add vim commands: `:help`, `:noh`, `:w`, `:set`, `:<num>`, `:$`
- Add full help buffer with vimdoc-style rendering
- Add hjkl navigation in help buffer
- Redesign empty state and help drawer in vim style
- Document command line, themes, and controls in README

> **Note:** v0.2.7 was tagged but no GitHub release was created. It was
> immediately superseded by v0.2.8.

## [v0.2.6] — 2026-05-09

- Add `-v`/`-p`/`-f` shorthand flags (PR #47)

## [v0.2.5] — 2026-05-09

- Fix release workflow tokens

> **Note:** v0.2.4 and v0.2.5 point to the same commit (ed0f580). Both tags
> exist for historical reasons.

## [v0.2.4] — 2026-05-09

- Fix release workflow tokens

## [v0.2.3] — 2026-05-08

- Add flat navigation mode (`-f` / `--flat`) (PR #45)
- Add logo and wordmark to README
- Refine piping section in README

> **Note:** v0.2.3 was tagged but no GitHub release was created.

## [v0.2.2] — 2026-03-30

- Fix editor line jump offsets and cursor snapping
- Fix status bar background color erase in tmux
- Fix demo.gif in README

## [v0.2.1] — 2026-03-27

- Prevent diff highlight color from bleeding into file tree
- Fix help page not showing in empty state
- Clean up README

## [v0.2.0] — 2026-03-27

- Initial support for refined highlighting in diff views
- Clean up noise comments

## [v0.1.10] — 2026-03-27

- Refine highlight logic for diff views
- Refactor UI with update.go and view.go
- Modernize goreleaser config to v2 syntax
- Update links for new GitHub username

## [v0.1.9] — 2026-03-22

- Fix CalculateFileLine off-by-one and header misparse (PR #30)
- Include untracked files in diff list (PR #32)

## [v0.1.8] — 2026-02-22

- Add Mercurial (hg) support with VCS abstraction layer
- Allow custom editor override in config.yaml
- Fix sidebar tree item overflow clipping

## [v0.1.7] — 2026-02-16

- Support reading diffs from stdin (pipe support)
- Adjust installation instructions for Homebrew core

## [v0.1.6] — 2026-02-12

- Fix long line rendering (reserve 2 lines for view)

## [v0.1.5] — 2026-02-08

- Refine diff view with clean headers

## [v0.1.4] — 2026-02-07

- Add `--plain` flag for CI/headless support
- Fix version embedding in GoReleaser builds
- Update go.mod/go.sum

## [v0.1.3] — 2026-02-07

- Add `--plain` non-interactive mode

## [v0.1.2] — 2026-02-06

- Add integration with vim-fugitive

## [v0.1.1] — 2026-02-04

- Fix long line truncation
- Add AUR install instructions
- Default diff target to HEAD

## [v0.1.0] — 2026-02-01

- Refactor README for clarity and new features

## [v0.0.4] — 2026-02-01

- Add Homebrew version support
- Fix text color rendering

## [v0.0.3] — 2026-02-01

- Revise README for clarity

## [v0.0.2] — 2026-01-31

- Add version flag (`--version`)
- Fix text color in dark terminals

## [v0.0.1] — 2026-01-31

- Initial release: pixel-perfect terminal diff viewer
- Vim motions support
- YAML config support
- Dynamic target selection
- Empty state display

---

## Retracted Versions

### v0.2.30 (ghost version) — 2026-07-06

On 2026-07-06, two commits landed on `main` that set the embedded version string
to `0.2.30`:

- `99024b8` Add legal-pad theme and bump to 0.2.30
- `80e9c66` Harden light theme support with theme-aware file tree and pane backgrounds

These commits (and several follow-ups) were later removed from `main` via
`git reset --hard` back to `0b9699d` (the v0.2.9 commit). No tag `v0.2.30` was
ever created, and the commits are now orphaned.

**Impact:** Anyone who installed difi during the ~24-hour window when these
commits were on `main` (via `go install`, `brew install --HEAD`, or a
Homebrew formula bump) has a binary that reports version `0.2.30`. Homebrew
refuses to upgrade because `0.2.30 > 0.2.12`.

**Resolution:** Release v0.2.31 skips past the ghost version so `brew upgrade`
works automatically. Affected users don't need to reinstall manually.

### Missing Tags

The following version gaps exist intentionally:

- **v0.2.10**: Never created — v0.2.11 was tagged directly from the v0.2.9
  commit to avoid a tag re-point conflict.
- **v0.2.12–v0.2.29**: Never existed. The jump from v0.2.11 to v0.2.30 (and
  back) was a local version-string increment that was never formalized as tags.

---

## Fixing a Homebrew Version Mismatch

If `brew info difi` shows a formula version lower than your installed version
(e.g., formula says `0.2.12` but installed says `0.2.30`), Homebrew skips
upgrades. To force a reinstall:

```bash
# Force reinstall the current formula version (downgrades if needed)
brew reinstall --force difi

# Verify the version matches
difi --version
```

If the above doesn't work because Homebrew's receipt still records the ghost
version, uninstall first:

```bash
brew uninstall --ignore-dependencies difi
brew install difi
difi --version
```

This is safe — difi has no persistent state outside its binary and config file
(`~/.config/difi/config.yaml`), which is preserved across reinstalls.

---

## Release Process (for maintainers)

### Version model

The git tag is the **single source of truth** for the version. The embedded
string in `cmd/difi/main.go` is always `"dev"` — it is overwritten at build
time via `-ldflags "-X main.version=<tag>"`.

| Build method       | Version source              |
| ------------------ | --------------------------- |
| `make build`       | `git describe --tags`       |
| GoReleaser (CI)    | The pushed git tag          |
| `go install`       | Falls back to `"dev"`       |

This means you never edit the version string in source code. You only create
a git tag.

### Cutting a release

1. Ensure all changes for the release are merged to `main`.
2. Verify the CI is green on `main`.
3. Tag the release:
   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```
4. CI runs `.github/workflows/release.yml`, which:
   - Verifies the source version is `"dev"` (guard)
   - Validates the tag follows semver `vX.Y.Z` (guard)
   - Checks the tag has never been released before (guard)
   - Builds binaries, creates a GitHub Release, and updates the Homebrew tap
5. Update `CHANGELOG.md` with the new entry.

### Rules

- **Never `git reset --hard` on `main`.** Use `git revert` to undo changes
  that have been pushed. Force-pushing to `main` orphans commits that users
  may have already installed.
- **Never force-push over an existing tag.** Tags are immutable. If a release
  is broken, cut a new patch version (e.g., `v0.2.13`) instead.
- **Never hardcode a version in `main.go`.** PR CI enforces that the version
  string stays `"dev"`.
- **Protect `main` from force pushes** in GitHub repository settings:
  Settings → Branches → Add rule → Branch name pattern: `main`, enable
  "Lock branch" and restrict force pushes.
