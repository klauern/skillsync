// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/sync"
)

// ConflictAction represents the action to perform after conflict resolution.
type ConflictAction int

const (
	// ConflictActionNone means no action was taken (user quit).
	ConflictActionNone ConflictAction = iota
	// ConflictActionResolve means the user resolved conflicts and wants to apply.
	ConflictActionResolve
	// ConflictActionCancel means the user cancelled.
	ConflictActionCancel
)

// ConflictResolution holds the resolution for a single conflict.
type ConflictResolution struct {
	SkillName  string
	Resolution sync.ResolutionChoice
	Content    string // The resolved content (relevant for merge)
}

// ConflictListResult contains the result of the conflict resolution interaction.
type ConflictListResult struct {
	Action      ConflictAction
	Resolutions []ConflictResolution
}

// conflictPhase represents the current phase of conflict resolution.
type conflictPhase int

const (
	phaseList conflictPhase = iota
	phaseDetail
)

// conflictKeyMap defines the key bindings for conflict resolution.
type conflictKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Source   key.Binding
	Target   key.Binding
	Merge    key.Binding
	Skip     key.Binding
	Confirm  key.Binding
	Back     key.Binding
	Help     key.Binding
	Quit     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

func defaultConflictKeyMap() conflictKeyMap {
	return conflictKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view details"),
		),
		Source: key.NewBinding(
			key.WithKeys("s", "1"),
			key.WithHelp("s/1", "use source"),
		),
		Target: key.NewBinding(
			key.WithKeys("t", "2"),
			key.WithHelp("t/2", "use target"),
		),
		Merge: key.NewBinding(
			key.WithKeys("m", "3"),
			key.WithHelp("m/3", "merge"),
		),
		Skip: key.NewBinding(
			key.WithKeys("x", "4"),
			key.WithHelp("x/4", "skip"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "apply resolutions"),
		),
		Back: key.NewBinding(
			key.WithKeys("b", "esc"),
			key.WithHelp("b/esc", "back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
	}
}

// ConflictListModel is the BubbleTea model for conflict resolution.
type ConflictListModel struct {
	conflicts   []*sync.Conflict
	resolutions map[string]sync.ResolutionChoice
	table       table.Model
	viewport    viewport.Model
	keys        conflictKeyMap
	result      ConflictListResult
	phase       conflictPhase
	cursor      int
	showHelp    bool
	confirmMode bool
	width       int
	height      int
	quitting    bool
	ready       bool
}

// Styles for the conflict resolution TUI.
var conflictStyles = struct {
	Title        lipgloss.Style
	Help         lipgloss.Style
	Status       lipgloss.Style
	Header       lipgloss.Style
	Added        lipgloss.Style
	Removed      lipgloss.Style
	Context      lipgloss.Style
	Info         lipgloss.Style
	Warning      lipgloss.Style
	Resolved     lipgloss.Style
	Unresolved   lipgloss.Style
	HunkHeader   lipgloss.Style
	Confirm      lipgloss.Style
	SourceLabel  lipgloss.Style
	TargetLabel  lipgloss.Style
	SectionTitle lipgloss.Style
}{
	Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Status:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Header:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
	Added:        lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	Removed:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	Context:      lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
	Info:         lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true),
	Warning:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Resolved:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	Unresolved:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	HunkHeader:   lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Confirm:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(0, 1),
	SourceLabel:  lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),
	TargetLabel:  lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	SectionTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(1, 0),
}

// formatConflictContentWithLineNumbers formats content with line numbers for display.
func formatConflictContentWithLineNumbers(content string, style lipgloss.Style) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	for i, line := range lines {
		lineNum := fmt.Sprintf("%4d │ ", i+1)
		b.WriteString(conflictStyles.Context.Render(lineNum))
		b.WriteString(style.Render(line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// NewConflictListModel creates a new conflict resolution model.
func NewConflictListModel(conflicts []*sync.Conflict) ConflictListModel {
	resolutions := make(map[string]sync.ResolutionChoice)

	// Build table columns
	columns := []table.Column{
		{Title: "Status", Width: 8},
		{Title: "Skill Name", Width: 25},
		{Title: "Type", Width: 12},
		{Title: "Changes", Width: 20},
		{Title: "Resolution", Width: 12},
	}

	// Build table rows
	rows := make([]table.Row, len(conflicts))
	for i, c := range conflicts {
		rows[i] = buildConflictRow(c, "")
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return ConflictListModel{
		conflicts:   conflicts,
		resolutions: resolutions,
		table:       t,
		keys:        defaultConflictKeyMap(),
		phase:       phaseList,
	}
}

func buildConflictRow(c *sync.Conflict, resolution string) table.Row {
	status := "○"
	if resolution != "" {
		status = "✓"
	}

	resStr := "-"
	if resolution != "" {
		resStr = resolution
	}

	return table.Row{
		status,
		c.SkillName,
		string(c.Type),
		c.DiffSummary(),
		resStr,
	}
}

// Init implements tea.Model.
func (m ConflictListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ConflictListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseList:
		return m.updateList(msg)
	case phaseDetail:
		return m.updateDetail(msg)
	}
	return m, nil
}

func (m ConflictListModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		newHeight := max(msg.Height-10, 5)
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode first
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = ConflictListResult{
					Action:      ConflictActionResolve,
					Resolutions: m.buildResolutions(),
				}
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Select):
			if len(m.conflicts) > 0 {
				m.cursor = m.table.Cursor()
				m.phase = phaseDetail
				m.ready = false
				return m, nil
			}

		case key.Matches(msg, m.keys.Source):
			if len(m.conflicts) > 0 {
				m.resolveCurrentConflict(sync.ResolutionUseSource)
				return m, nil
			}

		case key.Matches(msg, m.keys.Target):
			if len(m.conflicts) > 0 {
				m.resolveCurrentConflict(sync.ResolutionUseTarget)
				return m, nil
			}

		case key.Matches(msg, m.keys.Merge):
			if len(m.conflicts) > 0 {
				m.resolveCurrentConflict(sync.ResolutionMerge)
				return m, nil
			}

		case key.Matches(msg, m.keys.Skip):
			if len(m.conflicts) > 0 {
				m.resolveCurrentConflict(sync.ResolutionSkip)
				return m, nil
			}

		case key.Matches(msg, m.keys.Confirm):
			if m.allResolved() {
				m.confirmMode = true
				return m, nil
			}

		case key.Matches(msg, m.keys.Back):
			m.result = ConflictListResult{Action: ConflictActionCancel}
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m ConflictListModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 6
		footerHeight := 4
		viewportHeight := max(msg.Height-headerHeight-footerHeight, 5)

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, viewportHeight)
			m.viewport.SetContent(m.buildDetailContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = viewportHeight
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Back):
			m.phase = phaseList
			return m, nil

		case key.Matches(msg, m.keys.Source):
			m.resolveConflictAt(m.cursor, sync.ResolutionUseSource)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil

		case key.Matches(msg, m.keys.Target):
			m.resolveConflictAt(m.cursor, sync.ResolutionUseTarget)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil

		case key.Matches(msg, m.keys.Merge):
			m.resolveConflictAt(m.cursor, sync.ResolutionMerge)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil

		case key.Matches(msg, m.keys.Skip):
			m.resolveConflictAt(m.cursor, sync.ResolutionSkip)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ConflictListModel) resolveCurrentConflict(resolution sync.ResolutionChoice) {
	cursor := m.table.Cursor()
	m.resolveConflictAt(cursor, resolution)
}

func (m *ConflictListModel) resolveConflictAt(idx int, resolution sync.ResolutionChoice) {
	if idx < 0 || idx >= len(m.conflicts) {
		return
	}

	c := m.conflicts[idx]
	m.resolutions[c.SkillName] = resolution

	// Update the table row
	m.updateTableRow(idx)
}

func (m *ConflictListModel) updateTableRow(idx int) {
	if idx < 0 || idx >= len(m.conflicts) {
		return
	}

	c := m.conflicts[idx]
	resolution := ""
	if res, ok := m.resolutions[c.SkillName]; ok {
		resolution = string(res)
	}

	rows := m.table.Rows()
	if idx < len(rows) {
		rows[idx] = buildConflictRow(c, resolution)
		m.table.SetRows(rows)
	}
}

func (m ConflictListModel) allResolved() bool {
	for _, c := range m.conflicts {
		if _, ok := m.resolutions[c.SkillName]; !ok {
			return false
		}
	}
	return len(m.conflicts) > 0
}

func (m ConflictListModel) buildResolutions() []ConflictResolution {
	var result []ConflictResolution
	for _, c := range m.conflicts {
		if res, ok := m.resolutions[c.SkillName]; ok {
			content := ""
			switch res {
			case sync.ResolutionUseSource:
				content = c.Source.Content
			case sync.ResolutionUseTarget:
				content = c.Target.Content
			case sync.ResolutionMerge:
				// For merge, we'd need to invoke the merger
				// For now, use source as fallback
				content = c.Source.Content
			}
			result = append(result, ConflictResolution{
				SkillName:  c.SkillName,
				Resolution: res,
				Content:    content,
			})
		}
	}
	return result
}

func (m ConflictListModel) buildDetailContent() string {
	if m.cursor < 0 || m.cursor >= len(m.conflicts) {
		return "No conflict selected"
	}

	c := m.conflicts[m.cursor]
	var b strings.Builder

	// Conflict summary
	b.WriteString(conflictStyles.SectionTitle.Render("Conflict Details"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Skill: %s\n", c.SkillName))
	b.WriteString(fmt.Sprintf("  Type:  %s\n", c.Type))
	b.WriteString(fmt.Sprintf("  %s\n", c.DiffSummary()))

	// Current resolution
	if res, ok := m.resolutions[c.SkillName]; ok {
		b.WriteString("\n")
		b.WriteString(conflictStyles.Resolved.Render(fmt.Sprintf("  Resolution: %s", res)))
		b.WriteString("\n")
	}

	// Diff hunks
	if len(c.Hunks) > 0 {
		b.WriteString("\n")
		b.WriteString(conflictStyles.SectionTitle.Render("Changes"))
		b.WriteString("\n")

		for i, hunk := range c.Hunks {
			header := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				hunk.SourceStart, hunk.SourceCount,
				hunk.TargetStart, hunk.TargetCount)
			b.WriteString(conflictStyles.HunkHeader.Render(header))
			b.WriteString("\n")

			for _, line := range hunk.Lines {
				var styled string
				switch line.Type {
				case sync.DiffLineAdded:
					styled = conflictStyles.Added.Render("+" + line.Content)
				case sync.DiffLineRemoved:
					styled = conflictStyles.Removed.Render("-" + line.Content)
				default:
					styled = conflictStyles.Context.Render(" " + line.Content)
				}
				b.WriteString(styled)
				b.WriteString("\n")
			}

			if i < len(c.Hunks)-1 {
				b.WriteString("\n")
			}
		}
	} else {
		// No hunks, show full content comparison
		b.WriteString("\n")
		b.WriteString(conflictStyles.SectionTitle.Render("Source Content"))
		b.WriteString("\n")
		b.WriteString(formatConflictContentWithLineNumbers(c.Source.Content, conflictStyles.Removed))
		b.WriteString("\n\n")

		b.WriteString(conflictStyles.SectionTitle.Render("Target Content"))
		b.WriteString("\n")
		b.WriteString(formatConflictContentWithLineNumbers(c.Target.Content, conflictStyles.Added))
	}

	// Resolution options reminder
	b.WriteString("\n\n")
	b.WriteString(conflictStyles.Info.Render("Press: s=source, t=target, m=merge, x=skip"))

	return b.String()
}

// View implements tea.Model.
func (m ConflictListModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.phase {
	case phaseDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

func (m ConflictListModel) viewList() string {
	var b strings.Builder

	// Title
	title := conflictStyles.Title.Render("⚠️  Resolve Conflicts")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Info message
	info := conflictStyles.Info.Render("Select a resolution for each conflict before applying")
	b.WriteString(info)
	b.WriteString("\n\n")

	// Confirmation dialog
	if m.confirmMode {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		confirmMsg := fmt.Sprintf("Apply %d resolution(s)? (y/n)", len(m.resolutions))
		b.WriteString(conflictStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	resolved := len(m.resolutions)
	total := len(m.conflicts)
	status := fmt.Sprintf("%d/%d resolved", resolved, total)
	if resolved == total && total > 0 {
		status += " • Press y to apply"
	}
	b.WriteString(conflictStyles.Status.Render(status))
	b.WriteString("\n")

	// Help
	if m.showHelp {
		help := m.renderFullHelp()
		b.WriteString("\n")
		b.WriteString(help)
	} else {
		help := m.renderShortHelp()
		b.WriteString(help)
	}

	return b.String()
}

func (m ConflictListModel) viewDetail() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	skillName := ""
	if m.cursor >= 0 && m.cursor < len(m.conflicts) {
		skillName = m.conflicts[m.cursor].SkillName
	}
	title := conflictStyles.Title.Render(fmt.Sprintf("📄 Conflict: %s", skillName))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Status bar
	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	status := fmt.Sprintf("Scroll: %d%%", scrollPercent)
	b.WriteString(conflictStyles.Status.Render(status))
	b.WriteString("\n")

	// Help
	if m.showHelp {
		help := m.renderDetailHelp()
		b.WriteString("\n")
		b.WriteString(help)
	} else {
		help := m.renderDetailShortHelp()
		b.WriteString(help)
	}

	return b.String()
}

func (m ConflictListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter details",
		"s source",
		"t target",
		"m merge",
		"x skip",
		"? help",
		"q quit",
	}
	return conflictStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ConflictListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  Enter    View conflict details

Resolution:
  s/1      Use source version
  t/2      Use target version
  m/3      Merge both versions
  x/4      Skip this conflict

Actions:
  y        Apply all resolutions
  b/Esc    Cancel and go back

General:
  ?        Toggle full help
  q        Quit`
	return conflictStyles.Help.Render(help)
}

func (m ConflictListModel) renderDetailShortHelp() string {
	keys := []string{
		"↑/↓ scroll",
		"s source",
		"t target",
		"m merge",
		"x skip",
		"b back",
		"? help",
	}
	return conflictStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ConflictListModel) renderDetailHelp() string {
	help := `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  PgUp     Page up
  PgDown   Page down

Resolution:
  s/1      Use source version
  t/2      Use target version
  m/3      Merge both versions
  x/4      Skip this conflict

Actions:
  b/Esc    Go back to list

General:
  ?        Toggle full help
  q        Quit`
	return conflictStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m ConflictListModel) Result() ConflictListResult {
	return m.result
}

// RunConflictList runs the interactive conflict resolution and returns the result.
func RunConflictList(conflicts []*sync.Conflict) (ConflictListResult, error) {
	if len(conflicts) == 0 {
		return ConflictListResult{}, nil
	}

	mdl := NewConflictListModel(conflicts)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ConflictListResult{}, err
	}

	if m, ok := finalModel.(ConflictListModel); ok {
		return m.Result(), nil
	}

	return ConflictListResult{}, nil
}
