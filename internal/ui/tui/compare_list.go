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

	"github.com/klauern/skillsync/internal/similarity"
	"github.com/klauern/skillsync/internal/sync"
)

// CompareAction represents the action to perform after compare interaction.
type CompareAction int

const (
	// CompareActionNone means no action was taken (user quit).
	CompareActionNone CompareAction = iota
	// CompareActionView means the user wants to view a detailed comparison.
	CompareActionView
)

// CompareListResult contains the result of the compare list TUI interaction.
type CompareListResult struct {
	Action             CompareAction
	SelectedComparison *similarity.ComparisonResult
}

// compareListKeyMap defines the key bindings for the compare list.
type compareListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	View     key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultCompareListKeyMap() compareListKeyMap {
	return compareListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		View: key.NewBinding(
			key.WithKeys("enter", "v"),
			key.WithHelp("enter/v", "view diff"),
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

// CompareListModel is the BubbleTea model for interactive skill comparison.
type CompareListModel struct {
	table       table.Model
	comparisons []*similarity.ComparisonResult
	filtered    []*similarity.ComparisonResult
	keys        compareListKeyMap
	result      CompareListResult
	filter      string
	filtering   bool
	showHelp    bool
	viewingDiff bool
	viewport    viewport.Model
	width       int
	height      int
	quitting    bool
	ready       bool
}

// Styles for the compare list TUI.
var compareListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Status      lipgloss.Style
	Score       lipgloss.Style
	HighScore   lipgloss.Style
	MedScore    lipgloss.Style
	LowScore    lipgloss.Style
	Header      lipgloss.Style
	Added       lipgloss.Style
	Removed     lipgloss.Style
	Unchanged   lipgloss.Style
	SectionHdr  lipgloss.Style
	Info        lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Score:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	HighScore:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	MedScore:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	LowScore:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	Header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
	Added:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	Removed:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	Unchanged:   lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
	SectionHdr:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Padding(1, 0),
	Info:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true),
}

// NewCompareListModel creates a new compare list model from comparison results.
func NewCompareListModel(comparisons []*similarity.ComparisonResult) CompareListModel {
	columns := []table.Column{
		{Title: "Skill 1", Width: 22},
		{Title: "Platform", Width: 5},
		{Title: "Skill 2", Width: 22},
		{Title: "Platform", Width: 5},
		{Title: "Name%", Width: 6},
		{Title: "Content%", Width: 8},
		{Title: "Changes", Width: 18},
	}

	// Sort by content similarity descending (highest similarity first)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].ContentScore > comparisons[j].ContentScore
	})

	m := CompareListModel{
		comparisons: comparisons,
		filtered:    comparisons,
		keys:        defaultCompareListKeyMap(),
	}

	rows := m.comparisonsToRows(comparisons)

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

func (m CompareListModel) comparisonsToRows(comparisons []*similarity.ComparisonResult) []table.Row {
	rows := make([]table.Row, len(comparisons))
	for i, c := range comparisons {
		name1 := c.Skill1.Name
		if len(name1) > 22 {
			name1 = name1[:19] + "..."
		}

		name2 := c.Skill2.Name
		if len(name2) > 22 {
			name2 = name2[:19] + "..."
		}

		plat1 := c.Skill1.Platform.Short()
		plat2 := c.Skill2.Platform.Short()

		nameScore := "-"
		if c.NameScore > 0 {
			nameScore = fmt.Sprintf("%.0f%%", c.NameScore*100)
		}

		contentScore := "-"
		if c.ContentScore > 0 {
			contentScore = fmt.Sprintf("%.0f%%", c.ContentScore*100)
		}

		changes := c.DiffSummary()

		rows[i] = table.Row{
			name1,
			plat1,
			name2,
			plat2,
			nameScore,
			contentScore,
			changes,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m CompareListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m CompareListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.viewingDiff {
			headerHeight := 4
			footerHeight := 3
			viewportHeight := max(msg.Height-headerHeight-footerHeight, 5)

			if !m.ready {
				m.viewport = viewport.New(msg.Width-2, viewportHeight)
				m.ready = true
			} else {
				m.viewport.Width = msg.Width - 2
				m.viewport.Height = viewportHeight
			}
		} else {
			// Adjust table height based on window
			newHeight := max(msg.Height-12, 5)
			m.table.SetHeight(newHeight)
		}

	case tea.KeyMsg:
		// Handle diff viewing mode
		if m.viewingDiff {
			switch msg.String() {
			case "b", "esc":
				m.viewingDiff = false
				m.ready = false
				return m, nil
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			}
			// Pass other keys to viewport for scrolling
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
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

		case key.Matches(msg, m.keys.View):
			if len(m.filtered) > 0 {
				selected := m.getSelectedComparison()
				if selected != nil {
					m.viewingDiff = true
					m.ready = false
					// Initialize viewport on next size message
					m.viewport = viewport.New(m.width-2, max(m.height-12, 10))
					m.viewport.SetContent(m.buildDiffContent(selected))
					m.ready = true
				}
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *CompareListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.comparisons
	} else {
		var filtered []*similarity.ComparisonResult
		lowerFilter := strings.ToLower(m.filter)
		for _, c := range m.comparisons {
			if strings.Contains(strings.ToLower(c.Skill1.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(c.Skill2.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(c.Skill1.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(string(c.Skill2.Platform)), lowerFilter) {
				filtered = append(filtered, c)
			}
		}
		// Maintain sort order by content similarity descending
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].ContentScore > filtered[j].ContentScore
		})
		m.filtered = filtered
	}
	m.table.SetRows(m.comparisonsToRows(m.filtered))
}

func (m CompareListModel) getSelectedComparison() *similarity.ComparisonResult {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return nil
}

func (m CompareListModel) diffContentWidth() int {
	if m.viewport.Width > 0 {
		return m.viewport.Width
	}
	if m.width > 0 {
		return m.width
	}
	return 80
}

func (m CompareListModel) buildDiffContent(c *similarity.ComparisonResult) string {
	var b strings.Builder
	contentWidth := m.diffContentWidth()

	// Skill 1 info
	b.WriteString(compareListStyles.SectionHdr.Render("Skill 1"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Name:        %s\n", c.Skill1.Name))
	b.WriteString(fmt.Sprintf("  Platform:    %s\n", c.Skill1.Platform))
	b.WriteString(fmt.Sprintf("  Scope:       %s\n", c.Skill1.DisplayScope()))
	if c.Skill1.Description != "" {
		b.WriteString(wrapLabeledText("  Description: ", c.Skill1.Description, contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Skill 2 info
	b.WriteString(compareListStyles.SectionHdr.Render("Skill 2"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Name:        %s\n", c.Skill2.Name))
	b.WriteString(fmt.Sprintf("  Platform:    %s\n", c.Skill2.Platform))
	b.WriteString(fmt.Sprintf("  Scope:       %s\n", c.Skill2.DisplayScope()))
	if c.Skill2.Description != "" {
		b.WriteString(wrapLabeledText("  Description: ", c.Skill2.Description, contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Similarity scores
	b.WriteString(compareListStyles.SectionHdr.Render("Similarity Scores"))
	b.WriteString("\n")
	if c.NameScore > 0 {
		b.WriteString(fmt.Sprintf("  Name similarity:    %.1f%%\n", c.NameScore*100))
	}
	if c.ContentScore > 0 {
		b.WriteString(fmt.Sprintf("  Content similarity: %.1f%%\n", c.ContentScore*100))
	}
	b.WriteString(fmt.Sprintf("  Lines added:        +%d\n", c.LinesAdded))
	b.WriteString(fmt.Sprintf("  Lines removed:      -%d\n", c.LinesRemoved))
	b.WriteString("\n")

	// Diff hunks
	if len(c.Hunks) > 0 {
		b.WriteString(compareListStyles.SectionHdr.Render("Differences"))
		b.WriteString("\n")

		for _, hunk := range c.Hunks {
			// Hunk header
			hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				hunk.SourceStart, hunk.SourceCount,
				hunk.TargetStart, hunk.TargetCount)
			b.WriteString(compareListStyles.Info.Render(hunkHeader))
			b.WriteString("\n")

			// Hunk lines
			for _, line := range hunk.Lines {
				switch line.Type {
				case sync.DiffLineAdded:
					b.WriteString(compareListStyles.Added.Render("+" + line.Content))
				case sync.DiffLineRemoved:
					b.WriteString(compareListStyles.Removed.Render("-" + line.Content))
				default:
					b.WriteString(compareListStyles.Unchanged.Render(" " + line.Content))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	} else if c.Skill1.Content == c.Skill2.Content {
		b.WriteString(compareListStyles.Info.Render("  Contents are identical"))
		b.WriteString("\n")
	} else {
		b.WriteString(compareListStyles.SectionHdr.Render("Content Preview"))
		b.WriteString("\n")
		b.WriteString(compareListStyles.Info.Render("  (No diff hunks computed - showing content summary)"))
		b.WriteString("\n\n")

		srcLines := strings.Count(c.Skill1.Content, "\n") + 1
		tgtLines := strings.Count(c.Skill2.Content, "\n") + 1
		b.WriteString(fmt.Sprintf("  Skill 1: %d lines\n", srcLines))
		b.WriteString(fmt.Sprintf("  Skill 2: %d lines\n", tgtLines))
	}

	return b.String()
}

// View implements tea.Model.
func (m CompareListModel) View() string {
	if m.quitting {
		return ""
	}

	// Diff viewing mode
	if m.viewingDiff {
		return m.viewDiff()
	}

	// List view mode
	var b strings.Builder

	// Title
	title := compareListStyles.Title.Render("🔍 Compare Skills - Side-by-Side Comparison")
	b.WriteString(title)
	b.WriteString("\n")

	// Info message
	info := compareListStyles.Status.Render("Select a pair to view detailed comparison. Press Enter or v to view.")
	b.WriteString(info)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := compareListStyles.Filter.Render("Filter: ")
		filterVal := compareListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	status := fmt.Sprintf("Showing %d similar skill pair(s)", len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d of %d pair(s) (filtered)", len(m.filtered), len(m.comparisons))
	}
	b.WriteString(compareListStyles.Status.Render(status))
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

func (m CompareListModel) viewDiff() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	selected := m.getSelectedComparison()
	titleText := "Comparison Details"
	if selected != nil {
		titleText = fmt.Sprintf("📄 %s ↔ %s", selected.Skill1.Name, selected.Skill2.Name)
	}

	// Title
	title := compareListStyles.Title.Render(titleText)
	b.WriteString(title)
	b.WriteString("\n\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Status bar
	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	status := fmt.Sprintf("Scroll: %d%% • Press b or Esc to go back", scrollPercent)
	b.WriteString(compareListStyles.Status.Render(status))
	b.WriteString("\n")

	// Help
	if m.showHelp {
		help := m.renderDiffHelp()
		b.WriteString("\n")
		b.WriteString(help)
	} else {
		keys := []string{
			"↑/↓ scroll",
			"b back",
			"? help",
			"q quit",
		}
		b.WriteString(compareListStyles.Help.Render(strings.Join(keys, " • ")))
	}

	return b.String()
}

func (m CompareListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter view",
		"/ filter",
		"? help",
		"q quit",
	}
	return compareListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m CompareListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Actions:
  Enter/v  View detailed comparison for selected pair

Filter:
  /        Start filtering (by skill name or platform)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit

Tip: Use the comparison view to see exact differences between similar skills!`
	return compareListStyles.Help.Render(help)
}

func (m CompareListModel) renderDiffHelp() string {
	help := `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  PgUp     Page up
  PgDown   Page down

Actions:
  b/Esc    Go back to list

General:
  ?        Toggle full help
  q        Quit`
	return compareListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m CompareListModel) Result() CompareListResult {
	return m.result
}

// RunCompareList runs the interactive compare list and returns the result.
func RunCompareList(comparisons []*similarity.ComparisonResult) (CompareListResult, error) {
	if len(comparisons) == 0 {
		return CompareListResult{}, nil
	}

	mdl := NewCompareListModel(comparisons)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return CompareListResult{}, err
	}

	if m, ok := finalModel.(CompareListModel); ok {
		return m.Result(), nil
	}

	return CompareListResult{}, nil
}
