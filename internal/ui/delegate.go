package ui

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/xguot/difi/internal/config"
	"github.com/xguot/difi/internal/tree"
)

type TreeDelegate struct {
	Config  config.Config
	Focused bool
}

func (d TreeDelegate) Height() int  { return 1 }
func (d TreeDelegate) Spacing() int { return 0 }

func (d TreeDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			i, ok := m.SelectedItem().(tree.TreeItem)
			if !ok {
				return nil
			}

			c := exec.Command(d.Config.Editor, i.Title())
			return tea.ExecProcess(c, func(err error) tea.Msg {
				return nil
			})
		}
	}
	return nil
}

func (d TreeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(tree.TreeItem)
	if !ok {
		return
	}

	title := i.Title()
	maxWidth := m.Width() - 2
	if maxWidth < 4 {
		maxWidth = 4
	}
	title = ansi.Truncate(title, maxWidth, "…")

	if index == m.Index() {
		style := lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Width(maxWidth)

		if !d.Focused {
			style = style.Foreground(lipgloss.Color("245"))
		}

		fmt.Fprint(w, style.Render(title))
	} else {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(maxWidth)
		fmt.Fprint(w, style.Render(title))
	}
}
