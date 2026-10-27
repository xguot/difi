package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xguot/difi/internal/tree"
)

type sessionMsg struct {
	Request uint64
	Files   []string
	Stats   StatsMsg
	Err     error
}

func (m *Model) refreshSession() tea.Cmd {
	m.refreshRequest++
	// Invalidate work that started before this refresh, even if the selected
	// file remains the same after the repository has been reloaded.
	m.diffRequest++
	snapshot := *m
	return func() tea.Msg {
		var files []string
		var err error
		var stats StatsMsg
		if snapshot.fromStdin {
			files = snapshot.vcs.ParseFilesFromDiff(snapshot.pipedDiff)
			stats = snapshot.computePipedStatsCmd()().(StatsMsg)
		} else {
			files, err = snapshot.vcs.ListChangedFiles(snapshot.targetBranch)
			if err == nil {
				stats = snapshot.fetchStatsCmd(snapshot.targetBranch)().(StatsMsg)
				err = stats.Err
			}
		}
		return sessionMsg{Request: snapshot.refreshRequest, Files: files, Stats: stats, Err: err}
	}
}

func (m *Model) replaceFiles(files []string) {
	previous := m.selectedPath
	next := tree.New(files)
	// Preserve expansion state for directories that survive the refresh.
	states := make(map[string]bool)
	var collect func(*tree.Node)
	collect = func(node *tree.Node) {
		states[node.FullPath] = node.Expanded
		for _, child := range node.Children {
			collect(child)
		}
	}
	collect(m.treeState.Root)
	var preserve func(*tree.Node)
	preserve = func(node *tree.Node) {
		if expanded, ok := states[node.FullPath]; ok {
			node.Expanded = expanded
		}
		for _, child := range node.Children {
			preserve(child)
		}
	}
	preserve(next.Root)
	m.treeState = next
	m.rebuildFileList(previous)
}
