package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type AccessoryPanel struct {
	accessory docker.Accessory
}

func NewAccessoryPanel(accessory *docker.Accessory) AccessoryPanel {
	return AccessoryPanel{accessory: *accessory}
}

func (p AccessoryPanel) View(selected bool) string {
	lines := []string{
		p.headerLine(),
		p.detailLine("scope", string(p.accessory.Settings.Scope)),
	}
	if owner := p.accessory.Settings.OwnerApp; owner != "" {
		lines = append(lines, p.detailLine("owner", owner))
	}
	if image := p.accessory.Settings.Image; image != "" {
		lines = append(lines, p.detailLine("image", image))
	}
	if host := p.accessory.Settings.Proxy.Host; host != "" {
		lines = append(lines, p.detailLine("proxy", host))
	}
	if len(p.accessory.Settings.Ports) > 0 {
		lines = append(lines, p.detailLine("ports", formatAccessoryPorts(p.accessory.Settings.Ports)))
	}

	content := strings.Join(lines, "\n")
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Colors.Border).Padding(0, 1)
	if selected {
		style = style.Background(Colors.BackgroundTint)
	}
	return style.Render(content)
}

func (p AccessoryPanel) headerLine() string {
	status := "stopped"
	if p.accessory.Running {
		status = "running"
	}
	health := "no healthcheck"
	switch p.accessory.Settings.HealthCheck.Type {
	case docker.AccessoryHealthCheckHTTP, docker.AccessoryHealthCheckExec:
		health = string(p.accessory.Settings.HealthCheck.Type)
	}
	return fmt.Sprintf("%s [%s] %s", p.accessory.Settings.Name, status, health)
}

func (p AccessoryPanel) detailLine(label, value string) string {
	return fmt.Sprintf("%s: %s", label, value)
}

func formatAccessoryPorts(ports []docker.AccessoryPortBinding) string {
	parts := make([]string, 0, len(ports))
	for _, port := range ports {
		binding := ""
		if port.HostIP != "" {
			binding += port.HostIP + ":"
		}
		if port.HostPort != 0 {
			binding += fmt.Sprintf("%d:", port.HostPort)
		}
		binding += fmt.Sprintf("%d", port.ContainerPort)
		parts = append(parts, binding)
	}
	return strings.Join(parts, ", ")
}
