<h1 align="center"><code>difi</code></h1>

<p align="center">
  <b>Review and refine Git diffs before you push.</b>
</p>

<p align="center">
  <b>git diff</b> shows changes. <b>difi</b> helps you <em>review</em> them.<br>
  Built in Go for instant startup, it turns raw diffs into a structured file tree with native <code>h j k l</code> navigation and a frictionless jump-to-editor workflow.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/Bubble_Tea-E2386F?style=for-the-badge&logo=tea&logoColor=white" />
  <img src="https://img.shields.io/github/license/xguot/difi?style=for-the-badge&color=2e3440" />
</p>

<p align="center">
  <img src= "https://github.com/user-attachments/assets/3695cfd2-148c-463d-9630-547d152adde0" alt="difi_demo" />
</p>

## Installation

#### Homebrew (macOS & Linux)

```bash
brew install difi
```

#### Go Install

```bash
go install github.com/xguot/difi/cmd/difi@latest
```

#### AUR (Arch Linux)

**Binary (pre-built):**

```bash
pikaur -S difi-bin
```

**Build from source:**

```bash
pikaur -S difi
```

#### Manual (Linux / Windows)

- Download the binary from Releases and add it to your `$PATH`.

## Workflow

Run difi in any Git repository against main:

```bash
cd my-project
difi
```

To compare against a specific branch or commit, just pass it as an argument:

```bash
# Compare against the main branch
difi main

# Compare against the previous commit
difi HEAD~1
```

**Piping & Alternative VCS**

- You can also pass raw diffs directly into `difi` via standard input. This is perfect for patch files or other version control systems like Jujutsu:

```bash
# Review a saved patch file
cat changes.patch | difi

# Review changes in Jujutsu (jj)
jj diff --git | difi

# Pipe standard git diff output
git diff | difi
```

## Controls

| Key           | Action                                       |
| ------------- | -------------------------------------------- |
| `Tab`         | Toggle focus between File Tree and Diff View |
| `j / k`       | Move cursor down / up                        |
| `h / l`       | Focus Left (Tree) / Focus Right (Diff)       |
| `e` / `Enter` | Edit file (opens editor at selected line)    |
| `?`           | Toggle help drawer                           |
| `q`           | Quit                                         |

## Configuration

`difi` can be configured using a YAML file located at `~/.config/difi/config.yaml`. If the file doesn't exist, `difi` will use sensible defaults.

### Example `config.yaml`

```yaml
editor: "nvim"

ui:
  line_numbers: "hybrid"
  theme: "default"
  diff_add_bg: "#2b3328" # Optional: Custom background for added lines
  diff_del_bg: "#4a2323" # Optional: Custom background for deleted lines
```

### Options

| Key               | Default                                       | Description                                              |
| :---------------- | :-------------------------------------------- | :------------------------------------------------------- |
| `editor`          | `$DIFI_EDITOR`, `$EDITOR`, `$VISUAL`, or `vi` | The editor to open when pressing `e` on a file.          |
| `ui.line_numbers` | `"hybrid"`                                    | The style of line numbers in the diff view.              |
| `ui.theme`        | `"default"`                                   | The core theme used for syntax highlighting.             |
| `ui.diff_add_bg`  | `""`                                          | Hex code or terminal color for added line backgrounds.   |
| `ui.diff_del_bg`  | `""`                                          | Hex code or terminal color for deleted line backgrounds. |

## Integrations

#### vim-fugitive

- **The "Unix philosophy" approach:** Uses the industry-standard Git wrapper to provide a robust, side-by-side editing experience.
- **Side-by-Side Editing:** Instantly opens a vertical split (:Gvdiffsplit!) against the index.
- **Merge Conflicts:** Automatically detects conflicts and opens a 3-way merge view for resolution.
- **Config**: Add the line below to if using **lazy.nvim**.

```lua
{
  "tpope/vim-fugitive",
  cmd = { "Gvdiffsplit", "Git" }, -- Add this line
}
```

<p align="left"> 
  <a href="https://github.com/tpope/vim-fugitive.git">
    <img src="https://img.shields.io/badge/Supports-vim--fugitive-4d4d4d?style=for-the-badge&logo=vim&logoColor=white" alt="Supports vim-fugitive" />
  </a>
</p>

#### difi.nvim

Get the ultimate review experience with **[difi.nvim](https://github.com/xguot/difi.nvim)**.

- **Auto-Open:** Instantly jumps to the file and line when you press `e` in the CLI.
- **Visual Diff:** Renders diffs inline with familiar green/red highlights—just like reviewing a PR on GitHub.
- **Interactive Review:** Restore a "deleted" line by simply removing the `-` marker. Discard an added line by deleting it entirely.
- **Context Aware:** Automatically syncs with your `difi` session target.

<p align="left">
  <a href="https://github.com/xguot/difi.nvim">
    <img src="https://img.shields.io/badge/Get_difi.nvim-57A143?style=for-the-badge&logo=neovim&logoColor=white" alt="Get difi.nvim" />
  </a>
</p>

## Git Integration

To use `difi` as a native git command (e.g., `git difi`), add it as an alias in your global git config:

```bash
git config --global alias.difi '!difi'
```

Now you can run it directly from git:

```bash
git difi
```

## Contributing

```bash
git clone https://github.com/xguot/difi
cd difi
go run cmd/difi/main.go
```

Contributions are especially welcome in:

- diff.nvim rendering edge cases
- UI polish and accessibility
- Windows support

## Star History

<a href="https://star-history.com/#xguot/difi&Date">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=xguot/difi&type=Date&theme=dark" />
      <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=xguot/difi&type=Date" />
      <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=xguot/difi&type=Date" />
    </picture>
  </a>
</div>
