package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QubeUtils/repowalk/internal/aggregator"
	"github.com/QubeUtils/repowalk/internal/tree"
	"github.com/QubeUtils/repowalk/internal/version"
	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkoukk/tiktoken-go"
)

var tkm, _ = tiktoken.GetEncoding("cl100k_base")

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3"))
	directoryStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498DB")).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7698")).Bold(true)
	metricsStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0")).MarginTop(1)
	keybindStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
)

type UINode struct {
	Name      string
	Path      string
	IsDir     bool
	Selected  bool
	Expanded  bool
	IsIgnored bool
	Children  map[string]*UINode
	FileNode  *walker.FileNode
}

type Model struct {
	root        *UINode
	templateStr string
	persona     string
	theme       string
	defaultOut  string
	costPer1M   float64

	flatNodes      []*flatNode
	cursor         int
	viewportOffset int
	height         int
	width          int

	totalSelected int
	totalSize     int64
	totalTokens   int
	message       string
	newVersion    string

	searchInput    textinput.Model
	isSearching    bool
	exportInput    textinput.Model
	exportStep     int // 0: none, 1: filename, 2: directory
	exportFileName string
	quitting       bool
	fileViewport   viewport.Model

	editMode   bool
	editCursor int
}

type flatNode struct {
	Node   *UINode
	Prefix string
	Level  int
}

func NewModel(files []walker.FileNode, templateStr, persona, theme, defaultOut string, costPer1M float64) *Model {
	root := &UINode{
		Name:     ".",
		Path:     ".",
		IsDir:    true,
		Selected: true, // Select all by default
		Expanded: true,
		Children: make(map[string]*UINode),
	}

	for i := range files {
		f := files[i]
		parts := strings.Split(filepath.ToSlash(f.Path), "/")
		current := root

		for j, part := range parts {
			if existing, exists := current.Children[part]; !exists {
				isDir := j < len(parts)-1 || (j == len(parts)-1 && f.IsDir)
				pathSoFar := strings.Join(parts[:j+1], "/")
				current.Children[part] = &UINode{
					Name:      part,
					Path:      pathSoFar,
					IsDir:     isDir,
					Selected:  !f.IsIgnored, // Select by default unless ignored
					Expanded:  true,
					IsIgnored: f.IsIgnored,
					Children:  make(map[string]*UINode),
				}
			} else {
				// If a node was already created but we now know it has children or is explicitly a dir
				if j < len(parts)-1 || f.IsDir {
					existing.IsDir = true
				}
			}
			current = current.Children[part]
		}
		current.FileNode = &f
	}

	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.CharLimit = 156
	ti.Width = 40

	ei := textinput.New()
	ei.Placeholder = defaultOut
	ei.SetValue(defaultOut)
	ei.CharLimit = 256
	ei.Width = 60

	m := &Model{
		root:        root,
		templateStr: templateStr,
		persona:     persona,
		theme:       theme,
		defaultOut:  defaultOut,
		costPer1M:   costPer1M,
		searchInput: ti,
		exportInput: ei,
	}

	vp := viewport.New(40, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)
	m.fileViewport = vp

	m.updateFlatNodes()
	m.updateMetrics()
	m.updateViewportContent()
	return m
}

func (m *Model) updateFlatNodes() {
	var flat []*flatNode

	var traverse func(node *UINode, drawingPrefix string, childPrefix string, level int) bool
	traverse = func(node *UINode, drawingPrefix string, childPrefix string, level int) bool {
		// If searching, check if this node or any child matches
		query := strings.ToLower(m.searchInput.Value())
		matches := true
		if query != "" {
			matches = strings.Contains(strings.ToLower(node.Name), query)
		}

		// Pre-calculate which children match if we are searching, to know if this dir should be shown
		var matchingChildren []string
		if node.IsDir {
			for k, v := range node.Children {
				if query == "" || strings.Contains(strings.ToLower(k), query) || hasMatchingChild(v, query) {
					matchingChildren = append(matchingChildren, k)
				}
			}
			if len(matchingChildren) > 0 {
				matches = true // Show this dir if any child matches
			}
		}

		if !matches && query != "" {
			return false
		}

		flat = append(flat, &flatNode{
			Node:   node,
			Prefix: drawingPrefix,
			Level:  level,
		})

		if node.IsDir && node.Expanded {
			var names []string
			if query != "" {
				names = matchingChildren
			} else {
				for k := range node.Children {
					names = append(names, k)
				}
			}
			sort.Strings(names)

			for i, name := range names {
				child := node.Children[name]
				isLast := i == len(names)-1

				connector := "├── "
				newChildPrefix := childPrefix + "│   "
				if isLast {
					connector = "└── "
					newChildPrefix = childPrefix + "    "
				}

				traverse(child, childPrefix+connector, newChildPrefix, level+1)
			}
		}
		return matches
	}

	traverse(m.root, "", "", 0)

	// Reset cursor if out of bounds after filtering
	if m.cursor >= len(flat) && len(flat) > 0 {
		m.cursor = len(flat) - 1
	} else if len(flat) == 0 {
		m.cursor = 0
	}

	m.flatNodes = flat
}

func hasMatchingChild(node *UINode, query string) bool {
	if strings.Contains(strings.ToLower(node.Name), query) {
		return true
	}
	for _, child := range node.Children {
		if hasMatchingChild(child, query) {
			return true
		}
	}
	return false
}

func (m *Model) updateMetrics() {
	m.totalSelected = 0
	m.totalSize = 0
	m.totalTokens = 0

	var traverse func(node *UINode)
	traverse = func(node *UINode) {
		if !node.IsDir && node.Selected && node.FileNode != nil {
			m.totalSelected++
			m.totalSize += node.FileNode.Size
			m.totalTokens += node.FileNode.Tokens
		}
		for _, child := range node.Children {
			traverse(child)
		}
	}

	traverse(m.root)
}

func (m *Model) toggleSelection(node *UINode, state bool) {
	node.Selected = state
	for _, child := range node.Children {
		m.toggleSelection(child, state)
	}
}

type updateMsg struct {
	NewVersion string
}

func checkUpdate() tea.Msg {
	// Set a small timeout so we don't hang if they don't have internet
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/QubeUtils/repowalk/releases/latest")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil
	}

	if rel.TagName != "" && rel.TagName != version.Version {
		return updateMsg{NewVersion: rel.TagName}
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return checkUpdate
}

func (m *Model) generateDump() string {
	var selectedFiles []walker.FileNode
	var traverse func(node *UINode)
	traverse = func(node *UINode) {
		if !node.IsDir && node.Selected && node.FileNode != nil {
			selectedFiles = append(selectedFiles, *node.FileNode)
		}

		var names []string
		for k := range node.Children {
			names = append(names, k)
		}
		sort.Strings(names)

		for _, name := range names {
			traverse(node.Children[name])
		}
	}
	traverse(m.root)

	treeStr := tree.GenerateTree(selectedFiles, 0)

	dumpOpts := aggregator.DumpOptions{
		TemplateString: m.templateStr,
		TreeString:     treeStr,
		Persona:        m.persona,
	}

	out, err := aggregator.GenerateMarkdown(selectedFiles, dumpOpts)
	if err != nil {
		return fmt.Sprintf("Error generating dump: %v", err)
	}
	return out
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editMode {
		return m.handleEditMode(msg)
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case updateMsg:
		m.newVersion = msg.NewVersion
		return m, nil
	case tea.KeyMsg:
		if m.isSearching {
			switch msg.String() {
			case "enter", "esc":
				m.isSearching = false
				m.searchInput.Blur()
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.updateFlatNodes()
			}
			return m, cmd
		}

		if m.exportStep > 0 {
			switch msg.String() {
			case "esc":
				m.exportStep = 0
				m.exportInput.Blur()
			case "enter":
				if m.exportStep == 1 {
					m.exportFileName = m.exportInput.Value()
					if strings.TrimSpace(m.exportFileName) == "" {
						m.exportFileName = filepath.Base(m.defaultOut)
					}
					m.exportStep = 2
					m.exportInput.Reset()
					dir := filepath.Dir(m.defaultOut)
					if dir == "" {
						dir = "."
					}
					m.exportInput.SetValue(dir)
				} else if m.exportStep == 2 {
					exportDir := m.exportInput.Value()
					if strings.TrimSpace(exportDir) == "" {
						exportDir = filepath.Dir(m.defaultOut)
						if exportDir == "" {
							exportDir = "."
						}
					}
					outPath := filepath.Join(exportDir, m.exportFileName)
					m.exportStep = 0
					m.exportInput.Blur()

					if err := os.MkdirAll(exportDir, 0755); err != nil {
						m.message = "Failed to create directory: " + err.Error()
						return m, nil
					}
					dump := m.generateDump()
					err := os.WriteFile(outPath, []byte(dump), 0644)
					if err != nil {
						m.message = fmt.Sprintf("Failed to save to %s: %v", outPath, err)
					} else {
						m.message = fmt.Sprintf("Successfully saved context to %s!", outPath)
					}
				}
			default:
				m.exportInput, cmd = m.exportInput.Update(msg)
			}
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "/":
			m.isSearching = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
			}

		case "left", "h":
			if len(m.flatNodes) > 0 {
				node := m.flatNodes[m.cursor].Node
				if node.IsDir && node.Expanded {
					node.Expanded = false
					m.updateFlatNodes()
				}
			}

		case "right", "l":
			if len(m.flatNodes) > 0 {
				node := m.flatNodes[m.cursor].Node
				if node.IsDir && !node.Expanded {
					node.Expanded = true
					m.updateFlatNodes()
				}
			}

		case " ":
			if len(m.flatNodes) > 0 {
				node := m.flatNodes[m.cursor].Node
				m.toggleSelection(node, !node.Selected)
				m.updateMetrics()
			}

		case "e":
			if len(m.flatNodes) > 0 {
				node := m.flatNodes[m.cursor].Node
				if !node.IsDir && node.FileNode != nil && !node.FileNode.IsBinary {
					m.editMode = true
					m.editCursor = 0
					m.renderEditViewport()
				}
			}

		case "s":
			m.exportStep = 1
			m.exportInput.Reset()
			m.exportInput.SetValue(filepath.Base(m.defaultOut))
			m.exportInput.Focus()
			return m, textinput.Blink

		case "enter":
			// Fast export to default
			dump := m.generateDump()
			err := os.WriteFile(m.defaultOut, []byte(dump), 0644)
			if err != nil {
				m.message = fmt.Sprintf("Failed to save to %s: %v", m.defaultOut, err)
			} else {
				m.message = fmt.Sprintf("Successfully saved context to %s!", m.defaultOut)
			}

		case "c":
			dump := m.generateDump()
			err := clipboard.WriteAll(dump)
			if err != nil {
				m.message = fmt.Sprintf("Failed to copy to clipboard: %v", err)
			} else {
				m.message = "Successfully copied context to clipboard!"
			}

		case "pgup":
			m.fileViewport.HalfPageUp()

		case "pgdown":
			m.fileViewport.HalfPageDown()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileViewport.Width = msg.Width/2 - 4
		m.fileViewport.Height = msg.Height - 12
	}

	// Basic viewport handling
	visibleLines := m.height - 12
	if visibleLines < 5 {
		visibleLines = 5
	}

	if m.cursor < m.viewportOffset {
		m.viewportOffset = m.cursor
	} else if m.cursor >= m.viewportOffset+visibleLines {
		m.viewportOffset = m.cursor - visibleLines + 1
	}

	m.updateViewportContent()
	return m, nil
}

func (m *Model) updateViewportContent() {
	if m.cursor >= 0 && m.cursor < len(m.flatNodes) {
		fn := m.flatNodes[m.cursor]
		if !fn.Node.IsDir && fn.Node.FileNode != nil {
			if fn.Node.FileNode.IsBinary {
				m.fileViewport.SetContent("[Binary File or Content Too Large]")
			} else {
				if m.editMode {
					m.renderEditViewport()
				} else {
					var buf bytes.Buffer
					code := string(fn.Node.FileNode.Content)

					// Attempt to highlight
					ext := strings.TrimPrefix(filepath.Ext(fn.Node.Name), ".")
					if ext == "" {
						ext = fn.Node.Name
					}

					err := quick.Highlight(&buf, code, ext, "terminal256", m.theme)
					if err == nil && buf.Len() > 0 {
						m.fileViewport.SetContent(buf.String())
					} else {
						m.fileViewport.SetContent(code)
					}
				}
			}
		} else {
			m.fileViewport.SetContent("[Directory]")
		}
	}
}

func (m *Model) handleEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "e":
			m.editMode = false
			m.updateViewportContent() // Reset to normal view
			return m, nil
		case "up", "k":
			if m.editCursor > 0 {
				m.editCursor--
				m.renderEditViewport()
			}
		case "down", "j":
			fn := m.flatNodes[m.cursor].Node.FileNode
			if m.editCursor < len(fn.OriginalLines)-1 {
				m.editCursor++
				m.renderEditViewport()
			}
		case " ":
			fn := m.flatNodes[m.cursor].Node.FileNode
			fn.LineStates[m.editCursor] = (fn.LineStates[m.editCursor] + 1) % 3
			m.reconstructFileContent(fn)
			m.updateMetrics() // Tokens changed
			m.renderEditViewport()
		case "pgdown":
			m.fileViewport.HalfPageDown()
		case "pgup":
			m.fileViewport.HalfPageUp()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileViewport.Width = msg.Width/2 - 4
		m.fileViewport.Height = msg.Height - 12
		m.renderEditViewport()
	}
	return m, nil
}

func (m *Model) renderEditViewport() {
	fn := m.flatNodes[m.cursor].Node.FileNode
	if fn == nil || fn.IsBinary {
		return
	}

	if fn.OriginalLines == nil {
		fn.OriginalLines = strings.Split(string(fn.Content), "\n")
		fn.LineStates = make(map[int]int)
	}

	hasExclusive := false
	for _, state := range fn.LineStates {
		if state == 2 {
			hasExclusive = true
			break
		}
	}

	// Auto-scroll viewport
	if m.editCursor < m.fileViewport.YOffset {
		m.fileViewport.SetYOffset(m.editCursor)
	} else if m.editCursor >= m.fileViewport.YOffset+m.fileViewport.Height {
		m.fileViewport.SetYOffset(m.editCursor - m.fileViewport.Height + 1)
	}

	var b strings.Builder
	for i, line := range fn.OriginalLines {
		style := lipgloss.NewStyle()

		// Remove Windows carriage return for cleaner rendering
		cleanLine := strings.TrimRight(line, "\r")

		state := fn.LineStates[i]

		if state == 1 {
			style = style.Strikethrough(true).Foreground(lipgloss.Color("#7F8C8D"))
			if i == m.editCursor {
				style = style.Background(lipgloss.Color("#34495E")) // Dimmed cursor for stripped lines
			}
		} else if state == 2 {
			style = style.Foreground(lipgloss.Color("#2ECC71")).Bold(true) // Green
			if i == m.editCursor {
				style = style.Background(lipgloss.Color("#27AE60")).Foreground(lipgloss.Color("#FFFFFF"))
			}
		} else {
			if hasExclusive {
				style = style.Foreground(lipgloss.Color("#7F8C8D")) // Gray out normal lines
			}
			if i == m.editCursor {
				style = style.Background(lipgloss.Color("#E74C3C")).Foreground(lipgloss.Color("#FFFFFF"))
			}
		}

		b.WriteString(style.Render(cleanLine) + "\n")
	}

	m.fileViewport.SetContent(b.String())
}

func (m *Model) reconstructFileContent(fn *walker.FileNode) {
	var keep []string
	hasExclusive := false
	for _, state := range fn.LineStates {
		if state == 2 {
			hasExclusive = true
			break
		}
	}

	for i, line := range fn.OriginalLines {
		state := fn.LineStates[i]
		if state == 2 {
			keep = append(keep, line)
		} else if state == 0 && !hasExclusive {
			keep = append(keep, line)
		}
	}
	fn.Content = []byte(strings.Join(keep, "\n"))
	if tkm != nil {
		fn.Tokens = len(tkm.Encode(string(fn.Content), nil, nil))
	}
}

func (m *Model) View() string {
	if m.quitting {
		return "Exiting RepoWalk UI...\n"
	}

	var header strings.Builder
	title := "RepoWalk Interactive Setup"
	if m.newVersion != "" {
		title += fmt.Sprintf(" | 🚀 Update available: %s! (Run 'npm i -g repowalk')", m.newVersion)
	}
	header.WriteString(titleStyle.Render(title))
	header.WriteString("\n")

	if m.isSearching {
		header.WriteString("\n" + m.searchInput.View() + "\n\n")
	} else if m.exportStep == 1 {
		header.WriteString("\nExport Filename: " + m.exportInput.View() + "\n\n")
	} else if m.exportStep == 2 {
		header.WriteString(fmt.Sprintf("\nExport Filename: %s\nExport Directory: %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(m.exportFileName), m.exportInput.View()))
	} else {
		header.WriteString("\n\n")
	}

	// 12 lines for header/footer (title, search, metrics, keybinds)
	visibleLines := m.height - 12
	if visibleLines < 5 {
		visibleLines = 5
	}

	endIdx := m.viewportOffset + visibleLines
	if endIdx > len(m.flatNodes) {
		endIdx = len(m.flatNodes)
	}

	var treePane strings.Builder
	for i := m.viewportOffset; i < endIdx; i++ {
		fn := m.flatNodes[i]

		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		if fn.Node.Selected {
			checkbox = selectedStyle.Render("[x]")
		} else {
			checkbox = unselectedStyle.Render(checkbox)
		}

		name := fn.Node.Name
		if fn.Node.IsDir {
			name = directoryStyle.Render(name)
			if fn.Node.Expanded {
				name += " [-]"
			} else {
				name += " [+]"
			}
		}

		if !fn.Node.Selected {
			name = unselectedStyle.Render(name)
		}

		if fn.Node.IsIgnored {
			name += " (ignored)"
		}

		prefix := fn.Prefix
		if fn.Level == 0 {
			prefix = ""
		} else {
			prefix = unselectedStyle.Render(prefix)
		}

		treePane.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, checkbox, prefix, name))
	}

	// Fill remaining height with newlines so the pane height is consistent
	linesRendered := endIdx - m.viewportOffset
	for i := linesRendered; i < visibleLines; i++ {
		treePane.WriteString("\n")
	}

	// Make left pane 50% width
	leftPaneWidth := m.width/2 - 2
	if leftPaneWidth < 20 {
		leftPaneWidth = 20
	}

	leftStyle := lipgloss.NewStyle().Width(leftPaneWidth).PaddingRight(2)

	// Right Pane Header (Breadcrumbs)
	rightHeader := ""
	if len(m.flatNodes) > 0 && m.cursor < len(m.flatNodes) {
		fn := m.flatNodes[m.cursor]
		parts := strings.Split(filepath.ToSlash(fn.Node.Path), "/")
		var breadcrumbs []string
		for i, p := range parts {
			if i == len(parts)-1 && !fn.Node.IsDir {
				breadcrumbs = append(breadcrumbs, "📄 "+p)
			} else {
				breadcrumbs = append(breadcrumbs, "📁 "+p)
			}
		}
		pathStr := strings.Join(breadcrumbs, " / ")

		// Add scroll indicator if it's a file
		if !fn.Node.IsDir && fn.Node.FileNode != nil && !fn.Node.FileNode.IsBinary {
			scrollPct := m.fileViewport.ScrollPercent() * 100
			if math.IsNaN(scrollPct) {
				scrollPct = 0
			}
			pathStr += fmt.Sprintf("   [↕ %3.0f%%]", scrollPct)
		}

		rightHeader = lipgloss.NewStyle().
			Background(lipgloss.Color("#2E4053")).
			Foreground(lipgloss.Color("#FDFEFE")).
			Padding(0, 1).
			Bold(true).
			Render(pathStr) + "\n\n"
	}

	// Create horizontal layout
	rightPane := lipgloss.JoinVertical(lipgloss.Left, rightHeader, m.fileViewport.View())
	middle := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(treePane.String()),
		rightPane,
	)

	var footer strings.Builder
	footer.WriteString("\n")

	// Metrics Dashboard
	cost := float64(m.totalTokens) / 1000000.0 * m.costPer1M
	metrics := fmt.Sprintf("Files: %d | Size: %d bytes | Tokens: %d (~$%.4f)", m.totalSelected, m.totalSize, m.totalTokens, cost)
	footer.WriteString(metricsStyle.Render(metrics) + "\n")

	// Message
	if m.message != "" {
		footer.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F39C12")).Render(m.message) + "\n")
	} else {
		footer.WriteString("\n")
	}

	// Keybinds
	help := "↑/k: up • ↓/j: down • space: toggle • e: edit • s: save as • enter: quick save • c: copy • /: search • q: quit\npgup/pgdn: scroll preview"
	if m.isSearching {
		help = "enter/esc: exit search"
	} else if m.exportStep == 1 {
		help = "enter: confirm filename • esc: cancel export"
	} else if m.exportStep == 2 {
		help = "enter: save file • esc: cancel export"
	} else if m.editMode {
		help = "↑/k: up • ↓/j: down • space: strip/include line • e/esc: exit edit"
	}
	footer.WriteString(keybindStyle.Width(m.width).Render(help))

	return header.String() + middle + footer.String()
}
