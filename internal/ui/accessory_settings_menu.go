package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type AccessorySettingsSectionType int

const (
	AccessorySettingsSectionGeneral AccessorySettingsSectionType = iota
	AccessorySettingsSectionEnvironment
	AccessorySettingsSectionStorage
	AccessorySettingsSectionNetwork
	AccessorySettingsSectionHealth
)

var accessorySettingsMenuCloseKey = WithHelp(NewKeyBinding("esc"), "esc", "close")

type AccessorySettingsMenuCloseMsg struct{}

type AccessorySettingsMenuSelectMsg struct {
	accessory *docker.Accessory
	section   AccessorySettingsSectionType
}

type AccessorySettingsMenu struct {
	accessory *docker.Accessory
	menu      Menu
	help      Help
}

func NewAccessorySettingsMenu(accessory *docker.Accessory) AccessorySettingsMenu {
	h := NewHelp()
	h.SetBindings([]key.Binding{accessorySettingsMenuCloseKey})
	return AccessorySettingsMenu{
		accessory: accessory,
		menu: NewMenu(
			MenuItem{Label: "General", Key: int(AccessorySettingsSectionGeneral), Shortcut: WithHelp(NewKeyBinding("g"), "g", "")},
			MenuItem{Label: "Environment", Key: int(AccessorySettingsSectionEnvironment), Shortcut: WithHelp(NewKeyBinding("e"), "e", "")},
			MenuItem{Label: "Storage", Key: int(AccessorySettingsSectionStorage), Shortcut: WithHelp(NewKeyBinding("s"), "s", "")},
			MenuItem{Label: "Network", Key: int(AccessorySettingsSectionNetwork), Shortcut: WithHelp(NewKeyBinding("n"), "n", "")},
			MenuItem{Label: "Health", Key: int(AccessorySettingsSectionHealth), Shortcut: WithHelp(NewKeyBinding("h"), "h", "")},
		),
		help: h,
	}
}

func (m AccessorySettingsMenu) Init() tea.Cmd { return nil }

func (m AccessorySettingsMenu) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case MouseEvent:
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case tea.KeyPressMsg:
		if key.Matches(msg, accessorySettingsMenuCloseKey) {
			return m, func() tea.Msg { return AccessorySettingsMenuCloseMsg{} }
		}
	case MenuSelectMsg:
		return m, m.selectSection(AccessorySettingsSectionType(msg.Key))
	}

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m AccessorySettingsMenu) View() string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Colors.Border).Padding(1, 4)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(Colors.Primary).MarginBottom(1)
	title := titleStyle.Render("Accessory Settings")
	m.menu.SetWidth(24)
	menuView := m.menu.View()
	helpView := m.help.View()
	menuWidth := lipgloss.Width(menuView)
	helpLine := lipgloss.NewStyle().MarginTop(1).Width(menuWidth).Align(lipgloss.Center).Render(helpView)
	content := lipgloss.JoinVertical(lipgloss.Center, title, menuView, helpLine)
	return boxStyle.Render(content)
}

// Private

func (m AccessorySettingsMenu) selectSection(section AccessorySettingsSectionType) tea.Cmd {
	return func() tea.Msg {
		return AccessorySettingsMenuSelectMsg{accessory: m.accessory, section: section}
	}
}
