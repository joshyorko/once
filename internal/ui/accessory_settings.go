package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type AccessorySettingsSubmitMsg struct {
	Settings docker.AccessorySettings
}

type AccessorySettingsCancelMsg struct{}

type AccessorySettings struct {
	namespace     *docker.Namespace
	accessory     *docker.Accessory
	section       AccessorySettingsSectionType
	width, height int
	help          Help
	form          Form
	env           Component
	err           error
	saving        bool
}

type accessorySettingsDeployFinishedMsg struct {
	err error
}

func NewAccessorySettings(ns *docker.Namespace, accessory *docker.Accessory, section AccessorySettingsSectionType) AccessorySettings {
	h := NewHelp()
	h.SetBindings([]key.Binding{settingsKeys.Back})

	if section == AccessorySettingsSectionEnvironment {
		return AccessorySettings{
			namespace: ns,
			accessory: accessory,
			section:   section,
			help:      h,
			env:       NewAccessoryEnvironment(accessory.Settings.EnvVars, nil),
		}
	}

	if section == AccessorySettingsSectionStorage {
		volumeField := NewTextField("data:/data:ro")
		volumeField.SetValue(accessoryMountField(accessory.Settings.Mounts, docker.AccessoryMountVolume))
		bindField := NewTextField("/host/path:/container/path:ro")
		bindField.SetValue(accessoryMountField(accessory.Settings.Mounts, docker.AccessoryMountBind))

		f := NewForm("Save",
			FormItem{Label: "Named volumes", Field: volumeField},
			FormItem{Label: "Bind mounts", Field: bindField},
		)
		f.OnSubmit(func(f *Form) tea.Cmd {
			settings := accessory.Settings
			volumes, err := parseAccessoryMountField(f.TextField(0).Value(), docker.AccessoryMountVolume)
			if err != nil {
				return func() tea.Msg {
					return accessorySettingsDeployFinishedMsg{err: fmt.Errorf("parsing volume mounts: %w", err)}
				}
			}
			binds, err := parseAccessoryMountField(f.TextField(1).Value(), docker.AccessoryMountBind)
			if err != nil {
				return func() tea.Msg {
					return accessorySettingsDeployFinishedMsg{err: fmt.Errorf("parsing bind mounts: %w", err)}
				}
			}
			settings.Mounts = append(volumes, binds...)
			return func() tea.Msg { return AccessorySettingsSubmitMsg{Settings: settings} }
		})
		f.OnCancel(func(f *Form) tea.Cmd {
			return func() tea.Msg { return AccessorySettingsCancelMsg{} }
		})
		return AccessorySettings{namespace: ns, accessory: accessory, section: section, help: h, form: f}
	}

	if section == AccessorySettingsSectionNetwork {
		proxyHost := NewTextField("app.example.com")
		proxyHost.SetValue(accessory.Settings.Proxy.Host)
		proxyPort := NewTextField("9001")
		proxyPort.SetDigitsOnly(true)
		if accessory.Settings.Proxy.TargetPort != 0 {
			proxyPort.SetValue(strconv.Itoa(accessory.Settings.Proxy.TargetPort))
		}
		tlsField := NewCheckboxField("Enabled", !accessory.Settings.Proxy.DisableTLS)
		publishField := NewTextField("127.0.0.1:9001:9000")
		publishField.SetValue(accessoryPortField(accessory.Settings.Ports))

		f := NewForm("Save",
			FormItem{Label: "Proxy host", Field: proxyHost},
			FormItem{Label: "Proxy port", Field: proxyPort},
			FormItem{Label: "TLS", Field: tlsField},
			FormItem{Label: "Published ports", Field: publishField},
		)
		f.OnSubmit(func(f *Form) tea.Cmd {
			settings := accessory.Settings
			ports, err := parseAccessoryPortField(f.TextField(3).Value())
			if err != nil {
				return func() tea.Msg {
					return accessorySettingsDeployFinishedMsg{err: fmt.Errorf("parsing published ports: %w", err)}
				}
			}
			settings.Proxy.Host = f.TextField(0).Value()
			settings.Proxy.TargetPort, _ = strconv.Atoi(f.TextField(1).Value())
			settings.Proxy.Enabled = settings.Proxy.Host != ""
			settings.Proxy.DisableTLS = !f.CheckboxField(2).Checked()
			settings.Ports = ports
			return func() tea.Msg { return AccessorySettingsSubmitMsg{Settings: settings} }
		})
		f.OnCancel(func(f *Form) tea.Cmd {
			return func() tea.Msg { return AccessorySettingsCancelMsg{} }
		})
		return AccessorySettings{namespace: ns, accessory: accessory, section: section, help: h, form: f}
	}

	if section == AccessorySettingsSectionHealth {
		healthType := NewTextField("none|http|exec")
		healthType.SetValue(string(accessory.Settings.HealthCheck.Type))
		healthPort := NewTextField("80")
		healthPort.SetDigitsOnly(true)
		if accessory.Settings.HealthCheck.Port != 0 {
			healthPort.SetValue(strconv.Itoa(accessory.Settings.HealthCheck.Port))
		}
		healthPath := NewTextField("/up")
		healthPath.SetValue(accessory.Settings.HealthCheck.Path)
		healthCommand := NewTextField("curl -fsS http://localhost/")
		healthCommand.SetValue(accessoryCommandField(accessory.Settings.HealthCheck.Command))

		f := NewForm("Save",
			FormItem{Label: "Type", Field: healthType},
			FormItem{Label: "HTTP port", Field: healthPort},
			FormItem{Label: "HTTP path", Field: healthPath},
			FormItem{Label: "Exec command", Field: healthCommand},
		)
		f.OnSubmit(func(f *Form) tea.Cmd {
			settings := accessory.Settings
			settings.HealthCheck.Type = docker.AccessoryHealthCheckType(strings.TrimSpace(f.TextField(0).Value()))
			settings.HealthCheck.Port, _ = strconv.Atoi(f.TextField(1).Value())
			settings.HealthCheck.Path = f.TextField(2).Value()
			settings.HealthCheck.Command = parseAccessoryCommandField(f.TextField(3).Value())
			return func() tea.Msg { return AccessorySettingsSubmitMsg{Settings: settings} }
		})
		f.OnCancel(func(f *Form) tea.Cmd {
			return func() tea.Msg { return AccessorySettingsCancelMsg{} }
		})
		return AccessorySettings{namespace: ns, accessory: accessory, section: section, help: h, form: f}
	}

	imageField := NewTextField("user/repo:tag")
	imageField.SetValue(accessory.Settings.Image)
	commandField := NewTextField("tunnel run")
	commandField.SetValue(accessoryCommandField(accessory.Settings.Command))
	valueStyle := lipgloss.NewStyle().Foreground(Colors.Border)
	renderValue := func(value string) string { return valueStyle.Render(value) }
	nameField := NewStaticField(accessory.Settings.Name, renderValue)
	scopeField := NewStaticField(string(accessory.Settings.Scope), renderValue)
	ownerField := NewStaticField(accessory.Settings.OwnerApp, renderValue)

	restartField := NewTextField("always")
	restartField.SetValue(accessory.Settings.RestartPolicy)

	f := NewForm("Save",
		FormItem{Label: "Name", Field: nameField},
		FormItem{Label: "Scope", Field: scopeField},
		FormItem{Label: "Owner app", Field: ownerField},
		FormItem{Label: "Image", Field: imageField},
		FormItem{Label: "Command", Field: commandField},
		FormItem{Label: "Restart", Field: restartField},
	)
	f.OnSubmit(func(f *Form) tea.Cmd {
		settings := accessory.Settings
		settings.Image = f.TextField(3).Value()
		settings.Command = parseAccessoryCommandField(f.TextField(4).Value())
		settings.RestartPolicy = strings.TrimSpace(f.TextField(5).Value())
		return func() tea.Msg { return AccessorySettingsSubmitMsg{Settings: settings} }
	})
	f.OnCancel(func(f *Form) tea.Cmd {
		return func() tea.Msg { return AccessorySettingsCancelMsg{} }
	})

	return AccessorySettings{
		namespace: ns,
		accessory: accessory,
		section:   section,
		help:      h,
		form:      f,
	}
}

func (m AccessorySettings) Init() tea.Cmd {
	if m.section == AccessorySettingsSectionEnvironment && m.env != nil {
		return m.env.Init()
	}
	return m.form.Init()
}

func (m AccessorySettings) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(m.width)
		if m.section == AccessorySettingsSectionEnvironment {
			m.env, _ = m.env.Update(msg)
			return m, nil
		}
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
	case AccessorySettingsCancelMsg:
		return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
	case AccessoryEnvironmentSubmitMsg:
		m.saving = true
		m.accessory.Settings.EnvVars = msg.EnvVars
		return m, func() tea.Msg {
			err := m.accessory.Deploy(context.Background(), nil)
			return accessorySettingsDeployFinishedMsg{err: err}
		}
	case AccessoryEnvironmentCancelMsg:
		return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
	case AccessorySettingsSubmitMsg:
		m.saving = true
		m.accessory.Settings = msg.Settings
		return m, func() tea.Msg {
			err := m.accessory.Deploy(context.Background(), nil)
			return accessorySettingsDeployFinishedMsg{err: err}
		}
	case accessorySettingsDeployFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.saving = false
			return m, nil
		}
		return m, func() tea.Msg { return NavigateToAccessoryLogsMsg{Accessory: m.accessory} }
	}

	if m.section == AccessorySettingsSectionEnvironment {
		var cmd tea.Cmd
		m.env, cmd = m.env.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m AccessorySettings) View() string {
	titleLine := Styles.TitleRule(m.width, m.accessory.Settings.Name, "settings")
	formView := m.form.View()
	if m.section == AccessorySettingsSectionEnvironment {
		formView = m.env.View()
	}
	helpLine := Styles.CenteredLine(m.width, m.help.View())
	middleHeight := m.height - 2 - lipgloss.Height(helpLine)
	centeredContent := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, formView)
	return titleLine + "\n\n" + centeredContent + "\n" + helpLine
}
