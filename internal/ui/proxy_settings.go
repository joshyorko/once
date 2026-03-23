package ui

import (
	"context"
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type ProxySettingsSubmitMsg struct {
	Settings docker.ProxySettings
}

type ProxySettingsCancelMsg struct{}

type ProxySettings struct {
	namespace     *docker.Namespace
	width, height int
	help          Help
	form          Form
	err           error
}

func NewProxySettings(ns *docker.Namespace) ProxySettings {
	current := docker.ProxySettings{
		BindAddress: "0.0.0.0",
		HTTPPort:    docker.DefaultHTTPPort,
		HTTPSPort:   docker.DefaultHTTPSPort,
		MetricsPort: docker.DefaultMetricsPort,
	}
	if ns.Proxy().Settings != nil {
		current = *ns.Proxy().Settings
	}

	bindField := NewTextField("0.0.0.0")
	bindField.SetValue(current.BindAddress)

	httpField := NewTextField("80")
	httpField.SetDigitsOnly(true)
	httpField.SetValue(strconv.Itoa(current.HTTPPort))

	httpsField := NewTextField("443")
	httpsField.SetDigitsOnly(true)
	httpsField.SetValue(strconv.Itoa(current.HTTPSPort))

	metricsField := NewTextField("1318")
	metricsField.SetDigitsOnly(true)
	metricsField.SetValue(strconv.Itoa(current.MetricsPort))

	f := NewForm("Save",
		FormItem{Label: "Bind", Field: bindField},
		FormItem{Label: "HTTP", Field: httpField},
		FormItem{Label: "HTTPS", Field: httpsField},
		FormItem{Label: "Metrics", Field: metricsField},
	)
	f.OnSubmit(func(f *Form) tea.Cmd {
		httpPort, _ := strconv.Atoi(f.TextField(1).Value())
		httpsPort, _ := strconv.Atoi(f.TextField(2).Value())
		metricsPort, _ := strconv.Atoi(f.TextField(3).Value())
		return func() tea.Msg {
			return ProxySettingsSubmitMsg{
				Settings: docker.ProxySettings{
					BindAddress: f.TextField(0).Value(),
					HTTPPort:    httpPort,
					HTTPSPort:   httpsPort,
					MetricsPort: metricsPort,
				},
			}
		}
	})
	f.OnCancel(func(f *Form) tea.Cmd {
		return func() tea.Msg { return ProxySettingsCancelMsg{} }
	})

	h := NewHelp()
	h.SetBindings([]key.Binding{settingsKeys.Back})
	return ProxySettings{
		namespace: ns,
		help:      h,
		form:      f,
	}
}

func (m ProxySettings) Init() tea.Cmd { return m.form.Init() }

func (m ProxySettings) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(m.width)
		m.form, _ = m.form.Update(msg)
		return m, nil
	case MouseEvent:
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case tea.KeyPressMsg:
		if key.Matches(msg, settingsKeys.Back) {
			return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
		}
	case ProxySettingsCancelMsg:
		return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
	case ProxySettingsSubmitMsg:
		if err := m.namespace.EnsureNetwork(context.Background()); err != nil {
			m.err = err
			return m, nil
		}
		if err := m.namespace.Proxy().ApplySettings(context.Background(), msg.Settings); err != nil {
			m.err = err
			return m, nil
		}
		return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m ProxySettings) View() string {
	titleLine := Styles.TitleRule(m.width, "proxy", "settings")
	formView := m.form.View()
	helpLine := Styles.CenteredLine(m.width, m.help.View())
	middleHeight := m.height - 2 - lipgloss.Height(helpLine)
	centeredContent := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, formView)
	if m.err != nil {
		centeredContent = lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Foreground(Colors.Error).Render(docker.ErrorMessage(m.err)), "", centeredContent)
	}
	return titleLine + "\n\n" + centeredContent + "\n" + helpLine
}
