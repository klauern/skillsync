package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/klauern/skillsync/internal/model"
)

// selectableListModel holds the shared state and behavior for TUI lists that
// support filtering, row selection, and table rendering.
type selectableListModel[T any] struct {
	table     table.Model
	skills    []T
	filtered  []T
	selected  map[string]bool
	filter    string
	filtering bool

	keyFn   func(T) string
	matchFn func(T, string) bool
	rowFn   func(T, bool) table.Row
}

func newSelectableListModel[T any](skills []T, selectedAll bool, columns []table.Column, height int, keyFn func(T) string, matchFn func(T, string) bool, rowFn func(T, bool) table.Row) selectableListModel[T] {
	selected := make(map[string]bool, len(skills))
	if selectedAll {
		for _, skill := range skills {
			selected[keyFn(skill)] = true
		}
	}

	m := selectableListModel[T]{
		skills:   skills,
		filtered: skills,
		selected: selected,
		keyFn:    keyFn,
		matchFn:  matchFn,
		rowFn:    rowFn,
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(m.skillsToRows(skills)),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	m.table = t
	return m
}

func (m selectableListModel[T]) skillsToRows(skills []T) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, skill := range skills {
		rows[i] = m.rowFn(skill, m.selected[m.keyFn(skill)])
	}
	return rows
}

func (m *selectableListModel[T]) refreshTable() {
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m selectableListModel[T]) getSelectedSkill() T {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	var zero T
	return zero
}

func (m selectableListModel[T]) getSelectedSkills() []T {
	var selected []T
	for _, skill := range m.skills {
		if m.selected[m.keyFn(skill)] {
			selected = append(selected, skill)
		}
	}
	return selected
}

func (m *selectableListModel[T]) toggleCurrentSelection() {
	if len(m.filtered) == 0 {
		return
	}

	skill := m.getSelectedSkill()
	key := m.keyFn(skill)
	m.selected[key] = !m.selected[key]
	m.refreshTable()
}

func (m *selectableListModel[T]) toggleAllSelection() {
	if len(m.filtered) == 0 {
		return
	}

	selectedCount := 0
	for _, skill := range m.filtered {
		if m.selected[m.keyFn(skill)] {
			selectedCount++
		}
	}

	selectAll := selectedCount < len(m.filtered)/2+1
	for _, skill := range m.filtered {
		m.selected[m.keyFn(skill)] = selectAll
	}
	m.refreshTable()
}

// skillListFilterMatch returns true when the lowercase filter matches the skill.
func skillListFilterMatch(skill model.Skill, lowerFilter string) bool {
	return strings.Contains(strings.ToLower(skill.Name), lowerFilter) ||
		strings.Contains(strings.ToLower(string(skill.Platform)), lowerFilter) ||
		strings.Contains(strings.ToLower(skill.DisplayScope()), lowerFilter) ||
		strings.Contains(strings.ToLower(skill.Description), lowerFilter)
}
