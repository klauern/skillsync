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

	"github.com/klauern/skillsync/internal/config"
)

// ConfigAction represents the action to perform after config editing.
type ConfigAction int

const (
	// ConfigActionNone means no action was taken (user quit without saving).
	ConfigActionNone ConfigAction = iota
	// ConfigActionSave means the user wants to save the configuration.
	ConfigActionSave
)

// ConfigListResult contains the result of the config list TUI interaction.
type ConfigListResult struct {
	Action ConfigAction
	Config *config.Config
}

// configItem represents a single configuration item for display.
type configItem struct {
	Section     string   // Section name (e.g., "Sync", "Cache")
	Key         string   // Setting key (e.g., "DefaultStrategy", "Enabled")
	Description string   // Human-readable description
	Value       string   // Current value as string
	ValueType   string   // Type: "bool", "string", "int", "duration", "float"
	Options     []string // For enum-type fields, the valid options
}

// configListKeyMap defines the key bindings for the config list.
type configListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	Edit     key.Binding
	Save     key.Binding
	Reset    key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultConfigListKeyMap() configListKeyMap {
	return configListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "enter"),
			key.WithHelp("space/enter", "toggle/cycle"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit value"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save changes"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset to default"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFlt: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear/cancel"),
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

// ConfigListModel is the BubbleTea model for interactive config editing.
type ConfigListModel struct {
	table       table.Model
	items       []configItem
	filtered    []configItem
	keys        configListKeyMap
	result      ConfigListResult
	cfg         *config.Config
	defaultCfg  *config.Config
	filter      string
	filtering   bool
	editing     bool
	editValue   string
	showHelp    bool
	confirmMode bool
	modified    bool
	width       int
	height      int
	quitting    bool
}

// Styles for the config list TUI.
var configListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Section     lipgloss.Style
	Key         lipgloss.Style
	Value       lipgloss.Style
	ValueBool   lipgloss.Style
	Modified    lipgloss.Style
	EditPrompt  lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Section:     lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Key:         lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	Value:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	ValueBool:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	Modified:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
	EditPrompt:  lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
}

// NewConfigListModel creates a new config list model.
func NewConfigListModel(cfg *config.Config) ConfigListModel {
	if cfg == nil {
		cfg = config.Default()
	}

	columns := []table.Column{
		{Title: "Section", Width: 12},
		{Title: "Setting", Width: 20},
		{Title: "Value", Width: 25},
		{Title: "Description", Width: 35},
	}

	m := ConfigListModel{
		cfg:        cfg,
		defaultCfg: config.Default(),
		keys:       defaultConfigListKeyMap(),
	}

	m.items = m.buildConfigItems()

	// Sort items alphabetically by section, then by key within section (case-insensitive)
	sort.Slice(m.items, func(i, j int) bool {
		if !strings.EqualFold(m.items[i].Section, m.items[j].Section) {
			return strings.ToLower(m.items[i].Section) < strings.ToLower(m.items[j].Section)
		}
		return strings.ToLower(m.items[i].Key) < strings.ToLower(m.items[j].Key)
	})

	m.filtered = m.items
	rows := m.itemsToRows(m.items)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
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

// buildConfigItems creates the list of editable config items from the config.
func (m *ConfigListModel) buildConfigItems() []configItem {
	cfg := m.cfg
	items := []configItem{
		// Sync settings
		{
			Section:     "Sync",
			Key:         "DefaultStrategy",
			Description: "Default conflict resolution strategy",
			Value:       cfg.Sync.DefaultStrategy,
			ValueType:   "string",
			Options:     []string{"overwrite", "skip", "newer", "merge", "three-way", "interactive"},
		},

		// Output settings
		{
			Section:     "Output",
			Key:         "Color",
			Description: "Color output mode",
			Value:       cfg.Output.Color,
			ValueType:   "string",
			Options:     []string{"auto", "always", "never"},
		},

		// Similarity settings
		{
			Section:     "Similarity",
			Key:         "NameThreshold",
			Description: "Name similarity threshold (0.0-1.0)",
			Value:       fmt.Sprintf("%.2f", cfg.Similarity.NameThreshold),
			ValueType:   "float",
		},
		{
			Section:     "Similarity",
			Key:         "ContentThreshold",
			Description: "Content similarity threshold (0.0-1.0)",
			Value:       fmt.Sprintf("%.2f", cfg.Similarity.ContentThreshold),
			ValueType:   "float",
		},
		{
			Section:     "Similarity",
			Key:         "Algorithm",
			Description: "Similarity algorithm",
			Value:       cfg.Similarity.Algorithm,
			ValueType:   "string",
			Options:     []string{"levenshtein", "jaro-winkler", "combined"},
		},
	}

	return items
}

// itemsToRows converts config items to table rows.
func (m *ConfigListModel) itemsToRows(items []configItem) []table.Row {
	rows := make([]table.Row, len(items))
	for i, item := range items {
		// Truncate long values
		value := item.Value
		if len(value) > 23 {
			value = value[:20] + "..."
		}

		// Truncate description
		desc := item.Description
		if len(desc) > 33 {
			desc = desc[:30] + "..."
		}

		rows[i] = table.Row{
			item.Section,
			item.Key,
			value,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m ConfigListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ConfigListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		newHeight := max(msg.Height-12, 5)
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update table
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// handleKeyMsg processes keyboard input based on current mode.
func (m ConfigListModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmMode {
		return m.handleConfirmMode(msg)
	}
	if m.editing {
		return m.handleEditingMode(msg)
	}
	if m.filtering {
		return m.handleFilteringMode(msg)
	}
	return m.handleNormalMode(msg)
}

// handleConfirmMode handles keys during confirmation dialog.
func (m ConfigListModel) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.result = ConfigListResult{
			Action: ConfigActionSave,
			Config: m.cfg,
		}
		m.quitting = true
		return m, tea.Quit
	case "n", "N", "esc":
		m.confirmMode = false
		return m, nil
	}
	return m, nil
}

// handleEditingMode handles keys during value editing.
func (m ConfigListModel) handleEditingMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.applyEditValue()
		m.editing = false
		m.editValue = ""
	case "esc":
		m.editing = false
		m.editValue = ""
	case "backspace":
		if len(m.editValue) > 0 {
			m.editValue = m.editValue[:len(m.editValue)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.editValue += msg.String()
		}
	}
	return m, nil
}

// handleFilteringMode handles keys during filter input.
func (m ConfigListModel) handleFilteringMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
	case "esc":
		m.filter = ""
		m.filtering = false
		m.applyFilter()
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

// handleNormalMode handles keys in normal browsing mode.
func (m ConfigListModel) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.modified {
			m.confirmMode = true
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
	case key.Matches(msg, m.keys.ClearFlt):
		if m.filter != "" {
			m.filter = ""
			m.applyFilter()
		}
	case key.Matches(msg, m.keys.Up):
		m.table.MoveUp(1)
	case key.Matches(msg, m.keys.Down):
		m.table.MoveDown(1)
	case key.Matches(msg, m.keys.Toggle):
		m.toggleOrCycleCurrentValue()
	case key.Matches(msg, m.keys.Edit):
		item := m.getCurrentItem()
		if item != nil && item.ValueType != "bool" && len(item.Options) == 0 {
			m.editing = true
			m.editValue = item.Value
		}
	case key.Matches(msg, m.keys.Save):
		m.result = ConfigListResult{
			Action: ConfigActionSave,
			Config: m.cfg,
		}
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.Reset):
		m.resetCurrentToDefault()
	}
	return m, nil
}

// getCurrentItem returns the currently selected config item.
func (m *ConfigListModel) getCurrentItem() *configItem {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return &m.filtered[cursor]
	}
	return nil
}

// toggleOrCycleCurrentValue toggles boolean values or cycles through options.
func (m *ConfigListModel) toggleOrCycleCurrentValue() {
	item := m.getCurrentItem()
	if item == nil {
		return
	}

	var newValue string

	if item.ValueType == "bool" {
		// Toggle boolean
		if item.Value == "true" {
			newValue = "false"
		} else {
			newValue = "true"
		}
	} else if len(item.Options) > 0 {
		// Cycle through options
		currentIdx := -1
		for i, opt := range item.Options {
			if opt == item.Value {
				currentIdx = i
				break
			}
		}
		nextIdx := (currentIdx + 1) % len(item.Options)
		newValue = item.Options[nextIdx]
	} else {
		// Not toggleable
		return
	}

	m.updateConfigValue(item.Section, item.Key, newValue)
	m.refreshItems()
}

// applyEditValue applies the edited value to the config.
func (m *ConfigListModel) applyEditValue() {
	item := m.getCurrentItem()
	if item == nil {
		return
	}

	m.updateConfigValue(item.Section, item.Key, m.editValue)
	m.refreshItems()
}

// resetCurrentToDefault resets the current item to its default value.
func (m *ConfigListModel) resetCurrentToDefault() {
	item := m.getCurrentItem()
	if item == nil {
		return
	}

	// Find the default value
	defaultItems := m.buildDefaultItems()
	for _, di := range defaultItems {
		if di.Section == item.Section && di.Key == item.Key {
			m.updateConfigValue(item.Section, item.Key, di.Value)
			m.refreshItems()
			return
		}
	}
}

// buildDefaultItems builds items from the default config.
func (m *ConfigListModel) buildDefaultItems() []configItem {
	orig := m.cfg
	m.cfg = m.defaultCfg
	items := m.buildConfigItems()
	m.cfg = orig
	return items
}

// updateConfigValue updates a config value by section and key.
func (m *ConfigListModel) updateConfigValue(section, key, value string) {
	switch section {
	case "Sync":
		m.updateSyncConfig(key, value)
	case "Output":
		m.updateOutputConfig(key, value)
	case "Similarity":
		m.updateSimilarityConfig(key, value)
	}
	m.modified = true
}

func (m *ConfigListModel) updateSyncConfig(key, value string) {
	switch key {
	case "DefaultStrategy":
		m.cfg.Sync.DefaultStrategy = value
	}
}

func (m *ConfigListModel) updateOutputConfig(key, value string) {
	switch key {
	case "Color":
		m.cfg.Output.Color = value
	}
}

func (m *ConfigListModel) updateSimilarityConfig(key, value string) {
	switch key {
	case "NameThreshold":
		if v, err := parseFloat(value); err == nil && v >= 0 && v <= 1 {
			m.cfg.Similarity.NameThreshold = v
		}
	case "ContentThreshold":
		if v, err := parseFloat(value); err == nil && v >= 0 && v <= 1 {
			m.cfg.Similarity.ContentThreshold = v
		}
	case "Algorithm":
		m.cfg.Similarity.Algorithm = value
	}
}

// refreshItems rebuilds the items list from the current config.
func (m *ConfigListModel) refreshItems() {
	cursor := m.table.Cursor()
	m.items = m.buildConfigItems()
	m.applyFilter()
	// Restore cursor position
	if cursor < len(m.filtered) {
		m.table.SetCursor(cursor)
	}
}

// applyFilter filters items by the current filter text.
func (m *ConfigListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.items
	} else {
		var filtered []configItem
		lowerFilter := strings.ToLower(m.filter)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item.Section), lowerFilter) ||
				strings.Contains(strings.ToLower(item.Key), lowerFilter) ||
				strings.Contains(strings.ToLower(item.Description), lowerFilter) ||
				strings.Contains(strings.ToLower(item.Value), lowerFilter) {
				filtered = append(filtered, item)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(m.itemsToRows(m.filtered))
}

// View implements tea.Model.
func (m ConfigListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := "⚙️  Configuration"
	if m.modified {
		title += configListStyles.Modified.Render(" [modified]")
	}
	b.WriteString(configListStyles.Title.Render(title))
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := configListStyles.Filter.Render("Filter: ")
		filterVal := configListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Edit prompt
	if m.editing {
		item := m.getCurrentItem()
		if item != nil {
			prompt := fmt.Sprintf("Edit %s.%s: ", item.Section, item.Key)
			b.WriteString(configListStyles.EditPrompt.Render(prompt))
			b.WriteString(configListStyles.FilterInput.Render(m.editValue + "█"))
			b.WriteString("\n\n")
		}
	}

	// Confirm dialog
	if m.confirmMode {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		confirmMsg := "⚠️  Save changes before quitting? (y/n)"
		b.WriteString(configListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	item := m.getCurrentItem()
	var statusText string
	if item != nil {
		if item.ValueType == "bool" {
			statusText = "Press space/enter to toggle"
		} else if len(item.Options) > 0 {
			statusText = fmt.Sprintf("Options: %s", strings.Join(item.Options, ", "))
		} else {
			statusText = "Press 'e' to edit, 'r' to reset"
		}
	}
	b.WriteString(configListStyles.Status.Render(statusText))
	b.WriteString("\n")

	// Help
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.renderFullHelp())
	} else {
		b.WriteString(m.renderShortHelp())
	}

	return b.String()
}

func (m ConfigListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"e edit",
		"s save",
		"r reset",
		"/ filter",
		"? help",
		"q quit",
	}
	return configListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ConfigListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down

Editing:
  Space    Toggle boolean / cycle options
  Enter    Toggle boolean / cycle options
  e        Edit value (for text/number fields)
  r        Reset to default value

Actions:
  s        Save configuration
  /        Filter settings
  Esc      Clear filter / cancel edit

General:
  ?        Toggle full help
  q        Quit (prompts to save if modified)`
	return configListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m ConfigListModel) Result() ConfigListResult {
	return m.result
}

// RunConfigList runs the interactive config editor and returns the result.
func RunConfigList(cfg *config.Config) (ConfigListResult, error) {
	model := NewConfigListModel(cfg)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return ConfigListResult{}, err
	}

	if m, ok := finalModel.(ConfigListModel); ok {
		return m.Result(), nil
	}

	return ConfigListResult{}, nil
}

// Helper function for parsing values.
func parseFloat(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}
