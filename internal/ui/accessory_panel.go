package ui

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type AccessoryPanel struct {
	accessory     docker.Accessory
	dockerScraper *docker.Scraper
}

func NewAccessoryPanel(accessory *docker.Accessory, dockerScraper *docker.Scraper) AccessoryPanel {
	return AccessoryPanel{
		accessory:     *accessory,
		dockerScraper: dockerScraper,
	}
}

func (p AccessoryPanel) View(selected bool, toggling bool, width int, scales DashboardScales) string {
	innerWidth := max(width-3, 0) // indicator(1) + left padding(1) + right padding(1)
	detailed := p.accessory.Running

	var cards [2]MetricCard
	if p.accessory.Running {
		cards = p.buildMetricCards(scales)
	}

	displayName := p.accessory.DisplayName()
	name := Styles.Title.Render(displayName)
	imageName := docker.NameFromImageRef(p.accessoryImage())
	if imageName == "" {
		imageName = "accessory"
	}
	leftParts := []string{p.renderHealthBadge(cards), name}
	if imageName != "" && imageName != displayName {
		leftParts = append(leftParts, lipgloss.NewStyle().Foreground(Colors.Border).Render("("+imageName+")"))
	}
	left := strings.Join(leftParts, " ")
	right := renderAccessoryStateInfo(&p.accessory, toggling)
	gap := max(innerWidth-2-lipgloss.Width(left)-lipgloss.Width(right), 1)
	titleLine := " " + left + strings.Repeat(" ", gap) + right + " "

	lines := []string{titleLine}

	const minCardWidth = 8
	const cardCount = 3
	const cardGaps = cardCount - 1
	if detailed && innerWidth >= minCardWidth*cardCount+cardGaps {
		lines = append(lines, p.renderCards(innerWidth, cards))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	height := PanelHeight
	if !detailed {
		height = StoppedPanelHeight
	}

	bodyStyle := lipgloss.NewStyle().
		Width(width-1).
		Padding(0, 1).
		Height(height)

	var body string
	if selected {
		body = bodyStyle.Background(Colors.BackgroundTint).Render(content)
		body = WithBackground(Colors.BackgroundTint, body)
	} else {
		body = bodyStyle.Render(content)
	}

	indicator := p.renderIndicator(selected, height)
	topTrans := p.renderTopTransition(selected, width)
	bottomTrans := p.renderBottomTransition(selected, width)

	return topTrans + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, indicator, body) + "\n" + bottomTrans
}

func (p AccessoryPanel) Height() int {
	bodyHeight := PanelHeight
	if !p.accessory.Running {
		bodyHeight = StoppedPanelHeight
	}
	return bodyHeight + 2 // top + bottom transition lines
}

// Private

func (p AccessoryPanel) renderHealthBadge(cards [2]MetricCard) string {
	if !p.accessory.Running {
		return lipgloss.NewStyle().Foreground(Colors.Border).Render("●")
	}

	worst := healthNormal
	for _, c := range cards {
		worst = max(worst, c.Health())
	}

	return lipgloss.NewStyle().Foreground(worst.Color()).Render("●")
}

func (p AccessoryPanel) renderCards(innerWidth int, cards [2]MetricCard) string {
	gaps := 2
	summaryWidth := (innerWidth - gaps) / 3
	remaining := (innerWidth - gaps) - summaryWidth
	metricWidth := remaining / 2
	metricRem := remaining % 2

	cpuWidth := metricWidth
	memWidth := metricWidth
	if metricRem > 0 {
		cpuWidth++
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		p.renderSummaryCard(summaryWidth),
		" ",
		cards[0].View(cpuWidth),
		" ",
		cards[1].View(memWidth),
	)
}

func (p AccessoryPanel) renderSummaryCard(width int) string {
	inner := width - 2
	left := boxSide()
	right := boxSide()

	contentLines := make([]string, 3)
	contentLines[0] = left + padOrTruncate(" "+p.summaryLine(), inner) + right
	contentLines[1] = left + padOrTruncate(" "+p.detailLine(), inner) + right
	contentLines[2] = left + padOrTruncate(" "+p.extraLine(), inner) + right

	return boxTop("Details", inner) + "\n" + strings.Join(contentLines, "\n") + "\n" + boxBottom(inner)
}

func (p AccessoryPanel) summaryLine() string {
	if host := p.accessory.Settings.Proxy.Host; host != "" {
		return host
	}
	if ports := formatAccessoryPorts(p.accessory.Settings.Ports); ports != "" {
		return ports
	}
	return fmt.Sprintf("%s accessory", p.accessory.Settings.Scope)
}

func (p AccessoryPanel) detailLine() string {
	switch p.accessory.Settings.Scope {
	case docker.AccessoryScopePerApp:
		if p.accessory.Settings.OwnerApp != "" {
			return "owner: " + p.accessory.Settings.OwnerApp
		}
		return "per-app runtime"
	case docker.AccessoryScopeShared:
		return "shared runtime"
	default:
		return string(p.accessory.Settings.Scope)
	}
}

func (p AccessoryPanel) extraLine() string {
	switch p.accessory.Settings.HealthCheck.Type {
	case docker.AccessoryHealthCheckHTTP, docker.AccessoryHealthCheckExec:
		return "health: " + string(p.accessory.Settings.HealthCheck.Type)
	default:
		return "health: none"
	}
}

func (p AccessoryPanel) buildMetricCards(scales DashboardScales) [2]MetricCard {
	cpuData, memData := p.fetchDockerData()

	cpuScale := scales.CPU
	cpuLimit := ""
	if c := p.accessory.Settings.Resources.CPUs; c > 0 {
		cpuScale = ChartScale{max: float64(c) * 100}
		cpuLimit = UnitPercent.Format(float64(c) * 100)
	}

	memScale := scales.Memory
	memLimit := ""
	if mb := p.accessory.Settings.Resources.MemoryMB; mb > 0 {
		memScale = ChartScale{max: float64(mb) * 1024 * 1024}
		memLimit = UnitBytes.Format(float64(mb) * 1024 * 1024)
	}

	return [2]MetricCard{
		NewMetricCard("CPU", cpuData, cpuScale, UnitPercent, cpuLimit, defaultWarningPct, defaultErrorPct),
		NewMetricCard("Memory", memData, memScale, UnitBytes, memLimit, defaultWarningPct, defaultErrorPct),
	}
}

func (p AccessoryPanel) fetchDockerData() (cpu, memory []float64) {
	if p.dockerScraper == nil {
		return nil, nil
	}

	samples := p.dockerScraper.Fetch(p.accessory.StatsName(), containerStatsBuffer)
	cpu = make([]float64, len(samples))
	memory = make([]float64, len(samples))
	for i, s := range samples {
		cpu[i] = s.CPUPercent
		memory[i] = float64(s.MemoryBytes)
	}
	slices.Reverse(cpu)
	slices.Reverse(memory)
	return
}

func (p AccessoryPanel) accessoryImage() string {
	if p.accessory.Settings.Image != "" {
		return p.accessory.Settings.Image
	}
	if p.accessory.Settings.OwnerApp != "" {
		return p.accessory.Settings.OwnerApp
	}
	return ""
}

func (p AccessoryPanel) renderTopTransition(selected bool, width int) string {
	if !selected {
		return strings.Repeat(" ", width)
	}
	indicatorChar := lipgloss.NewStyle().Foreground(Colors.Focused).Render("▗")
	bodyChars := lipgloss.NewStyle().Foreground(Colors.BackgroundTint).Render(strings.Repeat("▄", width-1))
	return indicatorChar + bodyChars
}

func (p AccessoryPanel) renderBottomTransition(selected bool, width int) string {
	if !selected {
		return strings.Repeat(" ", width)
	}
	indicatorChar := lipgloss.NewStyle().Foreground(Colors.Focused).Render("▝")
	bodyChars := lipgloss.NewStyle().Foreground(Colors.BackgroundTint).Render(strings.Repeat("▀", width-1))
	return indicatorChar + bodyChars
}

func (p AccessoryPanel) renderIndicator(selected bool, height int) string {
	rows := make([]string, height)
	if selected {
		line := lipgloss.NewStyle().Foreground(Colors.Focused).Render("▐")
		for i := range rows {
			rows[i] = line
		}
	} else {
		for i := range rows {
			rows[i] = " "
		}
	}
	return strings.Join(rows, "\n")
}

// Helpers

func renderAccessoryStateInfo(accessory *docker.Accessory, toggling bool) string {
	var status string
	var statusColor color.Color
	if toggling && accessory.Running {
		status = "stopping..."
		statusColor = Colors.LightText
	} else if toggling {
		status = "starting..."
		statusColor = Colors.LightText
	} else if accessory.Running {
		status = "running"
		statusColor = Colors.Success
	} else {
		status = "stopped"
		statusColor = Colors.LightText
	}

	return lipgloss.NewStyle().Foreground(statusColor).Render(status)
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
