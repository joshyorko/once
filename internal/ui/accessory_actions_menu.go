package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

var accessoryActionsMenuCloseKey = WithHelp(NewKeyBinding("esc"), "esc", "close")

type AccessoryActionsMenuCloseMsg struct{}

type AccessoryActionsMenuSelectMsg struct {
	accessory *docker.Accessory
	action    AccessoryActionsMenuAction
}

type AccessoryActionsMenuAction int

const (
	AccessoryActionsMenuStartStop AccessoryActionsMenuAction = iota
	AccessoryActionsMenuRemove
)

type AccessoryActionsMenu struct {
	accessory *docker.Accessory
	menu      Menu
	help      Help
}

func NewAccessoryActionsMenu(accessory *docker.Accessory) AccessoryActionsMenu {
	startStopLabel := "Start"
	if accessory.Running {
		startStopLabel = "Stop"
	}

	h := NewHelp()
	h.SetBindings([]key.Binding{accessoryActionsMenuCloseKey})
	return AccessoryActionsMenu{
		accessory: accessory,
		menu: NewMenu(
			MenuItem{Label: startStopLabel, Key: int(AccessoryActionsMenuStartStop), Shortcut: WithHelp(NewKeyBinding("s"), "s", "")},
			MenuItem{Label: "Remove", Key: int(AccessoryActionsMenuRemove), Shortcut: WithHelp(NewKeyBinding("r"), "r", "")},
		),
		help: h,
	}
}

func (m AccessoryActionsMenu) Init() tea.Cmd { return nil }

func (m AccessoryActionsMenu) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case MouseEvent:
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case tea.KeyPressMsg:
		if key.Matches(msg, accessoryActionsMenuCloseKey) {
			return m, func() tea.Msg { return AccessoryActionsMenuCloseMsg{} }
		}
	case MenuSelectMsg:
		return m, m.selectAction(AccessoryActionsMenuAction(msg.Key))
	}

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m AccessoryActionsMenu) View() string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Colors.Border).Padding(1, 4)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(Colors.Primary).MarginBottom(1)
	title := titleStyle.Render("Accessory Actions")
	m.menu.SetWidth(24)
	menuView := m.menu.View()
	helpView := m.help.View()
	menuWidth := lipgloss.Width(menuView)
	helpLine := lipgloss.NewStyle().MarginTop(1).Width(menuWidth).Align(lipgloss.Center).Render(helpView)
	content := lipgloss.JoinVertical(lipgloss.Center, title, menuView, helpLine)
	return boxStyle.Render(content)
}

// Private

func (m AccessoryActionsMenu) selectAction(action AccessoryActionsMenuAction) tea.Cmd {
	return func() tea.Msg {
		return AccessoryActionsMenuSelectMsg{accessory: m.accessory, action: action}
	}
}
