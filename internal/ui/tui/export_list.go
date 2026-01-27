// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
)

// ExportAction represents the action to perform after export configuration.
type ExportAction int

const (
	// ExportActionNone means no action was taken (user quit).
	ExportActionNone ExportAction = iota
	// ExportActionExport means the user wants to export selected skills.
	ExportActionExport
)

// ExportListResult contains the result of the export list TUI interaction.
type ExportListResult struct {
	Action          ExportAction
	SelectedSkills  []model.Skill
	Format          export.Format
	IncludeMetadata bool
	Pretty          bool
}

// exportListKeyMap defines the key bindings for the export list.
type exportListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	ToggleAll key.Binding
	Format    key.Binding
	Metadata  key.Binding
	Confirm   key.Binding
	Filter    key.Binding
	ClearFlt  key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func defaultExportListKeyMap() exportListKeyMap {
	return exportListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "tab"),
			key.WithHelp("space/tab", "toggle"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		Format: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "cycle format"),
		),
		Metadata: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle metadata"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "export selected"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFlt: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ExportListModel is the BubbleTea model for interactive export skill selection.
type ExportListModel struct {
	table           table.Model
	skills          []model.Skill
	filtered        []model.Skill
	selected        map[string]bool // map of skill name+platform to selected state
	keys            exportListKeyMap
	result          ExportListResult
	filter          string
	filtering       bool
	showHelp        bool
	confirmMode     bool
	width           int
	height          int
	quitting        bool
	format          export.Format
	includeMetadata bool
	pretty          bool
}

// Styles for the export list TUI.
var exportListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Selected    lipgloss.Style
	Checkbox    lipgloss.Style
	Format      lipgloss.Style
	Option      lipgloss.Style
	OptionVal   lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Selected:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Format:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Option:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	OptionVal:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
}

// skillKey creates a unique key for a skill (name + platform combination).
func skillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s", s.Platform, s.Name)
}

// NewExportListModel creates a new export list model.
func NewExportListModel(skills []model.Skill) ExportListModel {
	columns := []table.Column{
		{Title: " ", Width: 3},            // Checkbox column
		{Title: "Name", Width: 25},        // Skill name
		{Title: "Platform", Width: 12},    // Platform
		{Title: "Scope", Width: 10},       // Scope
		{Title: "Description", Width: 40}, // Description
	}

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	// Initialize all skills as selected by default
	selected := make(map[string]bool)
	for _, s := range skills {
		selected[skillKey(s)] = true
	}

	m := ExportListModel{
		skills:          skills,
		filtered:        skills,
		selected:        selected,
		keys:            defaultExportListKeyMap(),
		format:          export.FormatJSON,
		includeMetadata: true,
		pretty:          true,
	}

	rows := m.skillsToRows(skills)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

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

	m.table = t
	return m
}

func (m ExportListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[skillKey(s)] {
			checkbox = "[✓]"
		}

		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		platform := string(s.Platform)
		if len(platform) > 12 {
			platform = platform[:9] + "..."
		}
		scope := s.DisplayScope()
		if len(scope) > 10 {
			scope = scope[:7] + "..."
		}
		desc := s.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		rows[i] = table.Row{
			checkbox,
			name,
			platform,
			scope,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m ExportListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ExportListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-12, 5) // Reserve space for title, options, help, status
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = ExportListResult{
					Action:          ExportActionExport,
					SelectedSkills:  m.getSelectedSkills(),
					Format:          m.format,
					IncludeMetadata: m.includeMetadata,
					Pretty:          m.pretty,
				}
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				return m, nil
			}
			return m, nil
		}

		// Handle filtering mode
		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filtering = false
				return m, nil
			case "esc":
				m.filter = ""
				m.filtering = false
				m.applyFilter()
				return m, nil
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.applyFilter()
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.filter += msg.String()
					m.applyFilter()
				}
				return m, nil
			}
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Filter):
			m.filtering = true
			return m, nil

		case key.Matches(msg, m.keys.ClearFlt):
			m.filter = ""
			m.applyFilter()
			return m, nil

		case key.Matches(msg, m.keys.Toggle):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				m.selected[skillKey(skill)] = !m.selected[skillKey(skill)]
				m.table.SetRows(m.skillsToRows(m.filtered))
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			// Count how many are currently selected
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[skillKey(s)] {
					selectedCount++
				}
			}
			// If all or most are selected, deselect all; otherwise select all
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[skillKey(s)] = selectAll
			}
			m.table.SetRows(m.skillsToRows(m.filtered))
			return m, nil

		case key.Matches(msg, m.keys.Format):
			// Cycle through formats: JSON -> YAML -> Markdown -> JSON
			switch m.format {
			case export.FormatJSON:
				m.format = export.FormatYAML
			case export.FormatYAML:
				m.format = export.FormatMarkdown
			case export.FormatMarkdown:
				m.format = export.FormatJSON
			}
			return m, nil

		case key.Matches(msg, m.keys.Metadata):
			m.includeMetadata = !m.includeMetadata
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			selectedSkills := m.getSelectedSkills()
			if len(selectedSkills) > 0 {
				m.confirmMode = true
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *ExportListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.skills
	} else {
		var filtered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range m.skills {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(s.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				filtered = append(filtered, s)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m ExportListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m ExportListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.skills {
		if m.selected[skillKey(s)] {
			selected = append(selected, s)
		}
	}
	return selected
}

// View implements tea.Model.
func (m ExportListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := exportListStyles.Title.Render("📤 Export Skills")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Export options line
	formatLabel := exportListStyles.Option.Render("Format: ")
	formatVal := exportListStyles.Format.Render(strings.ToUpper(string(m.format)))

	metadataLabel := exportListStyles.Option.Render("  Metadata: ")
	metadataVal := "No"
	if m.includeMetadata {
		metadataVal = "Yes"
	}
	metadataValStyled := exportListStyles.OptionVal.Render(metadataVal)

	optionsLine := formatLabel + formatVal + metadataLabel + metadataValStyled
	b.WriteString(optionsLine)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := exportListStyles.Filter.Render("Filter: ")
		filterVal := exportListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Confirmation dialog
	if m.confirmMode {
		selectedCount := len(m.getSelectedSkills())
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		confirmMsg := fmt.Sprintf("Export %d skill(s) as %s? (y/n)", selectedCount, strings.ToUpper(string(m.format)))
		b.WriteString(exportListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) selected of %d", selectedCount, len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, len(m.filtered), len(m.skills))
	}
	b.WriteString(exportListStyles.Status.Render(status))
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

func (m ExportListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"a toggle all",
		"f format",
		"m metadata",
		"y export",
		"/ filter",
		"? help",
		"q quit",
	}
	return exportListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ExportListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill
  a          Toggle all skills

Export Options:
  f        Cycle format (JSON → YAML → Markdown)
  m        Toggle metadata inclusion

Actions:
  y        Confirm and export selected skills

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without exporting`
	return exportListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m ExportListModel) Result() ExportListResult {
	return m.result
}

// RunExportList runs the interactive export list and returns the result.
func RunExportList(skills []model.Skill) (ExportListResult, error) {
	if len(skills) == 0 {
		return ExportListResult{}, nil
	}

	mdl := NewExportListModel(skills)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ExportListResult{}, err
	}

	if m, ok := finalModel.(ExportListModel); ok {
		return m.Result(), nil
	}

	return ExportListResult{}, nil
}
