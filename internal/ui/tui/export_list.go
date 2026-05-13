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
	NextPlat  key.Binding
	PrevPlat  key.Binding
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
		NextPlat: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next platform"),
		),
		PrevPlat: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "prev platform"),
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
	selectableListModel[model.Skill]
	keys            exportListKeyMap
	result          ExportListResult
	platformOptions []model.Platform
	platformIndex   int // Index into platformOptions (-1 = all)
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
	Title          lipgloss.Style
	Help           lipgloss.Style
	Filter         lipgloss.Style
	FilterInput    lipgloss.Style
	Confirm        lipgloss.Style
	Status         lipgloss.Style
	Selected       lipgloss.Style
	Checkbox       lipgloss.Style
	Format         lipgloss.Style
	Option         lipgloss.Style
	OptionVal      lipgloss.Style
	PlatformTab    lipgloss.Style
	PlatformActive lipgloss.Style
}{
	Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:         lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Selected:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Checkbox:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Format:         lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Option:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	OptionVal:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	PlatformTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	PlatformActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
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

	platformSet := make(map[model.Platform]bool)
	for _, s := range skills {
		platformSet[s.Platform] = true
	}

	var platformOptions []model.Platform
	for _, platform := range model.AllPlatforms() {
		if platformSet[platform] {
			platformOptions = append(platformOptions, platform)
		}
	}

	m := ExportListModel{
		keys:            defaultExportListKeyMap(),
		format:          export.FormatJSON,
		includeMetadata: true,
		pretty:          true,
		platformOptions: platformOptions,
		platformIndex:   -1,
	}

	m.selectableListModel = newSelectableListModel(skills, true, columns, 15, skillKey, skillListFilterMatch, func(s model.Skill, selected bool) table.Row {
		checkbox := "[ ]"
		if selected {
			checkbox = "[✓]"
		}
		return table.Row{
			checkbox,
			truncateTableValue(s.Name, 25),
			truncateTableValue(string(s.Platform), 12),
			truncateTableValue(s.DisplayScope(), 10),
			truncateTableValue(s.Description, 40),
		}
	})

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
	m.table.SetStyles(s)

	return m
}

// Init implements tea.Model.
func (m ExportListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
//
//nolint:gocyclo // interactive table/event handling is intentionally central here
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

		case key.Matches(msg, m.keys.NextPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex++
				if m.platformIndex >= len(m.platformOptions) {
					m.platformIndex = -1
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex--
				if m.platformIndex < -1 {
					m.platformIndex = len(m.platformOptions) - 1
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.Toggle):
			m.toggleCurrentSelection()
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			m.toggleAllSelection()
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
	filtered := m.skills

	if m.platformIndex >= 0 && m.platformIndex < len(m.platformOptions) {
		selectedPlatform := m.platformOptions[m.platformIndex]
		var platformFiltered []model.Skill
		for _, s := range filtered {
			if s.Platform == selectedPlatform {
				platformFiltered = append(platformFiltered, s)
			}
		}
		filtered = platformFiltered
	}

	if m.filter != "" {
		var textFiltered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range filtered {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(s.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				textFiltered = append(textFiltered, s)
			}
		}
		filtered = textFiltered
	}

	m.filtered = filtered
	m.refreshTable()
}

// View renders the export list UI.
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

	b.WriteString(m.renderPlatformTabs())
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
	if m.filter != "" || m.platformIndex >= 0 {
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
		"h/l platform",
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

Platform Filtering:
  h        Previous platform
  l        Next platform

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

func (m ExportListModel) renderPlatformTabs() string {
	var tabs []string

	if m.platformIndex == -1 {
		tabs = append(tabs, exportListStyles.PlatformActive.Render("[All]"))
	} else {
		tabs = append(tabs, exportListStyles.PlatformTab.Render(" All "))
	}

	for i, platform := range m.platformOptions {
		if i == m.platformIndex {
			tabs = append(tabs, exportListStyles.PlatformActive.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			tabs = append(tabs, exportListStyles.PlatformTab.Render(fmt.Sprintf(" %s ", platform)))
		}
	}

	return strings.Join(tabs, "")
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
