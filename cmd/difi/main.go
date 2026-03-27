package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/ui"
	"github.com/xguot/difi/internal/vcs"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	plain := flag.Bool("plain", false, "Print a plain summary")
	forceVCS := flag.String("vcs", "", "Force specific VCS (git or hg)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("difi version %s\n", version)
		os.Exit(0)
	}

	var pipedDiff string
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		b, _ := io.ReadAll(os.Stdin)
		pipedDiff = string(b)
	}

	// Detect or force VCS type
	var vcsClient vcs.VCS
	if *forceVCS != "" {
		switch *forceVCS {
		case "git":
			vcsClient = vcs.GitVCS{}
		case "hg":
			vcsClient = vcs.HgVCS{}
		default:
			fmt.Fprintf(os.Stderr, "Error: unsupported VCS '%s'. Supported values: git, hg\n", *forceVCS)
			os.Exit(1)
		}
	} else {
		vcsClient = vcs.DetectVCS()
	}

	target := "HEAD"
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}

	// For Mercurial, use "tip" as default instead of "HEAD"
	if _, isHg := vcsClient.(vcs.HgVCS); isHg && target == "HEAD" {
		target = "tip"
	}

	if *plain && pipedDiff == "" {
		// Use VCS-specific commands for plain output
		files, err := vcsClient.ListChangedFiles(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing changed files: %v\n", err)
			os.Exit(1)
		}
		for _, file := range files {
			fmt.Println(file)
		}
		os.Exit(0)
	}

	cfg := config.Load()

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if pipedDiff != "" {
		if tty, err := os.Open("/dev/tty"); err == nil {
			opts = append(opts, tea.WithInput(tty))
		}
	}

	p := tea.NewProgram(ui.NewModel(cfg, target, pipedDiff, vcsClient), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
