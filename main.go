package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	items    []string
	paths    []string
	cursor   int
	filter   string
	selected string
	root     string
	height   int
	offset   int  // scroll offset
	showHelp bool // show help screen
}

func newModel(start string) model {
	if start == "" {
		start, _ = os.Getwd()
	}
	if start == "~" {
		start, _ = os.UserHomeDir()
	}
	if strings.HasPrefix(start, "~/") {
		home, _ := os.UserHomeDir()
		start = home + start[1:]
	}

	items, paths := loadDir(start)
	return model{
		root:  start,
		items: items,
		paths: paths,
	}
}

func loadDir(root string) ([]string, []string) {
	// Show current folder name as first item (to select current dir)
	currentName := filepath.Base(root)
	if root == "/" {
		currentName = "/"
	}
	items := []string{"[" + currentName + "]"}
	paths := []string{root}

	entries, _ := os.ReadDir(root)
	var dirs []string
	dirMap := make(map[string]string)

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.Name() == "node_modules" || e.Name() == "vendor" {
			continue
		}
		dirs = append(dirs, e.Name())
		dirMap[e.Name()] = filepath.Join(root, e.Name())
	}

	sort.Strings(dirs)
	for _, d := range dirs {
		items = append(items, d)
		paths = append(paths, dirMap[d])
	}
	return items, paths
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) fixScroll() {
	visible := m.visibleLines()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m model) visibleLines() int {
	// Reserve lines for: path, filter, empty, scroll indicator, help
	reserved := 5
	if m.height <= reserved {
		return 5 // smaller fallback
	}
	return m.height - reserved
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		k := msg.String()
		filtered := m.filtered()

		switch k {
		case "ctrl+c":
			return m, tea.Quit
		case "f1":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			// Go to parent folder
			parent := filepath.Dir(m.root)
			if parent != m.root {
				m.root = parent
				m.filter = ""
				m.items, m.paths = loadDir(m.root)
				m.cursor = 0
				m.offset = 0
			}
		case "up":
			if m.cursor > 0 {
				m.cursor--
				m.fixScroll()
			}
		case "down":
			if m.cursor < len(filtered)-1 {
				m.cursor++
				m.fixScroll()
			}
		case "enter":
			if len(filtered) > 0 {
				selectedPath := filtered[m.cursor].path
				// If selecting current folder, go to parent instead
				if selectedPath == m.root {
					parent := filepath.Dir(m.root)
					if parent != m.root {
						m.root = parent
						m.filter = ""
						m.items, m.paths = loadDir(m.root)
						m.cursor = 0
						m.offset = 0
					}
				} else {
					m.root = selectedPath
					m.filter = ""
					m.items, m.paths = loadDir(m.root)
					m.cursor = 0
					m.offset = 0
				}
			}
		case "tab":
			if len(filtered) > 0 {
				m.selected = filtered[m.cursor].path
				return m, tea.Quit
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
				m.offset = 0
			}
		default:
			if len(k) == 1 && k >= " " {
				m.filter += k
				m.cursor = 0
				m.offset = 0
			}
		}
	}
	return m, nil
}

type item struct {
	name string
	path string
}

func (m model) filtered() []item {
	var result []item
	f := strings.ToLower(strings.TrimSpace(m.filter))

	if f == "" {
		for i, name := range m.items {
			result = append(result, item{name, m.paths[i]})
		}
		return result
	}

	// Split filter into words - ALL words must match
	words := strings.Fields(f)

	for i, name := range m.items {
		nameLower := strings.ToLower(name)
		allMatch := true
		for _, word := range words {
			if !strings.Contains(nameLower, word) {
				allMatch = false
				break
			}
		}
		if allMatch {
			result = append(result, item{name, m.paths[i]})
		}
	}
	return result
}

func (m model) helpView() string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  \033[1;34mpf - folder picker\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[1m↑ / ↓\033[0m       Navigate list")
	lines = append(lines, "  \033[1mEnter\033[0m       Open folder")
	lines = append(lines, "  \033[1mTab\033[0m         Select & cd to folder")
	lines = append(lines, "  \033[1mEsc\033[0m         Go to parent folder")
	lines = append(lines, "  \033[1mBackspace\033[0m   Clear filter character")
	lines = append(lines, "  \033[1mCtrl+C\033[0m      Quit without select")
	lines = append(lines, "  \033[1mF1\033[0m          Toggle this help")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mType any text to filter folders")
	lines = append(lines, "  Multiple words = match all\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mPress Esc or F1 to close\033[0m")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func (m model) View() string {
	if m.showHelp {
		return m.helpView()
	}

	var lines []string

	// Show path
	home, _ := os.UserHomeDir()
	path := m.root
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	lines = append(lines, "\033[1;34m"+path+"\033[0m")

	// Filter - always show
	if m.filter != "" {
		lines = append(lines, "\033[33mFilter: "+m.filter+"_\033[0m")
	} else {
		lines = append(lines, "\033[90mType to filter...\033[0m")
	}
	lines = append(lines, "") // empty line

	// Items (with scrolling)
	filtered := m.filtered()
	visible := m.visibleLines()
	start := m.offset
	end := start + visible
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		it := filtered[i]
		if i == m.cursor {
			lines = append(lines, "\033[1;34m> "+it.name+"\033[0m")
		} else {
			lines = append(lines, "  "+it.name)
		}
	}

	// Show scroll indicator if needed
	if len(filtered) > visible {
		lines = append(lines, fmt.Sprintf("\033[90m(%d-%d of %d)\033[0m", start+1, end, len(filtered)))
	} else {
		lines = append(lines, "") // keep spacing consistent
	}

	lines = append(lines, "\033[48;5;236m\033[97m ↑↓ nav • Esc back • Enter open • Tab select • F1 help \033[0m")

	return strings.Join(lines, "\n")
}

func installShellFunction() {
	home, _ := os.UserHomeDir()
	shell := os.Getenv("SHELL")
	rcFile := filepath.Join(home, ".zshrc")
	rcName := "~/.zshrc"
	if strings.Contains(shell, "bash") {
		rcFile = filepath.Join(home, ".bashrc")
		rcName = "~/.bashrc"
	}

	shellFunc := `
# pf - folder picker
pf() {
  local dir
  dir=$(command pf "$@")
  if [[ -n "$dir" && -d "$dir" ]]; then
    cd "$dir"
  fi
}
`

	// Check if already installed
	content, err := os.ReadFile(rcFile)
	if err == nil && strings.Contains(string(content), "dir=$(command pf") {
		fmt.Fprintln(os.Stderr, "pf is already installed in "+rcName)
		return
	}

	// Show what will be added
	fmt.Fprintln(os.Stderr, "This will add to "+rcName+":")
	fmt.Fprintln(os.Stderr, shellFunc)
	fmt.Fprint(os.Stderr, "\033[48;5;236m\033[97m Add to "+rcName+"? (y/n): \033[0m ")

	// Read answer
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" {
		fmt.Fprintln(os.Stderr, "Installation cancelled.")
		return
	}

	// Append to rc file
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening "+rcName+": "+err.Error())
		return
	}
	defer f.Close()

	_, err = f.WriteString(shellFunc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to "+rcName+": "+err.Error())
		return
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Installed! Run this to activate:")
	fmt.Fprintln(os.Stderr, "  source "+rcName)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "pf - folder picker")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Usage: pf [start-path]")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --install    Show shell function installation instructions")
			fmt.Fprintln(os.Stderr, "  --help, -h   Show this help")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Press F1 inside pf for keyboard shortcuts.")
			return
		case "--install":
			installShellFunction()
			return
		}
	}

	start := ""
	if len(os.Args) > 1 {
		start = os.Args[1]
	}

	// Output TUI to stderr so shell capture $() only gets the selected path
	p := tea.NewProgram(newModel(start), tea.WithOutput(os.Stderr))
	final, _ := p.Run()

	if m, ok := final.(model); ok && m.selected != "" {
		fmt.Println(m.selected)
	}
}
