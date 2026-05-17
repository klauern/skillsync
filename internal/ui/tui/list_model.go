// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listStyles are the shared styles used across all list screens.
var listStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
}

// listBaseKeys are the shared key bindings for all list screens.
type listBaseKeys struct {
	Up       key.Binding
	Down     key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultListBaseKeys() listBaseKeys {
	return listBaseKeys{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearFlt: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ActionBinding defines a key-triggered action in a ListModel.
// Apply is called immediately on key press and returns a result value stored in the model.
// NeedsConfirm, if set, shows a confirmation dialog; empty return means quit immediately.
type ActionBinding[T any] struct {
	Binding      key.Binding
	Apply        func(item T) any
	NeedsConfirm func(item T) string
}

// ListConfig configures a ListModel[T].
type ListConfig[T any] struct {
	Title   string
	Columns []table.Column
	// ToRows converts items to table rows.
	ToRows func([]T) []table.Row
	// Matches returns true if item matches the given lowercased filter string.
	// Must return true when lowerFilter is empty.
	Matches    func(item T, lowerFilter string) bool
	ShortHelp  func() string
	FullHelp   func() string
	StatusText func(filtered, total int, filter string) string
	Actions    []ActionBinding[T]

	// ReservedLines is subtracted from window height to compute table height (default 8).
	ReservedLines int

	// Optional hooks for screen-specific extensions.

	// Header is rendered between the title and the filter indicator.
	Header func(m *ListModel[T]) string
	// ExtraBody is rendered between the status bar and the help line.
	ExtraBody func(m *ListModel[T]) string
	// ExtraKeys handles screen-specific key presses in normal mode.
	// Receives a pointer to the model so it can call applyFilter or update table state.
	// Returns true if the key was handled.
	ExtraKeys func(m *ListModel[T], msg tea.KeyMsg) bool
	// OnWindowSize is called after the base WindowSizeMsg handling.
	// Use it to update responsive column widths or adjust table height.
	OnWindowSize func(m *ListModel[T], width, height int)
}

// ListModel is a generic BubbleTea model for interactive filterable lists with actions.
type ListModel[T any] struct {
	cfg         ListConfig[T]
	keys        listBaseKeys
	table       table.Model
	allItems    []T
	filtered    []T
	filter      string
	filtering   bool
	showHelp    bool
	confirmMode bool
	confirmMsg  string
	width       int
	height      int
	quitting    bool
	result      any
}

// newListModelTable builds a table.Model with the shared header/selection styles.
func newListModelTable(columns []table.Column, rows []table.Row) table.Model {
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
	return t
}

// NewListModel creates a new generic list model from items and a config.
func NewListModel[T any](items []T, cfg ListConfig[T]) ListModel[T] {
	return ListModel[T]{
		cfg:      cfg,
		keys:     defaultListBaseKeys(),
		table:    newListModelTable(cfg.Columns, cfg.ToRows(items)),
		allItems: items,
		filtered: items,
	}
}

// Init implements tea.Model.
func (m ListModel[T]) Init() tea.Cmd { return nil }

func (m *ListModel[T]) applyFilter() {
	lf := strings.ToLower(m.filter)
	var filtered []T
	for _, item := range m.allItems {
		if m.cfg.Matches(item, lf) {
			filtered = append(filtered, item)
		}
	}
	m.filtered = filtered
	m.table.SetRows(m.cfg.ToRows(m.filtered))
}

// Update implements tea.Model.
func (m ListModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		reserved := m.cfg.ReservedLines
		if reserved == 0 {
			reserved = 8
		}
		m.table.SetHeight(max(msg.Height-reserved, 5))
		if m.cfg.OnWindowSize != nil {
			m.cfg.OnWindowSize(&m, msg.Width, msg.Height)
		}

	case tea.KeyMsg:
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmMsg = ""
			}
			return m, nil
		}

		if m.filtering {
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
		}

		if m.cfg.ExtraKeys != nil {
			if handled := m.cfg.ExtraKeys(&m, msg); handled {
				return m, nil
			}
		}

		cursor := m.table.Cursor()
		if len(m.filtered) > 0 && cursor >= 0 && cursor < len(m.filtered) {
			item := m.filtered[cursor]
			for _, action := range m.cfg.Actions {
				if key.Matches(msg, action.Binding) {
					m.result = action.Apply(item)
					if action.NeedsConfirm != nil {
						if confirmMsg := action.NeedsConfirm(item); confirmMsg != "" {
							m.confirmMode = true
							m.confirmMsg = confirmMsg
							return m, nil
						}
					}
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m ListModel[T]) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(listStyles.Title.Render(m.cfg.Title))
	b.WriteString("\n\n")

	if m.cfg.Header != nil {
		b.WriteString(m.cfg.Header(&m))
		b.WriteString("\n\n")
	}

	if m.filter != "" || m.filtering {
		filterVal := listStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(listStyles.Filter.Render("Filter: ") + filterVal + "\n\n")
	}

	if m.confirmMode {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(listStyles.Confirm.Render(m.confirmMsg))
		return b.String()
	}

	b.WriteString(m.table.View())
	b.WriteString("\n")

	status := m.cfg.StatusText(len(m.filtered), len(m.allItems), m.filter)
	b.WriteString(listStyles.Status.Render(status))
	b.WriteString("\n")

	if m.cfg.ExtraBody != nil {
		if extra := m.cfg.ExtraBody(&m); extra != "" {
			b.WriteString(extra)
			b.WriteString("\n")
		}
	}

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(listStyles.Help.Render(m.cfg.FullHelp()))
	} else {
		b.WriteString(listStyles.Help.Render(m.cfg.ShortHelp()))
	}

	return b.String()
}
