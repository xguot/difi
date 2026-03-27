package tree

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

// FileTree holds the state of the entire file graph.
type FileTree struct {
	Root *Node
}

// Node represents a file or directory in the tree.
type Node struct {
	Name     string
	FullPath string
	IsDir    bool
	Children map[string]*Node
	Expanded bool
	Depth    int
}

// TreeItem represents a file or folder for the Bubble Tea list.
type TreeItem struct {
	Name     string
	FullPath string
	IsDir    bool
	Depth    int
	Expanded bool
	Icon     string
}

func (i TreeItem) FilterValue() string { return i.Name }
func (i TreeItem) Description() string { return "" }
func (i TreeItem) Title() string {
	indent := strings.Repeat("  ", i.Depth)
	disclosure := " "
	if i.IsDir {
		if i.Expanded {
			disclosure = "▾"
		} else {
			disclosure = "▸"
		}
	}
	return fmt.Sprintf("%s%s %s %s", indent, disclosure, i.Icon, i.Name)
}

// New creates a new FileTree from a list of changed file paths.
func New(paths []string) *FileTree {
	root := &Node{
		Name:     "root",
		IsDir:    true,
		Children: make(map[string]*Node),
		Expanded: true,
		Depth:    -1,
	}

	for _, path := range paths {
		addPath(root, path)
	}

	return &FileTree{Root: root}
}

func addPath(root *Node, path string) {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(cleanPath, "/")

	current := root
	for i, name := range parts {
		if _, exists := current.Children[name]; !exists {
			isFile := i == len(parts)-1
			nodePath := name
			if current.FullPath != "" {
				nodePath = current.FullPath + "/" + name
			}

			current.Children[name] = &Node{
				Name:     name,
				FullPath: nodePath,
				IsDir:    !isFile,
				Children: make(map[string]*Node),
				Expanded: true,
				Depth:    current.Depth + 1,
			}
		}
		current = current.Children[name]
	}
}

// Items returns the flattened, visible list items based on expansion state.
func (t *FileTree) Items() []list.Item {
	var items []list.Item
	flatten(t.Root, &items)
	return items
}

func flatten(node *Node, items *[]list.Item) {
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir != children[j].IsDir {
			return children[i].IsDir
		}
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})

	for _, child := range children {
		*items = append(*items, TreeItem{
			Name:     child.Name,
			FullPath: child.FullPath,
			IsDir:    child.IsDir,
			Depth:    child.Depth,
			Expanded: child.Expanded,
			Icon:     getIcon(child.Name, child.IsDir),
		})

		if child.IsDir && child.Expanded {
			flatten(child, items)
		}
	}
}

// ToggleExpand toggles the expansion state of a specific node.
func (t *FileTree) ToggleExpand(fullPath string) {
	node := findNode(t.Root, fullPath)
	if node != nil && node.IsDir {
		node.Expanded = !node.Expanded
	}
}

func findNode(node *Node, fullPath string) *Node {
	if node.FullPath == fullPath {
		return node
	}
	for _, child := range node.Children {
		if strings.HasPrefix(fullPath, child.FullPath) {
			if child.FullPath == fullPath {
				return child
			}
			if found := findNode(child, fullPath); found != nil {
				return found
			}
		}
	}
	return nil
}

func getIcon(name string, isDir bool) string {
	if isDir {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return ""
	case ".js", ".ts", ".tsx":
		return ""
	case ".css", ".scss":
		return ""
	case ".html":
		return ""
	case ".json", ".yaml", ".yml", ".toml":
		return ""
	case ".md":
		return ""
	case ".png", ".jpg", ".jpeg", ".svg":
		return ""
	case ".gitignore", ".gitmodules":
		return ""
	default:
		return ""
	}
}
