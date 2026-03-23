package ui

import (
	"fmt"
	"maps"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/mouse"
)

type AccessoryEnvironmentSubmitMsg struct {
	EnvVars map[string]string
}

type AccessoryEnvironmentCancelMsg struct{}

type AccessoryEnvironment struct {
	settingsFormBase
	width    int
	height   int
	scroll   int
	required map[string]struct{}
	envVars  map[string]string
}

func NewAccessoryEnvironment(envVars map[string]string, requiredKeys []string) AccessoryEnvironment {
	var items []FormItem

	keys := slices.Sorted(maps.Keys(envVars))
	required := make(map[string]struct{}, len(requiredKeys))
	for _, k := range requiredKeys {
		required[k] = struct{}{}
	}

	for _, k := range keys {
		_, isRequired := required[k]
		items = append(items, newAccessoryEnvKeyItem(k, isRequired), newAccessoryEnvValueItem(envVars[k], isRequired))
		delete(required, k)
	}
	for k := range required {
		items = append(items, newAccessoryEnvKeyItem(k, true), newAccessoryEnvValueItem("", true))
	}
	items = append(items, newAccessoryEnvKeyItem("", false), newAccessoryEnvValueItem("", false))

	m := AccessoryEnvironment{
		settingsFormBase: settingsFormBase{
			title: "Environment",
			form:  NewForm("Done", items...),
		},
		envVars: envVars,
	}

	m.form.OnRebuild(func(f *Form) {
		lastKeyIdx := f.ItemCount() - 2
		if lastKeyIdx >= 0 && f.TextField(lastKeyIdx).Value() != "" {
			f.AppendItems(newAccessoryEnvKeyItem("", false), newAccessoryEnvValueItem("", false))
		}
	})

	m.form.OnSubmit(func(f *Form) tea.Cmd {
		env := make(map[string]string)
		for i := 0; i < f.ItemCount(); i += 2 {
			k := f.TextField(i).Value()
			if k == "" {
				continue
			}
			if env == nil {
				env = make(map[string]string)
			}
			env[k] = f.TextField(i + 1).Value()
		}
		return func() tea.Msg { return AccessoryEnvironmentSubmitMsg{EnvVars: env} }
	})

	m.form.OnCancel(func(f *Form) tea.Cmd {
		return func() tea.Msg { return AccessoryEnvironmentCancelMsg{} }
	})

	return m
}

func (m AccessoryEnvironment) Update(msg tea.Msg) (Component, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	var cmd tea.Cmd
	m.settingsFormBase, cmd = m.update(msg)
	m.setFieldWidths()
	m.adjustScroll()
	return m, cmd
}

func (m AccessoryEnvironment) View() string {
	return m.renderContent()
}

// Private

func (m AccessoryEnvironment) rowCount() int {
	return m.form.ItemCount() / 2
}

func (m AccessoryEnvironment) columnWidths() (int, int) {
	totalWidth := max(min(m.width, 64), 6)
	keyWidth := totalWidth / 3
	valueWidth := totalWidth - keyWidth - 1
	return keyWidth, valueWidth
}

func (m AccessoryEnvironment) setFieldWidths() {
	keyWidth, valueWidth := m.columnWidths()
	for i := range m.form.ItemCount() {
		if i%2 == 0 {
			m.form.TextField(i).SetWidth(max(keyWidth-4, 1))
		} else {
			m.form.TextField(i).SetWidth(max(valueWidth-4, 1))
		}
	}
}

func (m *AccessoryEnvironment) adjustScroll() {
	maxVisible := m.maxVisibleRows()
	if maxVisible <= 0 {
		return
	}

	focusedRow := m.focusedRow()
	if focusedRow < 0 {
		focusedRow = m.rowCount() - 1
	}

	if focusedRow < m.scroll {
		m.scroll = focusedRow
	}
	if focusedRow >= m.scroll+maxVisible {
		m.scroll = focusedRow - maxVisible + 1
	}
}

func (m AccessoryEnvironment) focusedRow() int {
	focused := m.form.Focused()
	if focused < m.form.ItemCount() {
		return focused / 2
	}
	return -1
}

func (m AccessoryEnvironment) maxVisibleRows() int {
	if m.height <= 0 {
		return m.rowCount()
	}
	available := m.height - 11
	rowHeight := 4
	visible := available / rowHeight
	return max(visible, 1)
}

func (m AccessoryEnvironment) renderContent() string {
	keyWidth, valueWidth := m.columnWidths()

	headerStyle := lipgloss.NewStyle().Bold(true)
	keyHeader := headerStyle.Width(keyWidth).Render("Key")
	valueHeader := headerStyle.Width(valueWidth).Render("Value")
	header := lipgloss.JoinHorizontal(lipgloss.Top, keyHeader, " ", valueHeader)

	var parts []string
	parts = append(parts, header, "")

	maxVisible := m.maxVisibleRows()
	rows := m.rowCount()
	end := min(m.scroll+maxVisible, rows)

	if m.scroll > 0 {
		indicator := lipgloss.NewStyle().Foreground(Colors.Border).
			Render(fmt.Sprintf("↑ %d more above", m.scroll))
		parts = append(parts, indicator)
	}

	focused := m.form.Focused()
	for i := m.scroll; i < end; i++ {
		keyIdx := i * 2
		valIdx := i*2 + 1

		keyStyle := Styles.Focus(Styles.Input, focused == keyIdx).Width(keyWidth)
		valueStyle := Styles.Focus(Styles.Input, focused == valIdx).Width(valueWidth)

		keyView := mouse.Mark(fieldTarget(keyIdx), keyStyle.Render(m.form.TextField(keyIdx).View()))
		valueView := mouse.Mark(fieldTarget(valIdx), valueStyle.Render(m.form.TextField(valIdx).View()))

		rowView := lipgloss.JoinHorizontal(lipgloss.Top, keyView, " ", valueView)
		parts = append(parts, rowView, "")
	}

	if end < rows {
		remaining := rows - end
		indicator := lipgloss.NewStyle().Foreground(Colors.Border).
			Render(fmt.Sprintf("↓ %d more below", remaining))
		parts = append(parts, indicator)
	}

	submitIdx := m.form.ItemCount()
	cancelIdx := m.form.ItemCount() + 1
	submitButton := mouse.Mark("submit", Styles.Focus(Styles.ButtonPrimary, focused == submitIdx).
		Render("Done"))
	cancelButton := mouse.Mark("cancel", Styles.Focus(Styles.Button, focused == cancelIdx).
		Render("Cancel"))
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, submitButton, cancelButton)
	parts = append(parts, buttons)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Helpers

func newAccessoryEnvKeyItem(value string, required bool) FormItem {
	f := NewTextField("KEY")
	f.SetValue(value)
	f.SetCharLimit(256)
	item := FormItem{Field: f, Required: required}
	if value != "" {
		item.Label = value
	}
	return item
}

func newAccessoryEnvValueItem(value string, required bool) FormItem {
	f := NewTextField("value")
	f.SetValue(value)
	f.SetCharLimit(1024)
	return FormItem{Field: f, Required: required}
}
