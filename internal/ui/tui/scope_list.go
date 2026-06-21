// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/klauern/skillsync/internal/model"
)

// ScopeAction represents the action to perform after scope interaction.
type ScopeAction int

const (
	// ScopeActionNone means no action was taken (user quit).
	ScopeActionNone ScopeAction = iota
	// ScopeActionView means the user wants to view skill details.
	ScopeActionView
)

// ScopeListResult contains the result of the scope list TUI interaction.
type ScopeListResult struct {
	Action        ScopeAction
	SelectedSkill model.Skill
}

// scopeListStyles includes the scope-tab-specific styles on top of the shared listStyles.
var scopeListStyles = struct {
	ScopeTab    lipgloss.Style
	ScopeActive lipgloss.Style
	Description lipgloss.Style
}{
	ScopeTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	ScopeActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

// ScopeListModel is the BubbleTea model for interactive scope management.
type ScopeListModel struct {
	ListModel[model.Skill]
}

// Update wraps the base Update and preserves the ScopeListModel type.
func (m ScopeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[model.Skill])
	return m, cmd
}

// Result returns the result of the user interaction.
func (m ScopeListModel) Result() ScopeListResult {
	if r, ok := m.result.(ScopeListResult); ok {
		return r
	}
	return ScopeListResult{}
}

// NewScopeListModel creates a new scope list model.
func NewScopeListModel(skills []model.Skill) ScopeListModel {
	// Collect unique scopes in precedence order.
	scopeSet := make(map[model.SkillScope]bool)
	for _, s := range skills {
		scopeSet[s.Scope] = true
	}
	var scopeOptions []model.SkillScope
	for _, scope := range model.AllScopes() {
		if scopeSet[scope] {
			scopeOptions = append(scopeOptions, scope)
		}
	}

	// scopeIndex lives outside the model so closures share a single mutable reference.
	scopeIndex := -1 // -1 = all

	// Responsive column widths, also shared via closures.
	columnWidths := struct{ name, platform, scope, desc int }{25, 12, 40, 50}

	toRows := func(s []model.Skill) []table.Row {
		rows := make([]table.Row, len(s))
		for i, sk := range s {
			rows[i] = table.Row{
				truncateTableValue(sk.Name, columnWidths.name),
				truncateTableValue(string(sk.Platform), columnWidths.platform),
				truncateTableValue(sk.DisplayScope(), columnWidths.scope),
				truncateTableValue(sk.Description, columnWidths.desc),
			}
		}
		return rows
	}

	nextScopeKey := key.NewBinding(key.WithKeys("tab", "l"), key.WithHelp("tab/l", "next scope"))
	prevScopeKey := key.NewBinding(key.WithKeys("shift+tab", "h"), key.WithHelp("S-tab/h", "prev scope"))
	viewKey := key.NewBinding(key.WithKeys("enter", "v"), key.WithHelp("enter/v", "view details"))

	titleCaser := cases.Title(language.English)

	cfg := ListConfig[model.Skill]{
		Title: "📂 Scope Management - Browse Skills by Scope",
		Columns: []table.Column{
			{Title: "Name", Width: columnWidths.name},
			{Title: "Platform", Width: columnWidths.platform},
			{Title: "Scope", Width: columnWidths.scope},
			{Title: "Description", Width: columnWidths.desc},
		},
		ToRows: toRows,
		Matches: func(s model.Skill, lf string) bool {
			if scopeIndex >= 0 && s.Scope != scopeOptions[scopeIndex] {
				return false
			}
			if lf == "" {
				return true
			}
			return strings.Contains(strings.ToLower(s.Name), lf) ||
				strings.Contains(strings.ToLower(string(s.Platform)), lf) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lf) ||
				strings.Contains(strings.ToLower(s.Description), lf)
		},
		ReservedLines: 12,
		Actions: []ActionBinding[model.Skill]{
			{
				Binding: viewKey,
				Apply: func(s model.Skill) any {
					return ScopeListResult{Action: ScopeActionView, SelectedSkill: s}
				},
			},
		},
		StatusText: func(filtered, total int, _ string) string {
			scopeCounts := make(map[model.SkillScope]int)
			for _, s := range skills {
				scopeCounts[s.Scope]++
			}
			var counts []string
			for _, scope := range scopeOptions {
				counts = append(counts, fmt.Sprintf("%s: %d", scope, scopeCounts[scope]))
			}
			status := fmt.Sprintf("Showing %d of %d skills", filtered, total)
			if len(counts) > 0 {
				status += " | " + strings.Join(counts, ", ")
			}
			return status
		},
		Header: func() string {
			var tabs []string
			if scopeIndex == -1 {
				tabs = append(tabs, scopeListStyles.ScopeActive.Render("[All]"))
			} else {
				tabs = append(tabs, scopeListStyles.ScopeTab.Render(" All "))
			}
			for i, scope := range scopeOptions {
				name := titleCaser.String(string(scope))
				if i == scopeIndex {
					tabs = append(tabs, scopeListStyles.ScopeActive.Render(fmt.Sprintf("[%s]", name)))
				} else {
					tabs = append(tabs, scopeListStyles.ScopeTab.Render(fmt.Sprintf(" %s ", name)))
				}
			}
			return strings.Join(tabs, "")
		},
		ExtraBody: func(m *ListModel[model.Skill]) string {
			cursor := m.table.Cursor()
			if cursor < 0 || cursor >= len(m.filtered) {
				return ""
			}
			selected := m.filtered[cursor]
			if selected.Name == "" || selected.Description == "" {
				return ""
			}
			descWidth := max(m.width-2, 40)
			return scopeListStyles.Description.Render(formatDescription(selected.Description, descWidth))
		},
		ExtraKeys: func(m *ListModel[model.Skill], msg tea.KeyMsg) bool {
			switch {
			case key.Matches(msg, nextScopeKey):
				if len(scopeOptions) > 0 {
					scopeIndex++
					if scopeIndex >= len(scopeOptions) {
						scopeIndex = -1
					}
					m.applyFilter()
				}
				return true
			case key.Matches(msg, prevScopeKey):
				if len(scopeOptions) > 0 {
					scopeIndex--
					if scopeIndex < -1 {
						scopeIndex = len(scopeOptions) - 1
					}
					m.applyFilter()
				}
				return true
			}
			return false
		},
		OnWindowSize: func(m *ListModel[model.Skill], width, _ int) {
			const separatorWidth = 6
			newDesc := max(width-(columnWidths.name+columnWidths.platform+columnWidths.scope+separatorWidth), 40)
			columnWidths.desc = newDesc
			m.table.SetColumns([]table.Column{
				{Title: "Name", Width: columnWidths.name},
				{Title: "Platform", Width: columnWidths.platform},
				{Title: "Scope", Width: columnWidths.scope},
				{Title: "Description", Width: columnWidths.desc},
			})
			m.table.SetRows(m.cfg.ToRows(m.filtered))
		},
		ShortHelp: func() string {
			return strings.Join([]string{
				"↑/↓ navigate", "tab/S-tab scope", "enter view", "/ filter", "? help", "q quit",
			}, " • ")
		},
		FullHelp: func() string {
			return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Scope Filtering:
  Tab/l       Next scope
  Shift-Tab/h Previous scope

Actions:
  Enter/v  View skill details

Text Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit`
		},
	}

	return ScopeListModel{ListModel: NewListModel(skills, cfg)}
}

// RunScopeList runs the interactive scope list and returns the result.
func RunScopeList(skills []model.Skill) (ScopeListResult, error) {
	if len(skills) == 0 {
		return ScopeListResult{}, nil
	}

	mdl := NewScopeListModel(skills)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ScopeListResult{}, err
	}

	if m, ok := finalModel.(ScopeListModel); ok {
		return m.Result(), nil
	}

	return ScopeListResult{}, nil
}
