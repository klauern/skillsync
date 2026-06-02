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
	// CompareActionDedupe means the user wants to proceed to interactive
	// deletion of the duplicates (distinct from quitting).
	CompareActionDedupe
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
	Dedupe   key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

type compareListColumnWidths struct {
	name      int
	platform  int
	nameScore int
	content   int
	changes   int
}

func defaultCompareListColumnWidths() compareListColumnWidths {
	return compareListColumnWidths{
		name:      24,
		platform:  5,
		nameScore: 6,
		content:   8,
		changes:   20,
	}
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
		Dedupe: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "dedupe"),
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
	ListModel[*similarity.ComparisonResult]
	hScroll     horizontalTableState
	keys        compareListKeyMap
	result      CompareListResult
	viewingDiff bool
	viewport    viewport.Model
	ready       bool
}

// Styles for the compare list TUI.
var compareListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Status      lipgloss.Style
	Description lipgloss.Style
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
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
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
func NewCompareListModel(comparisons []*similarity.ComparisonResult) *CompareListModel {
	columnWidths := defaultCompareListColumnWidths()
	columns := []table.Column{
		{Title: "Skill 1", Width: columnWidths.name},
		{Title: "Platform", Width: columnWidths.platform},
		{Title: "Skill 2", Width: columnWidths.name},
		{Title: "Platform", Width: columnWidths.platform},
		{Title: "Name%", Width: columnWidths.nameScore},
		{Title: "Content%", Width: columnWidths.content},
		{Title: "Changes", Width: columnWidths.changes},
	}

	// Sort by content similarity descending (highest similarity first)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].ContentScore > comparisons[j].ContentScore
	})

	m := &CompareListModel{
		keys:    defaultCompareListKeyMap(),
		hScroll: newHorizontalTableState(columns),
	}

	m.ListModel = NewListModel(comparisons, ListConfig[*similarity.ComparisonResult]{
		Title:   "🔍 Compare Skills - Side-by-Side Comparison",
		Columns: columns,
		ToRows: func(items []*similarity.ComparisonResult) []table.Row {
			return m.comparisonsToRows(items)
		},
		Matches: func(item *similarity.ComparisonResult, lowerFilter string) bool {
			if lowerFilter == "" {
				return true
			}
			return strings.Contains(strings.ToLower(item.Skill1.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(item.Skill2.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(item.Skill1.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(string(item.Skill2.Platform)), lowerFilter)
		},
		ShortHelp: func() string { return m.renderShortHelp() },
		FullHelp:  func() string { return m.renderFullHelp() },
		StatusText: func(filtered, total int, filter string) string {
			status := fmt.Sprintf("Showing %d similar skill pair(s)", filtered)
			if filter != "" {
				status = fmt.Sprintf("%d of %d pair(s) (filtered)", filtered, total)
			}
			if scrollStatus := m.hScroll.Summary(m.width); scrollStatus != "" {
				status += " • " + scrollStatus
			}
			return status
		},
		Header: func() string {
			return compareListStyles.Status.Render("Select a pair to view detailed comparison. Press Enter or v to view.")
		},
		ExtraBody: func(lm *ListModel[*similarity.ComparisonResult]) string {
			return m.listExtraBody()
		},
		ExtraKeys: func(lm *ListModel[*similarity.ComparisonResult], msg tea.KeyMsg) bool {
			return m.handleExtraKeys(msg)
		},
		ReservedLines: 12,
		OnWindowSize: func(lm *ListModel[*similarity.ComparisonResult], width, height int) {
			m.onWindowSize(width, height)
		},
	})

	return m
}

func (m CompareListModel) comparisonsToRows(comparisons []*similarity.ComparisonResult) []table.Row {
	rows := make([]table.Row, len(comparisons))
	for i, c := range comparisons {
		nameScore := "-"
		if c.NameScore > 0 {
			nameScore = fmt.Sprintf("%.0f%%", c.NameScore*100)
		}

		contentScore := "-"
		if c.ContentScore > 0 {
			contentScore = fmt.Sprintf("%.0f%%", c.ContentScore*100)
		}

		rows[i] = table.Row{
			truncateTableValue(c.Skill1.Name, 22),
			c.Skill1.Platform.Short(),
			truncateTableValue(c.Skill2.Name, 22),
			c.Skill2.Platform.Short(),
			nameScore,
			contentScore,
			truncateTableValue(c.DiffSummary(), 18),
		}
	}
	return rows
}

// Init implements tea.Model.
func (m CompareListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *CompareListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.viewingDiff {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.onWindowSize(msg.Width, msg.Height)
			return m, nil
		case tea.KeyMsg:
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
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Proceed to interactive dedupe on an explicit 'd' (not while filtering,
	// where 'd' is literal input). This is distinct from quitting: 'q'/ctrl+c
	// leave Action as CompareActionNone so the caller does not open the dedupe
	// TUI on a quit.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.filtering && key.Matches(keyMsg, m.keys.Dedupe) {
		m.result = CompareListResult{Action: CompareActionDedupe}
		m.quitting = true
		return m, tea.Quit
	}

	updated, cmd := m.ListModel.Update(msg)
	if lm, ok := updated.(ListModel[*similarity.ComparisonResult]); ok {
		m.ListModel = lm
	}
	m.refreshTable()
	return m, cmd
}

func (m *CompareListModel) refreshTable() {
	m.hScroll.Apply(&m.table, m.width, m.comparisonsToRows(m.filtered))
}

func (m *CompareListModel) applyFilter() {
	m.ListModel.applyFilter()
	m.refreshTable()
}

func (m CompareListModel) getSelectedComparison() *similarity.ComparisonResult {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return nil
}

func (m *CompareListModel) handleExtraKeys(msg tea.KeyMsg) bool {
	switch {
	case msg.String() == "left" || msg.String() == "h":
		if m.hScroll.MoveLeft() {
			m.refreshTable()
		}
		return true
	case msg.String() == "right" || msg.String() == "l":
		if m.hScroll.MoveRight(m.width) {
			m.refreshTable()
		}
		return true
	case key.Matches(msg, m.keys.View):
		if len(m.filtered) > 0 {
			selected := m.getSelectedComparison()
			if selected != nil {
				m.viewingDiff = true
				m.ready = false
				m.viewport = viewport.New(m.width-2, max(m.height-12, 10))
				m.viewport.SetContent(m.buildDiffContent(selected))
				m.ready = true
			}
		}
		return true
	default:
		return false
	}
}

func (m CompareListModel) listExtraBody() string {
	selected := m.getSelectedComparison()
	if selected == nil || (selected.Skill1.Description == "" && selected.Skill2.Description == "") {
		return ""
	}

	descWidth := max(m.width-2, 40)
	var b strings.Builder
	if selected.Skill1.Description != "" {
		formatted := formatDetail("Skill 1: ", selected.Skill1.Description, descWidth)
		b.WriteString(compareListStyles.Description.Render(formatted))
		b.WriteString("\n")
	}
	if selected.Skill2.Description != "" {
		formatted := formatDetail("Skill 2: ", selected.Skill2.Description, descWidth)
		b.WriteString(compareListStyles.Description.Render(formatted))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *CompareListModel) onWindowSize(width, height int) {
	if m.viewingDiff {
		headerHeight := 4
		footerHeight := 3
		viewportHeight := max(height-headerHeight-footerHeight, 5)
		if !m.ready {
			m.viewport = viewport.New(width-2, viewportHeight)
			if selected := m.getSelectedComparison(); selected != nil {
				m.viewport.SetContent(m.buildDiffContent(selected))
			}
			m.ready = true
		} else {
			m.viewport.Width = width - 2
			m.viewport.Height = viewportHeight
		}
		return
	}
	m.refreshTable()
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
	fmt.Fprintf(&b, "  Name:        %s\n", c.Skill1.Name)
	fmt.Fprintf(&b, "  Platform:    %s\n", c.Skill1.Platform)
	fmt.Fprintf(&b, "  Scope:       %s\n", c.Skill1.DisplayScope())
	if c.Skill1.Description != "" {
		b.WriteString(wrapLabeledText("  Description: ", c.Skill1.Description, contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Skill 2 info
	b.WriteString(compareListStyles.SectionHdr.Render("Skill 2"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Name:        %s\n", c.Skill2.Name)
	fmt.Fprintf(&b, "  Platform:    %s\n", c.Skill2.Platform)
	fmt.Fprintf(&b, "  Scope:       %s\n", c.Skill2.DisplayScope())
	if c.Skill2.Description != "" {
		b.WriteString(wrapLabeledText("  Description: ", c.Skill2.Description, contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Similarity scores
	b.WriteString(compareListStyles.SectionHdr.Render("Similarity Scores"))
	b.WriteString("\n")
	if c.NameScore > 0 {
		fmt.Fprintf(&b, "  Name similarity:    %.1f%%\n", c.NameScore*100)
	}
	if c.ContentScore > 0 {
		fmt.Fprintf(&b, "  Content similarity: %.1f%%\n", c.ContentScore*100)
	}
	fmt.Fprintf(&b, "  Lines added:        +%d\n", c.LinesAdded)
	fmt.Fprintf(&b, "  Lines removed:      -%d\n", c.LinesRemoved)
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
		fmt.Fprintf(&b, "  Skill 1: %d lines\n", srcLines)
		fmt.Fprintf(&b, "  Skill 2: %d lines\n", tgtLines)
	}

	return b.String()
}

// View implements tea.Model.
func (m *CompareListModel) View() string {
	if m.quitting {
		return ""
	}
	if m.viewingDiff {
		return m.viewDiff()
	}
	return m.ListModel.View()
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
		"←/→ columns",
		"enter view",
		"d dedupe",
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
  ←/h      Show previous columns
  →/l      Show next columns
  g/Home   Go to top
  G/End    Go to bottom

Actions:
  Enter/v  View detailed comparison for selected pair
  d        Proceed to interactive dedupe (select and delete duplicates)

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

	if m, ok := finalModel.(*CompareListModel); ok {
		return m.Result(), nil
	}

	return CompareListResult{}, nil
}
