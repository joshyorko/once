package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	proxyImage = "basecamp/kamal-proxy:once-01"
	labelKey   = "once"
)

const (
	stateFileDir  = "/home/kamal-proxy/.config/kamal-proxy"
	stateFileName = "once-state.json"
	stateFilePath = stateFileDir + "/" + stateFileName
)

const (
	DefaultHTTPPort    = 80
	DefaultHTTPSPort   = 443
	DefaultMetricsPort = 1318
	deployTimeout      = "120s"
)

type ProxySettings struct {
	BindAddress string `json:"bindAddress,omitempty"`
	HTTPPort    int    `json:"httpPort"`
	HTTPSPort   int    `json:"httpsPort"`
	MetricsPort int    `json:"metricsPort"`
}

func UnmarshalProxySettings(s string) (ProxySettings, error) {
	var settings ProxySettings
	err := json.Unmarshal([]byte(s), &settings)
	return settings, err
}

func (s ProxySettings) Marshal() string {
	b, _ := json.Marshal(s)
	return string(b)
}

type DeployOptions struct {
	ServiceName     string
	Target          string
	Host            string
	TLS             bool
	Force           bool
	HealthCheckHost string
	HealthCheckPath string
	HealthCheckPort int
}

type Proxy struct {
	namespace *Namespace
	Settings  *ProxySettings
}

func NewProxy(ns *Namespace) *Proxy {
	return &Proxy{namespace: ns}
}

func (p *Proxy) Boot(ctx context.Context, settings ProxySettings) error {
	settings = normalizeProxySettings(settings)

	if settings.HTTPPort == 0 {
		settings.HTTPPort = DefaultHTTPPort
	}
	if settings.HTTPSPort == 0 {
		settings.HTTPSPort = DefaultHTTPSPort
	}
	if settings.MetricsPort == 0 {
		settings.MetricsPort = DefaultMetricsPort
	}

	info, err := p.namespace.client.ContainerInspect(ctx, p.containerName())
	if err == nil {
		return p.ensureRunning(ctx, info)
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("inspecting proxy container: %w", err)
	}

	reader, err := p.namespace.client.ImagePull(ctx, proxyImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling proxy image: %w", err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)

	name := p.containerName()
	metricsPortTCP := nat.Port(fmt.Sprintf("%d/tcp", settings.MetricsPort))

	resp, err := p.namespace.client.ContainerCreate(ctx,
		&container.Config{
			Image: proxyImage,
			Cmd:   []string{"kamal-proxy", "run", "--metrics-port", fmt.Sprintf("%d", settings.MetricsPort)},
			Labels: map[string]string{
				labelKey: settings.Marshal(),
			},
			ExposedPorts: nat.PortSet{
				"80/tcp":       struct{}{},
				"443/tcp":      struct{}{},
				metricsPortTCP: struct{}{},
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"80/tcp":       []nat.PortBinding{{HostIP: settings.BindAddress, HostPort: fmt.Sprintf("%d", settings.HTTPPort)}},
				"443/tcp":      []nat.PortBinding{{HostIP: settings.BindAddress, HostPort: fmt.Sprintf("%d", settings.HTTPSPort)}},
				metricsPortTCP: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%d", settings.MetricsPort)}},
			},
			RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyAlways},
			LogConfig:     ContainerLogConfig(),
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: name,
					Target: "/home/kamal-proxy/.config/kamal-proxy",
				},
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				p.namespace.name: {
					Aliases: []string{"kamal-proxy"},
				},
			},
		},
		nil,
		name,
	)
	if err != nil {
		return fmt.Errorf("creating proxy container: %w", err)
	}

	if err := p.namespace.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		if isPortConflict(err) {
			slog.Error("Port conflict starting proxy", "error", err)
			return ErrProxyPortInUse
		}
		return fmt.Errorf("starting proxy container: %w", err)
	}

	p.Settings = &settings
	return nil
}

func (p *Proxy) ApplySettings(ctx context.Context, settings ProxySettings) error {
	settings = normalizeProxySettings(settings)
	if settings.HTTPPort == 0 {
		settings.HTTPPort = DefaultHTTPPort
	}
	if settings.HTTPSPort == 0 {
		settings.HTTPSPort = DefaultHTTPSPort
	}
	if settings.MetricsPort == 0 {
		settings.MetricsPort = DefaultMetricsPort
	}

	info, err := p.namespace.client.ContainerInspect(ctx, p.containerName())
	if err != nil {
		if errdefs.IsNotFound(err) {
			return p.Boot(ctx, settings)
		}
		return fmt.Errorf("inspecting proxy container: %w", err)
	}

	current := p.Settings
	if current == nil {
		current = &ProxySettings{}
	}
	if proxySettingsEqual(*current, settings) {
		return p.ensureRunning(ctx, info)
	}

	if err := p.namespace.client.ContainerRemove(ctx, info.ID, container.RemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("removing proxy container: %w", err)
	}

	if err := p.Boot(ctx, settings); err != nil {
		return err
	}

	if err := p.reRegisterRoutes(ctx); err != nil {
		return fmt.Errorf("re-registering routes: %w", err)
	}

	return nil
}

func (p *Proxy) Destroy(ctx context.Context) error {
	containerName := p.containerName()

	if err := p.namespace.client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("removing proxy: %w", err)
		}
	}

	if err := p.namespace.client.VolumeRemove(ctx, containerName, true); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("removing proxy volume: %w", err)
		}
	}

	p.Settings = nil
	return nil
}

func (p *Proxy) Exec(ctx context.Context, cmd []string) error {
	output, err := p.ExecOutput(ctx, cmd)
	if err != nil && output != "" {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(output))
	}
	return err
}

func (p *Proxy) Remove(ctx context.Context, serviceName string) error {
	return p.Exec(ctx, []string{"kamal-proxy", "remove", serviceName})
}

func (p *Proxy) Deploy(ctx context.Context, opts DeployOptions) error {
	return p.Exec(ctx, p.deployArgs(opts))
}

func (p *Proxy) containerName() string {
	return p.namespace.name + "-proxy"
}

// Private

func (p *Proxy) ensureRunning(ctx context.Context, info container.InspectResponse) error {
	if !info.State.Running {
		if err := p.namespace.client.ContainerStart(ctx, info.ID, container.StartOptions{}); err != nil {
			if isPortConflict(err) {
				slog.Error("Port conflict starting proxy", "error", err)
				return ErrProxyPortInUse
			}
			return fmt.Errorf("starting proxy container: %w", err)
		}
	}

	label := info.Config.Labels[labelKey]
	if label != "" {
		settings, err := UnmarshalProxySettings(label)
		if err != nil {
			return fmt.Errorf("unmarshalling proxy settings: %w", err)
		}
		settings = normalizeProxySettings(settings)
		p.Settings = &settings
	}

	return nil
}

func (p *Proxy) deployArgs(opts DeployOptions) []string {
	args := []string{"kamal-proxy", "deploy", opts.ServiceName, "--target", opts.Target, "--deploy-timeout", deployTimeout}

	if opts.Host != "" {
		args = append(args, "--host", opts.Host)
	}

	if opts.TLS {
		args = append(args, "--tls")
	}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.HealthCheckHost != "" {
		args = append(args, "--health-check-host", opts.HealthCheckHost)
	}
	if opts.HealthCheckPath != "" {
		args = append(args, "--health-check-path", opts.HealthCheckPath)
	}
	if opts.HealthCheckPort != 0 {
		args = append(args, "--health-check-port", fmt.Sprintf("%d", opts.HealthCheckPort))
	}

	return args
}

func (p *Proxy) LoadState(ctx context.Context) (*State, error) {
	containerName := p.containerName()

	reader, _, err := p.namespace.client.CopyFromContainer(ctx, containerName, stateFilePath)
	if err != nil {
		// Return empty state when the file doesn't exist yet (first boot)
		if errdefs.IsNotFound(err) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("copying state from container: %w", err)
	}
	defer reader.Close()

	tr := tar.NewReader(reader)
	if _, err := tr.Next(); err != nil {
		return nil, fmt.Errorf("reading state tar: %w", err)
	}

	var state State
	if err := json.NewDecoder(tr).Decode(&state); err != nil {
		return nil, fmt.Errorf("decoding state: %w", err)
	}

	return &state, nil
}

func (p *Proxy) SaveState(ctx context.Context, state *State) error {
	containerName := p.containerName()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	header := &tar.Header{
		Name: stateFileName,
		Mode: 0o644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("writing tar header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing tar data: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("closing tar writer: %w", err)
	}

	if err := p.namespace.client.CopyToContainer(ctx, containerName, stateFileDir, &buf, container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("copying state to container: %w", err)
	}

	return nil
}

func (p *Proxy) ExecOutput(ctx context.Context, cmd []string) (string, error) {
	result, err := execInContainer(ctx, p.namespace.client, p.containerName(), cmd)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return result.Stdout + result.Stderr, fmt.Errorf("exec failed with exit code %d", result.ExitCode)
	}
	return result.Stdout, nil
}

func (p *Proxy) reRegisterRoutes(ctx context.Context) error {
	var errs []error
	for _, app := range p.namespace.Applications() {
		if !app.Running || app.Settings.Host == "" {
			continue
		}
		target, err := serviceTarget(ctx, p.namespace.client, app.ContainerName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := p.Deploy(ctx, DeployOptions{
			ServiceName: app.Settings.Name,
			Target:      target,
			Host:        app.Settings.Host,
			TLS:         app.Settings.TLSEnabled(),
		}); err != nil {
			errs = append(errs, err)
		}
	}
	for _, accessory := range p.namespace.Accessories() {
		if !accessory.Running || !accessory.Settings.Proxy.Enabled || accessory.Settings.Proxy.Host == "" {
			continue
		}
		target, err := serviceTarget(ctx, p.namespace.client, accessory.ContainerName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if port := accessory.Settings.Proxy.TargetPort; port != 0 && port != 80 {
			target += fmt.Sprintf(":%d", port)
		}
		if err := p.Deploy(ctx, DeployOptions{
			ServiceName:     accessory.Settings.Name,
			Target:          target,
			Host:            accessory.Settings.Proxy.Host,
			TLS:             !accessory.Settings.Proxy.DisableTLS,
			Force:           accessory.Settings.HealthCheck.Type != AccessoryHealthCheckHTTP,
			HealthCheckPath: accessoryHTTPHealthPath(accessory.Settings),
			HealthCheckPort: accessoryHTTPHealthPort(accessory.Settings),
		}); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Private

func normalizeProxySettings(settings ProxySettings) ProxySettings {
	if settings.BindAddress == "" {
		settings.BindAddress = "0.0.0.0"
	}
	return settings
}

func proxySettingsEqual(a, b ProxySettings) bool {
	a = normalizeProxySettings(a)
	b = normalizeProxySettings(b)
	return a == b
}

func serviceTarget(ctx context.Context, c *client.Client, containerNameFn func(context.Context) (string, error)) (string, error) {
	name, err := containerNameFn(ctx)
	if err != nil {
		return "", err
	}
	info, err := c.ContainerInspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("inspecting container %s: %w", name, err)
	}
	return info.ID[:12], nil
}

func accessoryHTTPHealthPath(settings AccessorySettings) string {
	if settings.HealthCheck.Type != AccessoryHealthCheckHTTP {
		return ""
	}
	if settings.HealthCheck.Path != "" {
		return settings.HealthCheck.Path
	}
	return "/"
}

func accessoryHTTPHealthPort(settings AccessorySettings) int {
	if settings.HealthCheck.Type != AccessoryHealthCheckHTTP {
		return 0
	}
	if settings.HealthCheck.Port != 0 {
		return settings.HealthCheck.Port
	}
	return settings.Proxy.TargetPort
}
