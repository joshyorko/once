package ui

import (
	"context"
	"fmt"
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type installStage int

const (
	stagePreparing installStage = iota
	stageDownloading
	stageStarting
	stageVerifying
)

type installProgressMsg struct {
	stage      installStage
	percentage int
}

type InstallActivityDoneMsg struct {
	App *docker.Application
}

type InstallActivityFailedMsg struct {
	Err error
}

type AccessoryInstallActivityDoneMsg struct{}

type AccessoryInstallActivityFailedMsg struct {
	Err error
}

type deployActivityResult struct {
	value any
	err   error
}

type deployActivityConfig struct {
	run        func(context.Context, func(docker.DeployProgress)) (any, error)
	successMsg func(any) tea.Msg
	failureMsg func(error) tea.Msg
}

type DeployActivity struct {
	config        deployActivityConfig
	width, height int
	stage         installStage
	percentage    int
	progress      Progress
	progressChan  chan installProgressMsg
	doneChan      chan deployActivityResult
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewInstallActivity(ns *docker.Namespace, imageRef, hostname string, disableTLS bool) *DeployActivity {
	return NewDeployActivity(deployActivityConfig{
		run: func(ctx context.Context, progress func(docker.DeployProgress)) (any, error) {
			return runApplicationInstall(ctx, ns, imageRef, hostname, disableTLS, progress)
		},
		successMsg: func(value any) tea.Msg {
			return InstallActivityDoneMsg{App: value.(*docker.Application)}
		},
		failureMsg: func(err error) tea.Msg {
			return InstallActivityFailedMsg{Err: err}
		},
	})
}

func NewAccessoryInstallActivity(ns *docker.Namespace, settings docker.AccessorySettings) *DeployActivity {
	return NewDeployActivity(deployActivityConfig{
		run: func(ctx context.Context, progress func(docker.DeployProgress)) (any, error) {
			return nil, runAccessoryInstall(ctx, ns, settings, progress)
		},
		successMsg: func(any) tea.Msg {
			return AccessoryInstallActivityDoneMsg{}
		},
		failureMsg: func(err error) tea.Msg {
			return AccessoryInstallActivityFailedMsg{Err: err}
		},
	})
}

func NewDeployActivity(config deployActivityConfig) *DeployActivity {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeployActivity{
		config:       config,
		stage:        stagePreparing,
		progress:     NewProgress(0, Colors.Primary),
		progressChan: make(chan installProgressMsg, 10),
		doneChan:     make(chan deployActivityResult, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (m *DeployActivity) Init() tea.Cmd {
	return tea.Batch(m.progress.Init(), m.startInstall(), m.waitForProgress())
}

func (m *DeployActivity) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress = m.progress.SetWidth(min(m.width-4, 60))

	case installProgressMsg:
		m.stage = msg.stage
		m.percentage = msg.percentage
		switch msg.stage {
		case stageDownloading:
			m.progress = m.progress.SetPercent(msg.percentage)
		default:
			m.progress = m.progress.SetPercent(-1)
		}
		return m.waitForProgress()

	case deployActivityResult:
		if msg.err != nil {
			return func() tea.Msg { return m.config.failureMsg(msg.err) }
		}
		return func() tea.Msg { return m.config.successMsg(msg.value) }

	case ProgressTickMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return cmd
	}

	return nil
}

func (m *DeployActivity) View() string {
	var status string
	switch m.stage {
	case stagePreparing:
		status = "Preparing..."
	case stageDownloading:
		status = "Downloading..."
	case stageStarting:
		status = "Starting..."
	case stageVerifying:
		status = "Verifying..."
	}

	statusLine := Styles.CenteredLine(m.width, status)
	progressView := Styles.CenteredLine(m.width, m.progress.View())

	return lipgloss.JoinVertical(lipgloss.Left, statusLine, progressView)
}

func (m *DeployActivity) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

// Private

func (m *DeployActivity) startInstall() tea.Cmd {
	return func() tea.Msg {
		go m.runInstall(m.ctx)
		return nil
	}
}

func (m *DeployActivity) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		select {
		case progress, ok := <-m.progressChan:
			if ok {
				return progress
			}
		case done := <-m.doneChan:
			return done
		}
		return nil
	}
}

func (m *DeployActivity) runInstall(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.doneChan <- deployActivityResult{err: fmt.Errorf("install panicked: %v", r)}
		}
	}()

	m.progressChan <- installProgressMsg{stage: stagePreparing}

	value, err := m.config.run(ctx, m.reportProgress)
	m.doneChan <- deployActivityResult{value: value, err: err}
}

func (m *DeployActivity) reportProgress(p docker.DeployProgress) {
	switch p.Stage {
	case docker.DeployStageDownloading:
		m.progressChan <- installProgressMsg{stage: stageDownloading, percentage: p.Percentage}
	case docker.DeployStageStarting:
		m.progressChan <- installProgressMsg{stage: stageStarting, percentage: 100}
	case docker.DeployStageFinished:
		m.progressChan <- installProgressMsg{stage: stageVerifying, percentage: 100}
	}
}

func runApplicationInstall(
	ctx context.Context,
	ns *docker.Namespace,
	imageRef, hostname string,
	disableTLS bool,
	progress func(docker.DeployProgress),
) (*docker.Application, error) {
	if err := ns.Setup(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", docker.ErrSetupFailed, err)
	}

	appName, err := ns.UniqueName(docker.NameFromImageRef(imageRef))
	if err != nil {
		return nil, fmt.Errorf("generating app name: %w", err)
	}

	app := docker.NewApplication(ns, docker.ApplicationSettings{
		Name:       appName,
		Image:      imageRef,
		Host:       hostname,
		DisableTLS: disableTLS,
		AutoUpdate: true,
	})

	if err := app.Deploy(ctx, progress); err != nil {
		if cleanupErr := app.Remove(context.Background(), true); cleanupErr != nil {
			slog.Error("Failed to clean up after deploy failure", "app", appName, "error", cleanupErr)
		}
		return nil, fmt.Errorf("%w: %w", docker.ErrDeployFailed, err)
	}

	progress(docker.DeployProgress{Stage: docker.DeployStageFinished})

	if err := app.VerifyHTTP(ctx); err != nil {
		if cleanupErr := app.Remove(context.Background(), true); cleanupErr != nil {
			slog.Error("Failed to clean up after verification failure", "app", appName, "error", cleanupErr)
		}
		return nil, err
	}

	return app, nil
}

func runAccessoryInstall(
	ctx context.Context,
	ns *docker.Namespace,
	settings docker.AccessorySettings,
	progress func(docker.DeployProgress),
) error {
	accessory := docker.NewAccessory(ns, settings)
	if err := accessory.Deploy(ctx, progress); err != nil {
		if cleanupErr := accessory.Remove(context.Background(), true); cleanupErr != nil {
			slog.Error("Failed to clean up after accessory deploy failure", "accessory", settings.Name, "error", cleanupErr)
		}
		return err
	}

	return nil
}
