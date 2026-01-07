package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DirEntry struct {
	Name  string
	Path  string
	IsDir bool
}

type DirectoryPicker struct {
	theme           *Theme
	currentDir      string
	entries         []DirEntry
	cursor          int
	offset          int
	height          int
	width           int
	selected        string
	cancelled       bool
	done            bool
	showHidden      bool
	dirOnly         bool
	err             error
	filterText      string
	filteredEntries []DirEntry
}

func NewDirectoryPicker(theme *Theme, startDir string, width, height int) *DirectoryPicker {
	if startDir == "" {
		startDir, _ = os.UserHomeDir()
	}

	if strings.HasPrefix(startDir, "~") {
		home, _ := os.UserHomeDir()
		startDir = filepath.Join(home, startDir[1:])
	}

	dp := &DirectoryPicker{
		theme:      theme,
		currentDir: startDir,
		height:     height,
		width:      width,
		dirOnly:    true,
	}

	dp.loadEntries()
	return dp
}

func (d *DirectoryPicker) loadEntries() {
	d.entries = nil
	d.filteredEntries = nil
	d.cursor = 0
	d.offset = 0
	d.err = nil

	files, err := os.ReadDir(d.currentDir)
	if err != nil {
		d.err = err
		return
	}

	if d.currentDir != "/" {
		d.entries = append(d.entries, DirEntry{
			Name:  "..",
			Path:  filepath.Dir(d.currentDir),
			IsDir: true,
		})
	}

	for _, f := range files {
		if !d.showHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}

		if d.dirOnly && !f.IsDir() {
			continue
		}

		d.entries = append(d.entries, DirEntry{
			Name:  f.Name(),
			Path:  filepath.Join(d.currentDir, f.Name()),
			IsDir: f.IsDir(),
		})
	}

	sort.Slice(d.entries, func(i, j int) bool {
		if d.entries[i].Name == ".." {
			return true
		}
		if d.entries[j].Name == ".." {
			return false
		}
		if d.entries[i].IsDir != d.entries[j].IsDir {
			return d.entries[i].IsDir
		}
		return strings.ToLower(d.entries[i].Name) < strings.ToLower(d.entries[j].Name)
	})

	d.applyFilter()
}

func (d *DirectoryPicker) applyFilter() {
	if d.filterText == "" {
		d.filteredEntries = d.entries
		return
	}

	d.filteredEntries = nil
	query := strings.ToLower(d.filterText)
	for _, e := range d.entries {
		if e.Name == ".." || strings.Contains(strings.ToLower(e.Name), query) {
			d.filteredEntries = append(d.filteredEntries, e)
		}
	}

	if d.cursor >= len(d.filteredEntries) {
		d.cursor = max(0, len(d.filteredEntries)-1)
	}
}

func (d *DirectoryPicker) Init() tea.Cmd {
	return nil
}

func (d *DirectoryPicker) Update(msg tea.Msg) (*DirectoryPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
				d.ensureVisible()
			}

		case "down", "j":
			if d.cursor < len(d.filteredEntries)-1 {
				d.cursor++
				d.ensureVisible()
			}

		case "pgup":
			d.cursor -= d.visibleHeight()
			if d.cursor < 0 {
				d.cursor = 0
			}
			d.ensureVisible()

		case "pgdown":
			d.cursor += d.visibleHeight()
			if d.cursor >= len(d.filteredEntries) {
				d.cursor = len(d.filteredEntries) - 1
			}
			d.ensureVisible()

		case "home", "g":
			d.cursor = 0
			d.offset = 0

		case "end", "G":
			d.cursor = len(d.filteredEntries) - 1
			d.ensureVisible()

		case "enter", "l", "right":
			if len(d.filteredEntries) > 0 {
				entry := d.filteredEntries[d.cursor]
				if entry.IsDir {
					d.currentDir = entry.Path
					d.filterText = ""
					d.loadEntries()
				}
			}

		case "backspace", "h", "left":
			if d.filterText != "" {
				d.filterText = d.filterText[:len(d.filterText)-1]
				d.applyFilter()
			} else if d.currentDir != "/" {
				d.currentDir = filepath.Dir(d.currentDir)
				d.loadEntries()
			}

		case " ":
			d.selected = d.currentDir
			d.done = true

		case "tab":
			d.selected = d.currentDir
			d.done = true

		case "esc":
			d.cancelled = true
			d.done = true

		case ".":
			d.showHidden = !d.showHidden
			d.loadEntries()

		default:
			if len(msg.String()) == 1 && msg.String() != " " {
				r := msg.Runes
				if len(r) == 1 && (r[0] >= 'a' && r[0] <= 'z' || r[0] >= 'A' && r[0] <= 'Z' || r[0] >= '0' && r[0] <= '9' || r[0] == '-' || r[0] == '_') {
					d.filterText += string(r)
					d.applyFilter()
				}
			}
		}
	}

	return d, nil
}

func (d *DirectoryPicker) ensureVisible() {
	visHeight := d.visibleHeight()
	if d.cursor < d.offset {
		d.offset = d.cursor
	} else if d.cursor >= d.offset+visHeight {
		d.offset = d.cursor - visHeight + 1
	}
}

func (d *DirectoryPicker) visibleHeight() int {
	return d.height - 6
}

func (d *DirectoryPicker) View() string {
	var b strings.Builder

	titleStyle := d.theme.Title.Copy().MarginBottom(1)
	b.WriteString(titleStyle.Render("Select Directory"))
	b.WriteString("\n")

	pathStyle := d.theme.Base.Copy().Foreground(d.theme.Secondary).Bold(true)
	path := d.currentDir
	if len(path) > d.width-4 {
		path = "..." + path[len(path)-d.width+7:]
	}
	b.WriteString(pathStyle.Render(path))
	b.WriteString("\n\n")

	if d.err != nil {
		errStyle := d.theme.StatusStyle(2)
		b.WriteString(errStyle.Render("Error: " + d.err.Error()))
		b.WriteString("\n")
		return d.wrapInBox(b.String())
	}

	if d.filterText != "" {
		filterStyle := d.theme.Base.Copy().Foreground(d.theme.Primary)
		b.WriteString(filterStyle.Render("Filter: " + d.filterText))
		b.WriteString("\n")
	}

	visHeight := d.visibleHeight()
	if d.filterText != "" {
		visHeight--
	}

	for i := d.offset; i < len(d.filteredEntries) && i < d.offset+visHeight; i++ {
		entry := d.filteredEntries[i]
		line := d.renderEntry(entry, i == d.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	for i := len(d.filteredEntries); i < d.offset+visHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := d.theme.Base.Copy().Faint(true)
	help := "j/k:nav  enter:open  space/tab:select  esc:cancel  .:hidden"
	b.WriteString(helpStyle.Render(help))

	return d.wrapInBox(b.String())
}

func (d *DirectoryPicker) renderEntry(entry DirEntry, selected bool) string {
	var style lipgloss.Style
	var icon string

	if entry.IsDir {
		icon = ""
		if selected {
			style = d.theme.SelectedItemStyle.Copy()
		} else {
			style = d.theme.NormalItemStyle.Copy().Foreground(d.theme.Primary)
		}
	} else {
		icon = ""
		if selected {
			style = d.theme.SelectedItemStyle.Copy()
		} else {
			style = d.theme.NormalItemStyle.Copy()
		}
	}

	name := entry.Name
	maxNameLen := d.width - 8
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}

	cursor := "  "
	if selected {
		cursor = "> "
	}

	return style.Render(cursor + icon + " " + name)
}

func (d *DirectoryPicker) wrapInBox(content string) string {
	boxStyle := d.theme.ViewportStyle.Copy().
		Width(d.width).
		Height(d.height).
		BorderForeground(d.theme.Primary)

	return boxStyle.Render(content)
}

func (d *DirectoryPicker) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DirectoryPicker) Selected() string {
	return d.selected
}

func (d *DirectoryPicker) IsDone() bool {
	return d.done
}

func (d *DirectoryPicker) IsCancelled() bool {
	return d.cancelled
}

func (d *DirectoryPicker) CurrentDirectory() string {
	return d.currentDir
}

func (d *DirectoryPicker) Reset(startDir string) {
	if startDir == "" {
		startDir, _ = os.UserHomeDir()
	}
	d.currentDir = startDir
	d.selected = ""
	d.cancelled = false
	d.done = false
	d.filterText = ""
	d.loadEntries()
}

type DirPickerSelectedMsg struct {
	Path string
}

type DirPickerCancelledMsg struct{}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
