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

// configListState holds config-specific mutable state captured by ListModel callbacks.
type configListState struct {
	cfg        *config.Config
	defaultCfg *config.Config
	modified   bool
	editing    bool
	editValue  string
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

func (s *configListState) selectedItem(m *ListModel[configItem]) *configItem {
	if s == nil || m == nil {
		return nil
	}
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filtered) {
		return nil
	}
	return &m.filtered[cursor]
}

func (s *configListState) matches(item configItem, lowerFilter string) bool {
	if lowerFilter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.Section), lowerFilter) ||
		strings.Contains(strings.ToLower(item.Key), lowerFilter) ||
		strings.Contains(strings.ToLower(item.Description), lowerFilter) ||
		strings.Contains(strings.ToLower(item.Value), lowerFilter)
}

func (s *configListState) statusText(filtered, total int, filter string) string {
	if filtered == 0 {
		return "No settings match the current filter"
	}
	if filter != "" {
		return fmt.Sprintf("%d of %d settings shown", filtered, total)
	}
	return "Press space/enter to toggle"
}

func (s *configListState) header() string {
	if s == nil || !s.modified {
		return ""
	}
	return configListStyles.Modified.Render("[modified]")
}

func (s *configListState) extraBody(m *ListModel[configItem]) string {
	if s == nil || m == nil || !s.editing {
		return ""
	}
	item := s.selectedItem(m)
	if item == nil {
		return ""
	}
	prompt := fmt.Sprintf("Edit %s.%s: ", item.Section, item.Key)
	return configListStyles.EditPrompt.Render(prompt) + configListStyles.FilterInput.Render(s.editValue+"█")
}

// extraKeys handles config-specific keys (toggle, edit, reset). It also propagates
// item changes back to the ListModel so filtering and table rows stay current.
func (s *configListState) extraKeys(m *ListModel[configItem], msg tea.KeyMsg) bool {
	if s == nil || m == nil {
		return false
	}
	switch msg.String() {
	case " ", "enter":
		item := s.selectedItem(m)
		if item == nil {
			return true
		}
		if item.ValueType == "bool" || len(item.Options) > 0 {
			current := *item
			cursor := m.table.Cursor()
			s.toggleOrCycleCurrentValue(&current)
			s.syncItemsToBase(m)
			if cursor < len(m.filtered) {
				m.table.SetCursor(cursor)
			}
		}
		return true
	case "e":
		item := s.selectedItem(m)
		if item != nil && item.ValueType != "bool" && len(item.Options) == 0 {
			s.editing = true
			s.editValue = item.Value
		}
		return true
	case "r":
		item := s.selectedItem(m)
		if item != nil {
			current := *item
			cursor := m.table.Cursor()
			s.resetCurrentToDefault(&current)
			s.syncItemsToBase(m)
			if cursor < len(m.filtered) {
				m.table.SetCursor(cursor)
			}
		}
		return true
	}
	return false
}

// syncItemsToBase rebuilds items from the current config and pushes them into the ListModel.
func (s *configListState) syncItemsToBase(m *ListModel[configItem]) {
	newItems := s.buildConfigItems()
	m.allItems = newItems
	m.filtered = s.applyFilterTo(newItems, m.filter)
	m.table.SetRows(m.cfg.ToRows(m.filtered))
}

func (s *configListState) applyFilterTo(items []configItem, filter string) []configItem {
	if filter == "" {
		return items
	}
	lf := strings.ToLower(filter)
	var filtered []configItem
	for _, item := range items {
		if s.matches(item, lf) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *configListState) toggleOrCycleCurrentValue(item *configItem) {
	if s == nil || item == nil {
		return
	}
	var newValue string
	if item.ValueType == "bool" {
		if item.Value == "true" {
			newValue = "false"
		} else {
			newValue = "true"
		}
	} else if len(item.Options) > 0 {
		currentIdx := -1
		for i, opt := range item.Options {
			if opt == item.Value {
				currentIdx = i
				break
			}
		}
		newValue = item.Options[(currentIdx+1)%len(item.Options)]
	} else {
		return
	}
	s.updateConfigValue(item.Section, item.Key, newValue)
}

func (s *configListState) resetCurrentToDefault(item *configItem) {
	if s == nil || item == nil || s.defaultCfg == nil {
		return
	}
	orig := s.cfg
	s.cfg = s.defaultCfg
	defaultItems := s.buildConfigItems()
	s.cfg = orig
	for _, di := range defaultItems {
		if di.Section == item.Section && di.Key == item.Key {
			s.updateConfigValue(item.Section, item.Key, di.Value)
			return
		}
	}
}

func (s *configListState) applyEditValue(item *configItem) {
	if s == nil || item == nil {
		return
	}
	s.updateConfigValue(item.Section, item.Key, s.editValue)
	s.editing = false
	s.editValue = ""
}

func (s *configListState) updateConfigValue(section, key, value string) {
	if s == nil || s.cfg == nil {
		return
	}
	switch section {
	case "Sync":
		switch key {
		case "DefaultStrategy":
			s.cfg.Sync.DefaultStrategy = value
		}
	case "Output":
		switch key {
		case "Color":
			s.cfg.Output.Color = value
		}
	case "Similarity":
		switch key {
		case "NameThreshold":
			if v, err := parseFloat(value); err == nil && v >= 0 && v <= 1 {
				s.cfg.Similarity.NameThreshold = v
			}
		case "ContentThreshold":
			if v, err := parseFloat(value); err == nil && v >= 0 && v <= 1 {
				s.cfg.Similarity.ContentThreshold = v
			}
		case "Algorithm":
			s.cfg.Similarity.Algorithm = value
		}
	}
	s.modified = true
}

func (s *configListState) buildConfigItems() []configItem {
	if s == nil || s.cfg == nil {
		return nil
	}
	cfg := s.cfg
	items := []configItem{
		{
			Section:     "Sync",
			Key:         "DefaultStrategy",
			Description: "Default conflict resolution strategy",
			Value:       cfg.Sync.DefaultStrategy,
			ValueType:   "string",
			Options:     []string{"overwrite", "skip", "newer", "merge", "three-way", "interactive"},
		},
		{
			Section:     "Output",
			Key:         "Color",
			Description: "Color output mode",
			Value:       cfg.Output.Color,
			ValueType:   "string",
			Options:     []string{"auto", "always", "never"},
		},
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
	sort.Slice(items, func(i, j int) bool {
		if !strings.EqualFold(items[i].Section, items[j].Section) {
			return strings.ToLower(items[i].Section) < strings.ToLower(items[j].Section)
		}
		return strings.ToLower(items[i].Key) < strings.ToLower(items[j].Key)
	})
	return items
}

// ConfigListModel is the BubbleTea model for interactive config editing.
// It embeds ListModel[configItem] for common list behavior (filtering, help, navigation)
// and adds config-specific state (modified flag, inline editing, save-on-quit confirm).
type ConfigListModel struct {
	ListModel[configItem] // promotes: table, filter, filtering, showHelp, confirmMode, quitting, filtered, allItems, width, height

	state      *configListState
	cfg        *config.Config
	defaultCfg *config.Config
	modified   bool
	editing    bool
	editValue  string
	result     ConfigListResult // shadows ListModel.result (any) with typed result
	items      []configItem     // proxy for allItems; kept in sync for test compatibility
}

// itemsToRows converts config items to table rows.
func itemsToRows(items []configItem) []table.Row {
	rows := make([]table.Row, len(items))
	for i, item := range items {
		rows[i] = table.Row{
			item.Section,
			item.Key,
			truncateTableValue(item.Value, 25),
			truncateTableValue(item.Description, 35),
		}
	}
	return rows
}

// NewConfigListModel creates a new config list model.
func NewConfigListModel(cfg *config.Config) ConfigListModel {
	if cfg == nil {
		cfg = config.Default()
	}

	state := &configListState{
		cfg:        cfg,
		defaultCfg: config.Default(),
	}
	items := state.buildConfigItems()

	saveKey := key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save changes"))

	m := ConfigListModel{
		cfg:        cfg,
		defaultCfg: state.defaultCfg,
		state:      state,
		items:      items,
	}

	m.ListModel = NewListModel(items, ListConfig[configItem]{
		Title: "⚙️  Configuration",
		Columns: []table.Column{
			{Title: "Section", Width: 12},
			{Title: "Setting", Width: 20},
			{Title: "Value", Width: 25},
			{Title: "Description", Width: 35},
		},
		ToRows:     itemsToRows,
		Matches:    state.matches,
		ShortHelp:  m.renderShortHelp,
		FullHelp:   m.renderFullHelp,
		StatusText: state.statusText,
		Actions: []ActionBinding[configItem]{
			{
				Binding: saveKey,
				Apply: func(configItem) any {
					return ConfigListResult{Action: ConfigActionSave, Config: cfg}
				},
			},
		},
		ReservedLines: 12,
		Header:        state.header,
		ExtraBody:     state.extraBody,
		ExtraKeys:     state.extraKeys,
	})
	return m
}

// Init implements tea.Model.
func (m ConfigListModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It intercepts editing mode, modified-quit guard, and
// confirm mode before delegating common behavior (filtering, help, navigation) to the base.
func (m ConfigListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Sync test-set mutable fields into state so callbacks see current values.
	if m.state != nil {
		m.state.modified = m.modified
		m.state.editing = m.editing
		m.state.editValue = m.editValue
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Editing mode: consume all characters before base key dispatch.
		if m.editing {
			return m.handleEditingMode(keyMsg)
		}

		// Config confirm mode: we must build ConfigListResult on 'y' (base only quits).
		if m.confirmMode {
			return m.handleConfirmMode(keyMsg)
		}

		// Modified-quit guard: intercept 'q'/'ctrl+c' before base quits unconditionally.
		if !m.filtering && m.modified &&
			(keyMsg.String() == "q" || keyMsg.Type == tea.KeyCtrlC) {
			m.confirmMode = true
			m.ListModel.confirmMsg = "⚠️  Save changes before quitting? (y/n)"
			return m, nil
		}

		// Config-specific keys: handle before base so we can sync ListModel state.
		if !m.filtering && !m.confirmMode && m.state != nil {
			if handled := m.state.extraKeys(&m.ListModel, keyMsg); handled {
				m.modified = m.state.modified
				m.editing = m.state.editing
				m.editValue = m.state.editValue
				m.items = m.allItems
				return m, nil
			}
		}
	}

	inner, cmd := m.ListModel.Update(msg)
	if next, ok := inner.(ListModel[configItem]); ok {
		m.ListModel = next
	}
	m.items = m.allItems
	if r, ok := m.ListModel.result.(ConfigListResult); ok {
		m.result = r
	}
	return m, cmd
}

func (m ConfigListModel) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.result = ConfigListResult{Action: ConfigActionSave, Config: m.cfg}
		m.quitting = true
		return m, tea.Quit
	case "n", "N", "esc":
		m.confirmMode = false
		m.ListModel.confirmMsg = ""
	}
	return m, nil
}

func (m ConfigListModel) handleEditingMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		item := m.state.selectedItem(&m.ListModel)
		if item != nil {
			m.state.editValue = m.editValue
			m.state.applyEditValue(item)
			cursor := m.table.Cursor()
			m.state.syncItemsToBase(&m.ListModel)
			if cursor < len(m.filtered) {
				m.table.SetCursor(cursor)
			}
			m.items = m.allItems
		}
		m.editing = false
		m.editValue = ""
		m.state.editing = false
		m.state.editValue = ""
		m.modified = m.state.modified
	case "esc":
		m.editing = false
		m.editValue = ""
		m.state.editing = false
		m.state.editValue = ""
	case "backspace":
		if len(m.editValue) > 0 {
			m.editValue = m.editValue[:len(m.editValue)-1]
			m.state.editValue = m.editValue
		}
	default:
		if len(msg.String()) == 1 {
			m.editValue += msg.String()
			m.state.editValue = m.editValue
		}
	}
	return m, nil
}

// View delegates to the base ListModel view. Config-specific elements ([modified],
// edit prompt) are rendered via the Header and ExtraBody callbacks.
func (m ConfigListModel) View() string {
	if m.state != nil {
		m.state.modified = m.modified
		m.state.editing = m.editing
		m.state.editValue = m.editValue
	}
	return m.ListModel.View()
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

// toggleOrCycleCurrentValue is kept for test compatibility.
func (m *ConfigListModel) toggleOrCycleCurrentValue() {
	item := m.state.selectedItem(&m.ListModel)
	if item == nil || (item.ValueType != "bool" && len(item.Options) == 0) {
		return
	}
	cursor := m.table.Cursor()
	current := *item
	m.state.toggleOrCycleCurrentValue(&current)
	m.state.syncItemsToBase(&m.ListModel)
	if cursor < len(m.filtered) {
		m.table.SetCursor(cursor)
	}
	m.items = m.allItems
	m.modified = m.state.modified
}

// Result returns the typed result of the user interaction.
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

// parseFloat parses a float64 from a string.
func parseFloat(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}
