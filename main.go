package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "1.1.2"

type model struct {
	items         []string
	paths         []string
	cursor        int
	filter        string
	selected      string
	root          string
	height        int
	offset        int    // scroll offset
	showHelp      bool   // show help screen
	confirmDelete bool   // show delete confirmation
	deleteTarget  string // path to delete
	deleteError   string // error message after delete attempt
	createMode    bool   // show create folder input
	newFolderName string // name for new folder
	createError   string // error message after create attempt
	confirmArchive bool   // show archive confirmation
	archiveTarget  string // path to archive
	archiveError   string // error message after archive attempt
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

		// Handle archive confirmation dialog
		if m.confirmArchive {
			switch k {
			case "y", "Y":
				// Create ~/Dev-Archive if needed
				home, _ := os.UserHomeDir()
				archiveDir := filepath.Join(home, "Dev-Archive")
				if err := os.MkdirAll(archiveDir, 0755); err != nil {
					m.archiveError = "Error creating archive dir: " + err.Error()
					m.confirmArchive = false
					m.archiveTarget = ""
					return m, nil
				}
				// Move folder to archive
				folderName := filepath.Base(m.archiveTarget)
				destPath := filepath.Join(archiveDir, folderName)
				err := os.Rename(m.archiveTarget, destPath)
				if err != nil {
					m.archiveError = "Error: " + err.Error()
					m.confirmArchive = false
					m.archiveTarget = ""
					return m, nil
				}
				// Refresh the current directory
				m.items, m.paths = loadDir(m.root)
				m.cursor = 0
				m.offset = 0
				m.confirmArchive = false
				m.archiveTarget = ""
				m.archiveError = ""
				return m, nil
			case "n", "N", "esc":
				m.confirmArchive = false
				m.archiveTarget = ""
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle delete confirmation dialog
		if m.confirmDelete {
			switch k {
			case "y", "Y":
				// Delete the folder
				err := os.RemoveAll(m.deleteTarget)
				if err != nil {
					m.deleteError = "Error: " + err.Error()
					m.confirmDelete = false
					m.deleteTarget = ""
					return m, nil
				}
				// Refresh the current directory
				m.items, m.paths = loadDir(m.root)
				m.cursor = 0
				m.offset = 0
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteError = ""
				return m, nil
			case "n", "N", "esc":
				m.confirmDelete = false
				m.deleteTarget = ""
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle create folder mode
		if m.createMode {
			switch k {
			case "enter":
				if m.newFolderName != "" {
					newPath := filepath.Join(m.root, m.newFolderName)
					err := os.Mkdir(newPath, 0755)
					if err != nil {
						m.createError = "Error: " + err.Error()
						m.createMode = false
						m.newFolderName = ""
						return m, nil
					}
					// Refresh and select the new folder
					m.items, m.paths = loadDir(m.root)
					m.cursor = 0
					m.offset = 0
					// Find and select the new folder
					for i, p := range m.paths {
						if p == newPath {
							m.cursor = i
							m.fixScroll()
							break
						}
					}
					m.createMode = false
					m.newFolderName = ""
					m.createError = ""
				}
				return m, nil
			case "esc":
				m.createMode = false
				m.newFolderName = ""
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			case "backspace":
				if len(m.newFolderName) > 0 {
					m.newFolderName = m.newFolderName[:len(m.newFolderName)-1]
				}
				return m, nil
			default:
				if len(k) == 1 && k >= " " {
					m.newFolderName += k
				}
				return m, nil
			}
		}

		// Clear error messages on any key
		if m.deleteError != "" {
			m.deleteError = ""
		}
		if m.createError != "" {
			m.createError = ""
		}
		if m.archiveError != "" {
			m.archiveError = ""
		}

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
				previousFolder := m.root
				m.root = parent
				m.filter = ""
				m.items, m.paths = loadDir(m.root)
				m.cursor = 0
				m.offset = 0
				// Select the folder we came from
				for i, p := range m.paths {
					if p == previousFolder {
						m.cursor = i
						m.fixScroll()
						break
					}
				}
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
						previousFolder := m.root
						m.root = parent
						m.filter = ""
						m.items, m.paths = loadDir(m.root)
						m.cursor = 0
						m.offset = 0
						// Select the folder we came from
						for i, p := range m.paths {
							if p == previousFolder {
								m.cursor = i
								m.fixScroll()
								break
							}
						}
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
		case "alt+backspace", "ctrl+backspace":
			// Delete folder - show confirmation
			if len(filtered) > 0 {
				selectedPath := filtered[m.cursor].path
				// Don't allow deleting the current folder indicator or root
				if selectedPath != m.root && selectedPath != "/" {
					m.confirmDelete = true
					m.deleteTarget = selectedPath
				}
			}
		case "ctrl+n":
			// Create new folder
			m.createMode = true
			m.newFolderName = ""
		case "ctrl+a":
			// Archive folder - move to ~/Dev-Archive
			if len(filtered) > 0 {
				selectedPath := filtered[m.cursor].path
				// Don't allow archiving the current folder indicator or root
				if selectedPath != m.root && selectedPath != "/" {
					m.confirmArchive = true
					m.archiveTarget = selectedPath
				}
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
	lines = append(lines, "  \033[1;34mpf - folder picker\033[0m  \033[90mv"+version+"\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[1m↑ / ↓\033[0m       Navigate list")
	lines = append(lines, "  \033[1mEnter\033[0m       Open folder")
	lines = append(lines, "  \033[1mTab\033[0m         Select & cd to folder")
	lines = append(lines, "  \033[1mEsc\033[0m         Go to parent folder")
	lines = append(lines, "  \033[1mBackspace\033[0m   Clear filter character")
	lines = append(lines, "  \033[1mCtrl+N\033[0m      Create new folder")
	lines = append(lines, "  \033[1mCtrl+A\033[0m      Archive folder (~/Dev-Archive)")
	lines = append(lines, "  \033[1mAlt+⌫\033[0m       Delete selected folder")
	lines = append(lines, "  \033[1mCtrl+C\033[0m      Quit without select")
	lines = append(lines, "  \033[1mF1\033[0m          Toggle this help")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mType any text to filter folders")
	lines = append(lines, "  Multiple words = match all\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mPress Esc or F1 to close\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mhttps://pf.pm7.dev\033[0m")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func (m model) confirmDeleteView() string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  \033[1;31mDelete folder?\033[0m")
	lines = append(lines, "")

	// Show the folder path nicely
	home, _ := os.UserHomeDir()
	displayPath := m.deleteTarget
	if strings.HasPrefix(displayPath, home) {
		displayPath = "~" + displayPath[len(home):]
	}
	lines = append(lines, "  \033[1m"+displayPath+"\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[90mThis will permanently delete the folder")
	lines = append(lines, "  and all its contents!\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[48;5;236m\033[97m y = delete • n/Esc = cancel \033[0m")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func (m model) createFolderView() string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  \033[1;32mCreate new folder\033[0m")
	lines = append(lines, "")

	// Show current path
	home, _ := os.UserHomeDir()
	displayPath := m.root
	if strings.HasPrefix(displayPath, home) {
		displayPath = "~" + displayPath[len(home):]
	}
	lines = append(lines, "  \033[90min "+displayPath+"/\033[0m")
	lines = append(lines, "")

	// Show input field
	lines = append(lines, "  \033[1mName: "+m.newFolderName+"_\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[48;5;236m\033[97m Enter = create • Esc = cancel \033[0m")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func (m model) confirmArchiveView() string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  \033[1;33mMove to Archive?\033[0m")
	lines = append(lines, "")

	// Show the source folder path
	home, _ := os.UserHomeDir()
	displayPath := m.archiveTarget
	if strings.HasPrefix(displayPath, home) {
		displayPath = "~" + displayPath[len(home):]
	}
	lines = append(lines, "  \033[1mFrom:\033[0m "+displayPath)

	// Show the destination path
	folderName := filepath.Base(m.archiveTarget)
	lines = append(lines, "  \033[1mTo:\033[0m   ~/Dev-Archive/"+folderName)
	lines = append(lines, "")
	lines = append(lines, "  \033[90mThe folder will be moved to your archive.\033[0m")
	lines = append(lines, "")
	lines = append(lines, "  \033[48;5;236m\033[97m y = archive • n/Esc = cancel \033[0m")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func (m model) View() string {
	if m.showHelp {
		return m.helpView()
	}

	if m.confirmDelete {
		return m.confirmDeleteView()
	}

	if m.confirmArchive {
		return m.confirmArchiveView()
	}

	if m.createMode {
		return m.createFolderView()
	}

	var lines []string

	// Show path
	home, _ := os.UserHomeDir()
	path := m.root
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	lines = append(lines, "\033[1;34m"+path+"\033[0m")

	// Show error if any
	if m.deleteError != "" {
		lines = append(lines, "\033[31m"+m.deleteError+"\033[0m")
	} else if m.createError != "" {
		lines = append(lines, "\033[31m"+m.createError+"\033[0m")
	} else if m.archiveError != "" {
		lines = append(lines, "\033[31m"+m.archiveError+"\033[0m")
	} else if m.filter != "" {
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

	lines = append(lines, "\033[48;5;236m\033[97m ↑↓ nav • Enter open • Tab select • ^N new • F1 help \033[0m")

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
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "https://pf.pm7.dev")
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
