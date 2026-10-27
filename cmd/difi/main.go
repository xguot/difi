package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/review"
	"github.com/xguot/difi/internal/ui"
	"github.com/xguot/difi/internal/vcs"
)

// version is set at build time via ldflags.
// Release binaries inject the git tag via -X main.version.
// Local builds report "dev" unless overridden.
var version = "dev"

type cliOptions struct {
	showVersion, plain, flat      bool
	staged, unstaged, noUntracked bool
	forceVCS, target              string
	targetProvided                bool
}

func parseArgs(args []string, output io.Writer) (cliOptions, error) {
	var options cliOptions
	flags := flag.NewFlagSet("difi", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.BoolVar(&options.showVersion, "version", false, "Show version")
	flags.BoolVar(&options.showVersion, "v", false, "Show version (shorthand)")
	flags.BoolVar(&options.plain, "plain", false, "Print a plain list of changed files")
	flags.BoolVar(&options.plain, "p", false, "Print a plain list (shorthand)")
	flags.BoolVar(&options.flat, "flat", false, "Use one-line file navigation")
	flags.BoolVar(&options.flat, "f", false, "Use one-line file navigation (shorthand)")
	flags.BoolVar(&options.staged, "staged", false, "Review the Git index against HEAD or target")
	flags.BoolVar(&options.staged, "s", false, "Review staged changes (shorthand)")
	flags.BoolVar(&options.unstaged, "unstaged", false, "Review the Git working tree against the index")
	flags.BoolVar(&options.unstaged, "u", false, "Review unstaged changes (shorthand)")
	flags.BoolVar(&options.noUntracked, "no-untracked", false, "Exclude untracked files")
	flags.StringVar(&options.forceVCS, "vcs", "", "Force VCS backend (git or hg)")
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: difi [options] [target]")
		fmt.Fprintln(output, "       difi [--plain] < changes.patch")
		fmt.Fprintln(output, "\nOptions:")
		fmt.Fprintln(output, "  -v, --version       Show version")
		fmt.Fprintln(output, "  -p, --plain         Print changed file paths and exit")
		fmt.Fprintln(output, "  -f, --flat          Use flat file navigation")
		fmt.Fprintln(output, "  -s, --staged        Review index vs HEAD or target (Git)")
		fmt.Fprintln(output, "  -u, --unstaged      Review working tree vs index (Git)")
		fmt.Fprintln(output, "      --no-untracked Exclude untracked files")
		fmt.Fprintln(output, "      --vcs git|hg   Force a VCS backend")
		fmt.Fprintln(output, "\nTarget defaults to HEAD (tip for Mercurial). --unstaged takes no target.")
	}
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() > 1 {
		return options, errors.New("expected at most one target")
	}
	if flags.NArg() == 1 {
		options.target, options.targetProvided = flags.Arg(0), true
		if strings.HasPrefix(options.target, "-") {
			return options, errors.New("target must be a branch, commit, or tag, not an option")
		}
	}
	return options, nil
}

func (o cliOptions) validate(piped, mercurial bool) error {
	if o.staged && o.unstaged {
		return errors.New("--staged and --unstaged cannot be combined")
	}
	if o.unstaged && o.targetProvided {
		return errors.New("--unstaged compares against the index and does not accept a target")
	}
	if mercurial && (o.staged || o.unstaged) {
		return errors.New("--staged and --unstaged require Git; Mercurial has no index")
	}
	if piped && (o.staged || o.unstaged || o.noUntracked || o.targetProvided) {
		return errors.New("piped patches already define their comparison; do not combine stdin with a target or review scope flags")
	}
	return nil
}

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	options, err := parseArgs(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 2
	}
	if options.showVersion {
		fmt.Fprintf(stdout, "difi version %s\n", version)
		return 0
	}

	var client vcs.VCS
	switch options.forceVCS {
	case "":
		client = vcs.DetectVCS()
	case "git":
		client = vcs.GitVCS{}
	case "hg":
		client = vcs.HgVCS{}
	default:
		fmt.Fprintln(stderr, "Error: --vcs must be git or hg")
		return 2
	}
	stat, err := stdin.Stat()
	if err != nil {
		fmt.Fprintln(stderr, "Error reading stdin:", err)
		return 1
	}
	piped := stat.Mode()&os.ModeCharDevice == 0
	_, mercurial := client.(vcs.HgVCS)
	if err := options.validate(piped, mercurial); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 2
	}
	var input string
	if piped {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintln(stderr, "Error reading stdin:", err)
			return 1
		}
		input = string(data)
	}

	reviewOptions := review.Options{NoUntracked: options.noUntracked}
	if options.staged {
		reviewOptions.Scope = review.Staged
	}
	if options.unstaged {
		reviewOptions.Scope = review.Unstaged
	}
	switch c := client.(type) {
	case vcs.GitVCS:
		c.Options = reviewOptions
		client = c
	case vcs.HgVCS:
		c.Options = reviewOptions
		client = c
	}
	target := options.target
	if !options.targetProvided {
		target = "HEAD"
		if mercurial {
			target = "tip"
		}
	}
	if options.plain {
		var files []string
		if piped {
			files = client.ParseFilesFromDiff(input)
		} else {
			files, err = client.ListChangedFiles(target)
		}
		if err != nil {
			fmt.Fprintln(stderr, "Error listing changed files:", err)
			return 1
		}
		for _, file := range files {
			fmt.Fprintln(stdout, file)
		}
		return 0
	}

	cfg := config.Load()
	programOptions := []tea.ProgramOption{tea.WithAltScreen(), tea.WithInput(stdin), tea.WithOutput(stdout)}
	if piped {
		tty, err := os.Open("/dev/tty")
		if err == nil {
			defer tty.Close()
			programOptions = append(programOptions, tea.WithInput(tty))
		}
	}
	model := ui.NewModel(cfg, target, input, client, options.flat, version, ui.Options{Review: reviewOptions, FromStdin: piped})
	if _, err := tea.NewProgram(model, programOptions...).Run(); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}
