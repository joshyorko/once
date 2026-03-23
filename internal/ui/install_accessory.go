package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/accessorytemplates"
	"github.com/basecamp/once/internal/docker"
)

var installAccessoryKeys = struct {
	Help key.Binding
	Back key.Binding
}{
	Help: WithHelp(NewKeyBinding("f1"), "F1", "help"),
	Back: WithHelp(NewKeyBinding("esc"), "esc", "back"),
}

type installAccessoryState int

const (
	installAccessoryStateTemplates installAccessoryState = iota
	installAccessoryStateForm
	installAccessoryStateEnvironment
	installAccessoryStateActivity
)

type InstallAccessorySubmitMsg struct{}

type InstallAccessory struct {
	namespace     *docker.Namespace
	width, height int
	help          Help
	state         installAccessoryState
	menu          Menu
	form          Form
	env           Component
	progress      Progress
	err           error
	template      accessorytemplates.Template
	settings      docker.AccessorySettings
	presetApp     string
}

type installAccessoryFinishedMsg struct {
	err error
}

type installAccessoryFormErrorMsg struct {
	err error
}

func NewInstallAccessory(ns *docker.Namespace, preset *docker.Accessory, _ string) InstallAccessory {
	h := NewHelp()
	h.SetBindings([]key.Binding{installAccessoryKeys.Back})
	m := InstallAccessory{
		namespace: ns,
		help:      h,
		state:     installAccessoryStateTemplates,
	}
	if preset != nil {
		m.presetApp = preset.Settings.OwnerApp
	}
	m.menu = NewMenu(
		MenuItem{Label: "Cloudflare Tunnel", Key: 0, Shortcut: WithHelp(NewKeyBinding("1"), "1", "")},
		MenuItem{Label: "MinIO", Key: 1, Shortcut: WithHelp(NewKeyBinding("2"), "2", "")},
		MenuItem{Label: "Prometheus", Key: 2, Shortcut: WithHelp(NewKeyBinding("3"), "3", "")},
		MenuItem{Label: "Alertmanager", Key: 3, Shortcut: WithHelp(NewKeyBinding("4"), "4", "")},
		MenuItem{Label: "Custom", Key: 4, Shortcut: WithHelp(NewKeyBinding("5"), "5", "")},
	)
	return m
}

func (m InstallAccessory) Init() tea.Cmd {
	if m.state == installAccessoryStateEnvironment && m.env != nil {
		return m.env.Init()
	}
	if m.state == installAccessoryStateForm {
		return m.form.Init()
	}
	return nil
}

func (m InstallAccessory) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(m.width)
		if m.state == installAccessoryStateForm {
			m.form, _ = m.form.Update(msg)
		}
		if m.state == installAccessoryStateEnvironment {
			m.env, _ = m.env.Update(msg)
		}
		if m.state == installAccessoryStateActivity {
			m.progress = m.progress.SetWidth(m.width)
		}
		return m, nil
	case MouseEvent:
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case tea.KeyPressMsg:
		if key.Matches(msg, installAccessoryKeys.Back) {
			return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
		}
	case MenuSelectMsg:
		var cmd tea.Cmd
		m, cmd = m.handleTemplateSelect(AccessoryTemplateIndex(msg.Key))
		return m, cmd
	case InstallAccessorySubmitMsg:
		if len(m.settings.EnvVars) > 0 || len(m.template.RequiredEnv) > 0 {
			m.state = installAccessoryStateEnvironment
			m.env = NewAccessoryEnvironment(m.settings.EnvVars, m.template.RequiredEnv)
			return m, m.env.Init()
		}
		m, cmd := m.deployAccessory()
		return m, cmd
	case AccessoryEnvironmentSubmitMsg:
		m.settings.EnvVars = msg.EnvVars
		m, cmd := m.deployAccessory()
		return m, cmd
	case AccessoryEnvironmentCancelMsg:
		m.state = installAccessoryStateForm
		return m, nil
	case installAccessoryFinishedMsg:
		if msg.err != nil {
			if len(m.template.RequiredEnv) > 0 || len(m.settings.EnvVars) > 0 {
				m.state = installAccessoryStateEnvironment
			} else {
				m.state = installAccessoryStateForm
			}
			m.err = msg.err
			return m, nil
		}
		return m, func() tea.Msg { return NavigateToDashboardMsg{AllowEmpty: true} }
	case installAccessoryFormErrorMsg:
		m.err = msg.err
		m.state = installAccessoryStateForm
		return m, nil
	}

	if m.state == installAccessoryStateForm {
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	}
	if m.state == installAccessoryStateEnvironment {
		var cmd tea.Cmd
		m.env, cmd = m.env.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m InstallAccessory) View() string {
	var content string
	switch m.state {
	case installAccessoryStateForm:
		content = m.form.View()
	case installAccessoryStateEnvironment:
		content = m.env.View()
	case installAccessoryStateActivity:
		content = m.progress.View()
	default:
		content = m.menu.View()
	}
	helpLine := Styles.CenteredLine(m.width, m.help.View())
	titleLine := Styles.TitleRule(m.width, "install accessory")
	if m.err != nil {
		errorLine := lipgloss.NewStyle().Foreground(Colors.Error).Width(m.width).Align(lipgloss.Center).Render(docker.ErrorMessage(m.err))
		content = lipgloss.JoinVertical(lipgloss.Center, errorLine, "", content)
	}
	return titleLine + "\n\n" + lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center, content) + "\n" + helpLine
}

// Private

type AccessoryTemplateIndex int

func (m InstallAccessory) handleTemplateSelect(index AccessoryTemplateIndex) (InstallAccessory, tea.Cmd) {
	var template accessorytemplates.Template
	switch index {
	case 0:
		template, _ = accessorytemplates.ByAlias("cloudflared")
	case 1:
		template, _ = accessorytemplates.ByAlias("minio")
	case 2:
		template, _ = accessorytemplates.ByAlias("prometheus")
	case 3:
		template, _ = accessorytemplates.ByAlias("alertmanager")
	default:
		template = accessorytemplates.Template{Settings: docker.AccessorySettings{Scope: docker.AccessoryScopeShared}}
	}
	m.template = template
	m.settings = template.Settings
	if m.presetApp != "" {
		m.settings.Scope = docker.AccessoryScopePerApp
		m.settings.OwnerApp = m.presetApp
		m.settings.InheritAppRuntime = true
	}

	nameField := NewTextField("accessory")
	imageField := NewTextField("user/repo:tag")
	imageField.SetValue(m.settings.Image)
	appField := NewTextField("owner-app")
	appField.SetValue(m.settings.OwnerApp)
	commandField := NewTextField("tunnel run")
	commandField.SetValue(accessoryCommandField(m.settings.Command))
	volumeField := NewTextField("data:/data:ro")
	volumeField.SetValue(accessoryMountField(m.settings.Mounts, docker.AccessoryMountVolume))
	bindField := NewTextField("/host/path:/container/path:ro")
	bindField.SetValue(accessoryMountField(m.settings.Mounts, docker.AccessoryMountBind))
	proxyHost := NewTextField("app.example.com")
	proxyHost.SetValue(m.settings.Proxy.Host)
	proxyPort := NewTextField("9001")
	proxyPort.SetDigitsOnly(true)
	if m.settings.Proxy.TargetPort != 0 {
		proxyPort.SetValue(strconv.Itoa(m.settings.Proxy.TargetPort))
	}
	publishField := NewTextField("127.0.0.1:9001:9000")
	publishField.SetValue(accessoryPortField(m.settings.Ports))
	healthType := NewTextField("none|http|exec")
	healthType.SetValue(string(m.settings.HealthCheck.Type))
	healthHTTPPort := NewTextField("80")
	healthHTTPPort.SetDigitsOnly(true)
	if m.settings.HealthCheck.Port != 0 {
		healthHTTPPort.SetValue(strconv.Itoa(m.settings.HealthCheck.Port))
	}
	healthHTTPPath := NewTextField("/up")
	healthHTTPPath.SetValue(m.settings.HealthCheck.Path)
	healthCommand := NewTextField("curl -fsS http://localhost/")
	healthCommand.SetValue(accessoryCommandField(m.settings.HealthCheck.Command))
	restartField := NewTextField("always")
	restartField.SetValue(m.settings.RestartPolicy)

	f := NewForm("Deploy",
		FormItem{Label: "Name", Field: nameField, Required: true},
		FormItem{Label: "Image", Field: imageField, Required: index == 4},
		FormItem{Label: "Owner app", Field: appField},
		FormItem{Label: "Command", Field: commandField},
		FormItem{Label: "Volumes", Field: volumeField},
		FormItem{Label: "Bind mounts", Field: bindField},
		FormItem{Label: "Published ports", Field: publishField},
		FormItem{Label: "Proxy host", Field: proxyHost},
		FormItem{Label: "Proxy port", Field: proxyPort},
		FormItem{Label: "Health type", Field: healthType},
		FormItem{Label: "Health port", Field: healthHTTPPort},
		FormItem{Label: "Health path", Field: healthHTTPPath},
		FormItem{Label: "Health command", Field: healthCommand},
		FormItem{Label: "Restart", Field: restartField},
	)
	f.OnSubmit(func(f *Form) tea.Cmd {
		m.settings.Name = f.TextField(0).Value()
		m.settings.Image = f.TextField(1).Value()
		m.settings.OwnerApp = f.TextField(2).Value()
		if m.settings.OwnerApp != "" {
			m.settings.Scope = docker.AccessoryScopePerApp
			m.settings.InheritAppRuntime = true
		}
		m.settings.Command = parseAccessoryCommandField(f.TextField(3).Value())

		volumes, err := parseAccessoryMountField(f.TextField(4).Value(), docker.AccessoryMountVolume)
		if err != nil {
			return func() tea.Msg { return installAccessoryFormErrorMsg{err: fmt.Errorf("parsing volume mounts: %w", err)} }
		}
		binds, err := parseAccessoryMountField(f.TextField(5).Value(), docker.AccessoryMountBind)
		if err != nil {
			return func() tea.Msg { return installAccessoryFormErrorMsg{err: fmt.Errorf("parsing bind mounts: %w", err)} }
		}
		m.settings.Mounts = append(volumes, binds...)

		ports, err := parseAccessoryPortField(f.TextField(6).Value())
		if err != nil {
			return func() tea.Msg {
				return installAccessoryFormErrorMsg{err: fmt.Errorf("parsing published ports: %w", err)}
			}
		}
		m.settings.Ports = ports

		m.settings.Proxy.Host = f.TextField(7).Value()
		m.settings.Proxy.TargetPort, _ = strconv.Atoi(f.TextField(8).Value())
		m.settings.Proxy.Enabled = m.settings.Proxy.Host != ""
		m.settings.HealthCheck.Type = docker.AccessoryHealthCheckType(strings.TrimSpace(f.TextField(9).Value()))
		m.settings.HealthCheck.Port, _ = strconv.Atoi(f.TextField(10).Value())
		m.settings.HealthCheck.Path = f.TextField(11).Value()
		m.settings.HealthCheck.Command = parseAccessoryCommandField(f.TextField(12).Value())
		m.settings.RestartPolicy = strings.TrimSpace(f.TextField(13).Value())
		return func() tea.Msg { return InstallAccessorySubmitMsg{} }
	})
	m.form = f
	m.state = installAccessoryStateForm
	return m, m.form.Init()
}

func (m InstallAccessory) deployAccessory() (InstallAccessory, tea.Cmd) {
	m.state = installAccessoryStateActivity
	m.progress = NewProgress(m.width, Colors.Border)
	return m, tea.Batch(m.progress.Init(), func() tea.Msg {
		accessory := docker.NewAccessory(m.namespace, m.settings)
		err := accessory.Deploy(context.Background(), nil)
		return installAccessoryFinishedMsg{err: err}
	})
}
