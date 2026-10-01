package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"

	"golang.org/x/mod/semver"

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
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"
	"github.com/pkoukk/tiktoken-go"
)

var tkm, _ = tiktoken.GetEncoding("cl100k_base")

// ==========================================
// Layout Configuration (Tweak these!)
// ==========================================
const (
	LayoutLeftPaneWidthRatio  = 2.0 // Divides total width (e.g., 2.0 = 50%)
	LayoutLeftPaneWidthOffset = 2   // Subtracts from the calculated left pane width

	LayoutPanePaddingY    = 0 // Top and Bottom padding inside the panes
	LayoutPanePaddingX    = 1 // Left and Right padding inside the panes
	LayoutGapBetweenPanes = 1 // Horizontal space between the left and right pane

	LayoutBannerMarginTop    = 1 // Space above the banner
	LayoutBannerMarginBottom = 1 // Space below the banner
	LayoutBannerMarginLeft   = 1 // Space to the left of the banner

	LayoutHeaderBottomLines     = 1 // Empty lines between the banner/search-bar and the main containers
	LayoutPaneTitleMarginBottom = 1 // Space between the "root" titles and the container box

	LayoutFooterMarginTop = 1 // Empty lines above the footer block

	LayoutVisibleLinesOffset = 16 // Vertical space reserved for UI (adjust if changing margins)
)

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	exclusiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3"))
	directoryStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498DB")).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7698")).Bold(true)
	metricsStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
	keybindStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
)

type UINode struct {
	Name        string
	Path        string
	IsDir       bool
	Selected    bool
	Expanded    bool
	IsIgnored   bool
	IsExclusive bool
	Children    map[string]*UINode
	FileNode    *walker.FileNode
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
	maxTreeWidth   int

	totalSelected int
	totalSize     int64
	totalTokens   int
	projectSize   int64
	projectFiles  int
	message       string
	newVersion    string

	searchInput    textinput.Model
	isSearching    bool
	exportInput    textinput.Model
	exportStep     int // 0: none, 1: filename, 2: directory
	exportFileName string
	quitting       bool
	fileViewport   viewport.Model

	editMode    bool
	editCursor  int
	minimalMode bool

	lastSpacedNode *UINode
	lastSpaceTime  time.Time
	showHelp       bool
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
					Selected:  !f.IsIgnored,                                  // Select by default unless ignored
					Expanded:  !f.IsIgnored && !strings.HasPrefix(part, "."), // Collapse ignored and hidden directories by default
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

	var pSize int64
	var pFiles int
	for i := range files {
		if !files[i].IsDir {
			pSize += files[i].Size
			pFiles++
		}
	}

	m := &Model{
		root:         root,
		templateStr:  templateStr,
		persona:      persona,
		theme:        theme,
		defaultOut:   defaultOut,
		costPer1M:    costPer1M,
		searchInput:  ti,
		exportInput:  ei,
		projectSize:  pSize,
		projectFiles: pFiles,
	}

	vp := viewport.New(40, 20)
	m.fileViewport = vp

	m.updateFlatNodes()
	m.updateMetrics()
	m.updateViewportContent()
	return m
}

func (m *Model) updateFlatNodes() {
	var flat []*flatNode
	var maxW int

	var traverse func(node *UINode, drawingPrefix string, childPrefix string, level int) bool
	traverse = func(node *UINode, drawingPrefix string, childPrefix string, level int) bool {
		// If searching, check if this node or any child matches
		rawQuery := m.searchInput.Value()
		queries := parseQueries(rawQuery)

		matches := true
		if len(queries) > 0 {
			matches = matchesAnyQuery(node.Name, node.IsDir, queries)
		}

		// Pre-calculate which children match if we are searching, to know if this dir should be shown
		var matchingChildren []string
		if node.IsDir {
			for k, v := range node.Children {
				if len(queries) == 0 || matchesAnyQuery(k, v.IsDir, queries) || hasMatchingChild(v, queries) {
					matchingChildren = append(matchingChildren, k)
				}
			}
			if len(matchingChildren) > 0 {
				matches = true // Show this dir if any child matches
			}
		}

		if !matches && len(queries) > 0 {
			return false
		}

		lineW := 6 + runewidth.StringWidth(drawingPrefix) + runewidth.StringWidth(node.Name)
		if node.IsDir {
			lineW += 4
		}
		if node.IsIgnored {
			lineW += 10
		}
		if lineW > maxW {
			maxW = lineW
		}

		flat = append(flat, &flatNode{
			Node:   node,
			Prefix: drawingPrefix,
			Level:  level,
		})

		if node.IsDir && node.Expanded {
			var names []string
			if len(queries) > 0 {
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
	m.maxTreeWidth = maxW
}

func parseQueries(raw string) []string {
	raw = strings.ToLower(raw)
	var parts []string
	if strings.Contains(raw, "+") {
		parts = strings.Split(raw, "+")
	} else if strings.Contains(raw, ",") {
		parts = strings.Split(raw, ",")
	} else if strings.Contains(raw, "|") {
		parts = strings.Split(raw, "|")
	} else {
		parts = []string{raw}
	}

	var queries []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			queries = append(queries, p)
		}
	}
	return queries
}

func matchesAnyQuery(name string, isDir bool, queries []string) bool {
	if len(queries) == 0 {
		return true
	}
	lowerName := strings.ToLower(name)
	for _, q := range queries {
		// If query starts with a dot and it's a file, do a strict extension or dotfile match
		if strings.HasPrefix(q, ".") && !isDir {
			if strings.HasSuffix(lowerName, q) || strings.HasPrefix(lowerName, q) {
				return true
			}
		} else {
			if strings.Contains(lowerName, q) {
				return true
			}
		}
	}
	return false
}

func hasMatchingChild(node *UINode, queries []string) bool {
	if matchesAnyQuery(node.Name, node.IsDir, queries) {
		return true
	}
	for _, child := range node.Children {
		if hasMatchingChild(child, queries) {
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

func (m *Model) toggleSelection(node *UINode, state bool, exclusive bool) {
	node.Selected = state
	node.IsExclusive = exclusive
	for _, child := range node.Children {
		m.toggleSelection(child, state, exclusive)
	}
}

func (m *Model) clearExclusiveFlags(node *UINode) {
	node.IsExclusive = false
	for _, child := range node.Children {
		m.clearExclusiveFlags(child)
	}
}

func (m *Model) clearAllSelections(node *UINode) {
	node.Selected = false
	node.IsExclusive = false
	for _, child := range node.Children {
		m.clearAllSelections(child)
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

	if rel.TagName != "" && semver.Compare(rel.TagName, version.Version) > 0 {
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
			case "enter":
				// Quick Export! Unselect everything first
				m.clearAllSelections(m.root)

				// Select only the currently filtered visible files
				for _, fn := range m.flatNodes {
					if !fn.Node.IsDir {
						fn.Node.Selected = true
					}
				}
				m.updateMetrics()

				m.isSearching = false
				m.searchInput.Blur()

				// Jump directly to export
				m.exportStep = 1
				m.exportInput.Focus()
				return m, textinput.Blink
			case "esc":
				m.isSearching = false
				m.searchInput.Blur()
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.flatNodes)-1 {
					m.cursor++
				}
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

		case "h", "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "esc":
			if m.showHelp {
				m.showHelp = false
			} else {
				m.message = ""
			}
			return m, nil

		case "/":
			m.isSearching = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "m":
			m.minimalMode = !m.minimalMode
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
			}

		case "left":
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

				now := time.Now()
				isDoubleTap := false
				if m.lastSpacedNode == node && now.Sub(m.lastSpaceTime) < 500*time.Millisecond {
					isDoubleTap = true
				}

				m.lastSpacedNode = node
				m.lastSpaceTime = now

				if isDoubleTap {
					// Exclusive Select
					m.clearAllSelections(m.root)
					m.toggleSelection(node, true, true)
					m.message = "Exclusively Selected"
				} else {
					// Normal toggle
					m.clearExclusiveFlags(m.root) // clear any previous exclusive state
					newState := !node.Selected
					m.toggleSelection(node, newState, false)
					if newState {
						m.message = "Selected"
					} else {
						m.message = "Deselected"
					}
				}

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
	}

	// Basic viewport handling
	visibleLines := m.height - LayoutVisibleLinesOffset
	if m.minimalMode {
		visibleLines = m.height - 6
		if m.isSearching || m.exportStep > 0 {
			visibleLines = m.height - 9
		}
	}
	if visibleLines < 5 {
		visibleLines = 5
	}

	maxAllowedLeftPaneWidth := int(float64(m.width)/LayoutLeftPaneWidthRatio) - LayoutLeftPaneWidthOffset
	leftPaneWidth := m.maxTreeWidth + (LayoutPanePaddingX * 2)
	if leftPaneWidth > maxAllowedLeftPaneWidth {
		leftPaneWidth = maxAllowedLeftPaneWidth
	}
	if leftPaneWidth < 20 {
		leftPaneWidth = 20
	}
	m.fileViewport.Height = visibleLines
	m.fileViewport.Width = m.width - leftPaneWidth - 4 - LayoutGapBetweenPanes - (LayoutPanePaddingX * 2)
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
					var content string
					if err == nil && buf.Len() > 0 {
						content = buf.String()
					} else {
						content = code
					}

					if m.fileViewport.Width > 0 {
						// Hard wrap unbroken words first, then let lipgloss handle normal word-wrapping
						content = wrap.String(content, m.fileViewport.Width)
						content = lipgloss.NewStyle().Width(m.fileViewport.Width).Render(content)
					}
					m.fileViewport.SetContent(content)
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
		visibleLines := m.height - LayoutVisibleLinesOffset
		if m.minimalMode {
			visibleLines = m.height - 6
			if m.isSearching || m.exportStep > 0 {
				visibleLines = m.height - 9
			}
		}
		if visibleLines < 5 {
			visibleLines = 5
		}

		maxAllowedLeftPaneWidth := int(float64(m.width)/LayoutLeftPaneWidthRatio) - LayoutLeftPaneWidthOffset
		leftPaneWidth := m.maxTreeWidth
		if leftPaneWidth > maxAllowedLeftPaneWidth {
			leftPaneWidth = maxAllowedLeftPaneWidth
		}
		if leftPaneWidth < 20 {
			leftPaneWidth = 20
		}
		m.fileViewport.Height = visibleLines
		m.fileViewport.Width = m.width - leftPaneWidth - 4 - LayoutGapBetweenPanes
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

	content := b.String()
	if m.fileViewport.Width > 0 {
		content = wrap.String(content, m.fileViewport.Width)
		content = lipgloss.NewStyle().Width(m.fileViewport.Width).Render(content)
	}
	m.fileViewport.SetContent(content)
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

	if m.showHelp {
		return m.renderHelpScreen()
	}

	var header strings.Builder
	if !m.minimalMode {
		header.WriteString(m.renderBanner())
		if m.newVersion != "" {
			updateMsg := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF7698")).
				Bold(true).
				MarginBottom(1).
				Render(fmt.Sprintf("🚀 Update available: %s! (Run 'npm i -g repowalk')", m.newVersion))
			header.WriteString(updateMsg + "\n")
		}
	}

	if m.isSearching {
		header.WriteString("\n" + m.searchInput.View() + strings.Repeat("\n", LayoutHeaderBottomLines))
	} else if m.exportStep == 1 {
		header.WriteString("\nExport Filename: " + m.exportInput.View() + strings.Repeat("\n", LayoutHeaderBottomLines))
	} else if m.exportStep == 2 {
		header.WriteString(fmt.Sprintf("\nExport Filename: %s\nExport Directory: %s%s", lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(m.exportFileName), m.exportInput.View(), strings.Repeat("\n", LayoutHeaderBottomLines)))
	} else if !m.minimalMode {
		header.WriteString(strings.Repeat("\n", LayoutHeaderBottomLines))
	}

	// Adjusted for modular banner height
	visibleLines := m.height - LayoutVisibleLinesOffset
	if m.minimalMode {
		visibleLines = m.height - 6
		if m.isSearching || m.exportStep > 0 {
			visibleLines = m.height - 9
		}
	}
	if visibleLines < 5 {
		visibleLines = 5
	}

	endIdx := m.viewportOffset + visibleLines
	if endIdx > len(m.flatNodes) {
		endIdx = len(m.flatNodes)
	}

	// Make left pane width dynamic based on configuration and longest item
	maxAllowedLeftPaneWidth := int(float64(m.width)/LayoutLeftPaneWidthRatio) - LayoutLeftPaneWidthOffset
	leftPaneWidth := m.maxTreeWidth + (LayoutPanePaddingX * 2)
	if leftPaneWidth > maxAllowedLeftPaneWidth {
		leftPaneWidth = maxAllowedLeftPaneWidth
	}
	if leftPaneWidth < 20 {
		leftPaneWidth = 20
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
			if fn.Node.IsExclusive {
				checkbox = exclusiveStyle.Render("[*]")
			} else {
				checkbox = selectedStyle.Render("[x]")
			}
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

		line := fmt.Sprintf("%s%s %s%s", cursor, checkbox, prefix, name)
		leftPaneInnerWidth := leftPaneWidth - (LayoutPanePaddingX * 2)
		if leftPaneInnerWidth < 0 {
			leftPaneInnerWidth = 0
		}
		line = truncateString(line, leftPaneInnerWidth)
		treePane.WriteString(line + "\n")
	}

	// Fill remaining height with newlines so the pane height is consistent
	linesRendered := endIdx - m.viewportOffset
	for i := linesRendered; i < visibleLines; i++ {
		treePane.WriteString("\n")
	}

	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(LayoutPanePaddingY, LayoutPanePaddingX)

	// Left Pane Header (Breadcrumbs of selected item)
	pathStr := ""
	if len(m.flatNodes) > 0 && m.cursor < len(m.flatNodes) {
		fn := m.flatNodes[m.cursor]
		parts := strings.Split(filepath.ToSlash(fn.Node.Path), "/")
		var breadcrumbs []string
		for i, p := range parts {
			if i == len(parts)-1 && !fn.Node.IsDir {
				breadcrumbs = append(breadcrumbs, "📄 "+p)
			} else if p != "." && p != "" {
				breadcrumbs = append(breadcrumbs, "📁 "+p)
			} else if p == "." {
				breadcrumbs = append(breadcrumbs, "📁 root")
			}
		}
		pathStr = strings.Join(breadcrumbs, " / ")
	}
	pathStr = truncateString(pathStr, leftPaneWidth+2)

	leftHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		MarginBottom(LayoutPaneTitleMarginBottom).
		Render(pathStr)

	rightPaneWidth := m.width - leftPaneWidth - 4 - LayoutGapBetweenPanes

	// Right Pane Header
	rightHeaderStr := ""
	if len(m.flatNodes) > 0 && m.cursor < len(m.flatNodes) {
		fn := m.flatNodes[m.cursor]
		if !fn.Node.IsDir {
			rightHeaderStr = "📄 " + fn.Node.Name
			if fn.Node.FileNode != nil && !fn.Node.FileNode.IsBinary {
				scrollPct := m.fileViewport.ScrollPercent() * 100
				if math.IsNaN(scrollPct) {
					scrollPct = 0
				}
				rightHeaderStr += fmt.Sprintf("   [↕ %3.0f%%]", scrollPct)
			}
		} else {
			rightHeaderStr = "📁 [Directory]"
		}
	}
	rightHeaderStr = truncateString(rightHeaderStr, rightPaneWidth+2)

	rightHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true).
		MarginBottom(LayoutPaneTitleMarginBottom).
		Render(rightHeaderStr)

	treePaneStr := panelStyle.Width(leftPaneWidth).Height(visibleLines).Render(strings.TrimRight(treePane.String(), "\n"))
	viewportPaneStr := panelStyle.Width(rightPaneWidth).Height(visibleLines).Render(m.fileViewport.View())

	leftPane := lipgloss.JoinVertical(lipgloss.Left, leftHeader, treePaneStr)
	rightPane := lipgloss.JoinVertical(lipgloss.Left, rightHeader, viewportPaneStr)

	middle := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPane,
		lipgloss.NewStyle().PaddingLeft(LayoutGapBetweenPanes).Render(rightPane),
	)

	var footer strings.Builder

	if !m.minimalMode {
		footer.WriteString(strings.Repeat("\n", LayoutFooterMarginTop))

		// Metrics Dashboard
		cost := float64(m.totalTokens) / 1000000.0 * m.costPer1M
		metrics := fmt.Sprintf("Files: %d/%d | Size: %s/%s | Tokens: %d (~$%.4f)", 
			m.totalSelected, m.projectFiles, 
			formatSize(m.totalSize), formatSize(m.projectSize), 
			m.totalTokens, cost)
		footer.WriteString(metricsStyle.Render(metrics) + "\n")

		// Message
		if m.message != "" {
			footer.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F39C12")).Render(m.message) + "\n")
		} else {
			footer.WriteString("\n")
		}

		// Keybinds
		help := "↑/k: up • ↓/j: down • space: toggle • e: edit • s: save as • enter: quick save • c: copy • /: search • m: UI • q: quit\npgup/pgdn: scroll preview"
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
	} else if m.message != "" {
		footer.WriteString("\n")
		footer.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F39C12")).Render(m.message) + "\n")
	}

	return header.String() + middle + footer.String()
}

func (m *Model) renderBanner() string {
	ascii := []string{
		"█▀█ █▀▀ █▀█ █▀█ █ █ █ ▄▀█ █░  █▄▀",
		"█▀▄ ██▄ █▀▀ █▄█ ▀▄▀▄▀ █▀█ █▄▄ █ █",
	}

	var banner strings.Builder
	for _, line := range ascii {
		runes := []rune(line)
		for i, r := range runes {
			pct := float64(i) / float64(len(runes)-1)
			var color string
			if pct < 0.5 {
				color = interpolateColor("#04B575", "#3498DB", pct*2)
			} else {
				color = interpolateColor("#3498DB", "#7D56F4", (pct-0.5)*2)
			}
			banner.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(string(r)))
		}
		banner.WriteString("\n")
	}
	return lipgloss.NewStyle().
		MarginTop(LayoutBannerMarginTop).
		MarginBottom(LayoutBannerMarginBottom).
		MarginLeft(LayoutBannerMarginLeft).
		Render(banner.String())
}

func interpolateColor(start, end string, percent float64) string {
	r1, g1, b1 := hexToRGB(start)
	r2, g2, b2 := hexToRGB(end)

	r := int(float64(r1) + percent*float64(r2-r1))
	g := int(float64(g1) + percent*float64(g2-g1))
	b := int(float64(b1) + percent*float64(b2-b1))

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func (m *Model) renderHelpScreen() string {
	keys := []string{
		"↑ / k", "↓ / j", "←", "→ / l", "Space", "e", "s", "Enter", "c", "/", "m", "h / ?", "q / esc", "pgup/pgdn",
	}
	actions := []string{
		"Move up", "Move down", "Collapse folder", "Expand folder", "Toggle select (Double-tap for Exclusive)",
		"Edit file contents (Exclude/Include lines)", "Save As (Export to custom path)", "Quick save / Quick export",
		"Copy output to clipboard", "Search (Multi-term: .go + yaml)", "Toggle Minimal UI", "Toggle this help screen",
		"Quit", "Scroll file preview",
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Width(15).Align(lipgloss.Right).PaddingRight(2)
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))

	var rows []string
	for i := range keys {
		row := lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(keys[i]), actionStyle.Render(actions[i]))
		rows = append(rows, row)
	}

	table := lipgloss.JoinVertical(lipgloss.Left, rows...)

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginBottom(2).Render("RepoWalk Keyboard Shortcuts")
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(2).Render("Press 'h' or 'esc' to close this menu.")

	content := lipgloss.JoinVertical(lipgloss.Center, title, table, footer)

	// Center vertically and horizontally
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if lipgloss.Width(s) > maxLen {
		return truncate.StringWithTail(s, uint(maxLen), "...")
	}
	return s
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
