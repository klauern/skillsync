// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"sort"
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
	ListModel[*sync.Conflict]
	resolutions map[string]sync.ResolutionChoice
	viewport    viewport.Model
	keys        conflictKeyMap
	phase       conflictPhase
	cursor      int
	detail      *sync.Conflict
	detailReady bool
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

	sort.Slice(conflicts, func(i, j int) bool {
		return strings.ToLower(conflicts[i].SkillName) < strings.ToLower(conflicts[j].SkillName)
	})

	cfg := ListConfig[*sync.Conflict]{
		Title: "⚠️  Resolve Conflicts",
		Columns: []table.Column{
			{Title: "Status", Width: 8},
			{Title: "Skill Name", Width: 25},
			{Title: "Type", Width: 12},
			{Title: "Changes", Width: 20},
			{Title: "Resolution", Width: 12},
		},
		ToRows: func(items []*sync.Conflict) []table.Row {
			rows := make([]table.Row, len(items))
			for i, c := range items {
				resolution := ""
				if res, ok := resolutions[c.SkillName]; ok {
					resolution = string(res)
				}
				rows[i] = buildConflictRow(c, resolution)
			}
			return rows
		},
		Matches: func(conflict *sync.Conflict, lf string) bool {
			if lf == "" {
				return true
			}
			return strings.Contains(strings.ToLower(conflict.SkillName), lf) ||
				strings.Contains(strings.ToLower(string(conflict.Type)), lf) ||
				strings.Contains(strings.ToLower(conflict.DiffSummary()), lf)
		},
		ReservedLines: 10,
		Header: func() string {
			return conflictStyles.Info.Render("Select a resolution for each conflict before applying")
		},
		StatusText: func(filtered, total int, filter string) string {
			status := fmt.Sprintf("%d/%d resolved", len(resolutions), total)
			if len(resolutions) == total && total > 0 {
				status += " • Press y to apply"
			}
			if filter != "" {
				status = fmt.Sprintf("%d of %d conflict(s) shown • %s", filtered, total, status)
			}
			return status
		},
		ShortHelp: func() string {
			return strings.Join([]string{
				"↑/↓ navigate",
				"enter details",
				"s source",
				"t target",
				"m merge",
				"x skip",
				"/ filter",
				"b cancel",
				"? help",
				"q quit",
			}, " • ")
		},
		FullHelp: func() string {
			return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  Enter    View conflict details

Resolution:
  s/1      Use source version
  t/2      Use target version
  m/3      Merge both versions
  x/4      Skip this conflict
  y        Apply all resolutions

Filter:
  /        Start filtering
  Esc      Clear filter
  Enter    Finish filtering

Actions:
  b        Cancel and go back

General:
  ?        Toggle full help
  q        Quit`
		},
	}

	return ConflictListModel{
		ListModel:   NewListModel(conflicts, cfg),
		resolutions: resolutions,
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
		truncateTableValue(c.SkillName, 25),
		truncateTableValue(string(c.Type), 12),
		truncateTableValue(c.DiffSummary(), 20),
		truncateTableValue(resStr, 12),
	}
}

// Update implements tea.Model.
func (m ConflictListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseDetail:
		return m.updateDetail(msg)
	case phaseList:
		if m.confirmMode {
			return m.updateConfirm(msg)
		}
		if handled, model, cmd := m.updateList(msg); handled {
			return model, cmd
		}
	}

	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[*sync.Conflict])
	return m, cmd
}

func (m ConflictListModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		m.result = ConflictListResult{
			Action:      ConflictActionResolve,
			Resolutions: m.buildResolutions(),
		}
		m.quitting = true
		return m, tea.Quit
	case "n", "N", "esc":
		m.confirmMode = false
		m.confirmMsg = ""
		return m, nil
	default:
		return m, nil
	}
}

func (m ConflictListModel) updateList(msg tea.Msg) (bool, tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return false, nil, nil
	}

	switch {
	case key.Matches(keyMsg, m.keys.Select):
		if selected := m.selectedConflict(); selected != nil {
			m.cursor = m.table.Cursor()
			m.detail = selected
			m.phase = phaseDetail
			m.detailReady = false
			return true, m, nil
		}
	case key.Matches(keyMsg, m.keys.Source):
		m.resolveCurrentConflict(sync.ResolutionUseSource)
		return true, m, nil
	case key.Matches(keyMsg, m.keys.Target):
		m.resolveCurrentConflict(sync.ResolutionUseTarget)
		return true, m, nil
	case key.Matches(keyMsg, m.keys.Merge):
		m.resolveCurrentConflict(sync.ResolutionMerge)
		return true, m, nil
	case key.Matches(keyMsg, m.keys.Skip):
		m.resolveCurrentConflict(sync.ResolutionSkip)
		return true, m, nil
	case key.Matches(keyMsg, m.keys.Confirm):
		if m.allResolved() {
			m.confirmMode = true
			m.confirmMsg = fmt.Sprintf("Apply %d resolution(s)? (y/n)", len(m.resolutions))
			return true, m, nil
		}
	case key.Matches(keyMsg, m.keys.Back):
		m.result = ConflictListResult{Action: ConflictActionCancel}
		m.quitting = true
		return true, m, tea.Quit
	}

	return false, nil, nil
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

		if !m.detailReady {
			m.viewport = viewport.New(msg.Width-2, viewportHeight)
			m.viewport.SetContent(m.buildDetailContent())
			m.detailReady = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = viewportHeight
			m.viewport.SetContent(m.buildDetailContent())
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
			m.resolveDetailConflict(sync.ResolutionUseSource)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil
		case key.Matches(msg, m.keys.Target):
			m.resolveDetailConflict(sync.ResolutionUseTarget)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil
		case key.Matches(msg, m.keys.Merge):
			m.resolveDetailConflict(sync.ResolutionMerge)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil
		case key.Matches(msg, m.keys.Skip):
			m.resolveDetailConflict(sync.ResolutionSkip)
			m.viewport.SetContent(m.buildDetailContent())
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ConflictListModel) selectedConflict() *sync.Conflict {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filtered) {
		return nil
	}
	return m.filtered[cursor]
}

func (m *ConflictListModel) resolveCurrentConflict(resolution sync.ResolutionChoice) {
	if selected := m.selectedConflict(); selected != nil {
		m.resolveConflict(selected, resolution)
	}
}

func (m *ConflictListModel) resolveDetailConflict(resolution sync.ResolutionChoice) {
	if m.detail == nil {
		return
	}
	m.resolveConflict(m.detail, resolution)
}

func (m *ConflictListModel) resolveConflict(conflict *sync.Conflict, resolution sync.ResolutionChoice) {
	m.resolutions[conflict.SkillName] = resolution
	m.table.SetRows(m.cfg.ToRows(m.filtered))
}

func (m *ConflictListModel) resolveConflictAt(idx int, resolution sync.ResolutionChoice) {
	if conflict := m.conflictAt(idx); conflict != nil {
		m.resolveConflict(conflict, resolution)
	}
}

func (m *ConflictListModel) updateTableRow(idx int) {
	if m.conflictAt(idx) == nil {
		return
	}
	m.table.SetRows(m.cfg.ToRows(m.filtered))
}

func (m ConflictListModel) conflictAt(idx int) *sync.Conflict {
	if idx < 0 || idx >= len(m.allItems) {
		return nil
	}
	return m.allItems[idx]
}

func (m ConflictListModel) allResolved() bool {
	for _, c := range m.allItems {
		if _, ok := m.resolutions[c.SkillName]; !ok {
			return false
		}
	}
	return len(m.allItems) > 0
}

func (m ConflictListModel) buildResolutions() []ConflictResolution {
	var result []ConflictResolution
	for _, c := range m.allItems {
		if res, ok := m.resolutions[c.SkillName]; ok {
			content := ""
			switch res {
			case sync.ResolutionUseSource:
				content = c.Source.Content
			case sync.ResolutionUseTarget:
				content = c.Target.Content
			case sync.ResolutionMerge:
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

func (m ConflictListModel) currentDetailConflict() *sync.Conflict {
	if m.detail != nil {
		return m.detail
	}
	return m.conflictAt(m.cursor)
}

func (m ConflictListModel) buildDetailContent() string {
	conflict := m.currentDetailConflict()
	if conflict == nil {
		return "No conflict selected"
	}

	var b strings.Builder

	b.WriteString(conflictStyles.SectionTitle.Render("Conflict Details"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Skill: %s\n", conflict.SkillName)
	fmt.Fprintf(&b, "  Type:  %s\n", conflict.Type)
	fmt.Fprintf(&b, "  %s\n", conflict.DiffSummary())

	if res, ok := m.resolutions[conflict.SkillName]; ok {
		b.WriteString("\n")
		b.WriteString(conflictStyles.Resolved.Render(fmt.Sprintf("  Resolution: %s", res)))
		b.WriteString("\n")
	}

	if len(conflict.Hunks) > 0 {
		b.WriteString("\n")
		b.WriteString(conflictStyles.SectionTitle.Render("Changes"))
		b.WriteString("\n")

		for i, hunk := range conflict.Hunks {
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

			if i < len(conflict.Hunks)-1 {
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString("\n")
		b.WriteString(conflictStyles.SectionTitle.Render("Source Content"))
		b.WriteString("\n")
		b.WriteString(formatConflictContentWithLineNumbers(conflict.Source.Content, conflictStyles.Removed))
		b.WriteString("\n\n")

		b.WriteString(conflictStyles.SectionTitle.Render("Target Content"))
		b.WriteString("\n")
		b.WriteString(formatConflictContentWithLineNumbers(conflict.Target.Content, conflictStyles.Added))
	}

	b.WriteString("\n\n")
	b.WriteString(conflictStyles.Info.Render("Press: s=source, t=target, m=merge, x=skip"))

	return b.String()
}

// View implements tea.Model.
func (m ConflictListModel) View() string {
	if m.quitting {
		return ""
	}
	if m.phase == phaseDetail {
		return m.viewDetail()
	}
	return m.ListModel.View()
}

func (m ConflictListModel) viewDetail() string {
	if !m.detailReady {
		return "Loading..."
	}

	var b strings.Builder

	skillName := ""
	if conflict := m.currentDetailConflict(); conflict != nil {
		skillName = conflict.SkillName
	}
	b.WriteString(conflictStyles.Title.Render(fmt.Sprintf("📄 Conflict: %s", skillName)))
	b.WriteString("\n\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	b.WriteString(conflictStyles.Status.Render(fmt.Sprintf("Scroll: %d%%", scrollPercent)))
	b.WriteString("\n")

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.renderDetailHelp())
	} else {
		b.WriteString(m.renderDetailShortHelp())
	}

	return b.String()
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
	if r, ok := m.result.(ConflictListResult); ok {
		return r
	}
	return ConflictListResult{}
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
