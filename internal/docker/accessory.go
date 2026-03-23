package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
)

const accessoryVerifyTimeout = 120 * time.Second

type Accessory struct {
	namespace    *Namespace
	Settings     AccessorySettings
	Running      bool
	RunningSince time.Time
}

func NewAccessory(ns *Namespace, settings AccessorySettings) *Accessory {
	return &Accessory{
		namespace: ns,
		Settings:  settings,
	}
}

func (a *Accessory) ContainerName(ctx context.Context) (string, error) {
	containers, err := a.namespace.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", err
	}

	var (
		foundName string
		foundTime int64
	)

	for _, c := range containers {
		if len(c.Names) == 0 {
			continue
		}
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			if a.namespace.containerAccessoryName(name) != a.Settings.Name {
				continue
			}
			if foundName == "" || c.Created > foundTime {
				foundName = name
				foundTime = c.Created
			}
		}
	}

	if foundName == "" {
		return "", fmt.Errorf("no container found for accessory %s", a.Settings.Name)
	}
	return foundName, nil
}

func (a *Accessory) Start(ctx context.Context) error {
	name, err := a.ContainerName(ctx)
	if err != nil {
		return err
	}

	if err := a.namespace.client.ContainerStart(ctx, name, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}

	return a.Verify(ctx)
}

func (a *Accessory) Stop(ctx context.Context) error {
	name, err := a.ContainerName(ctx)
	if err != nil {
		return err
	}

	return a.namespace.client.ContainerStop(ctx, name, container.StopOptions{})
}

func (a *Accessory) Deploy(ctx context.Context, progress DeployProgressCallback) error {
	_, err := a.reconcile(ctx, true, progress)
	a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
	return err
}

func (a *Accessory) Update(ctx context.Context, progress DeployProgressCallback) (bool, error) {
	changed, err := a.reconcile(ctx, false, progress)
	a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
	return changed, err
}

func (a *Accessory) Remove(ctx context.Context, removeData bool) error {
	var errs []error

	if a.Settings.Proxy.Enabled && a.Settings.Proxy.Host != "" {
		if err := a.namespace.Proxy().Remove(ctx, a.Settings.Name); err != nil {
			errs = append(errs, fmt.Errorf("removing from proxy: %w", err))
		}
	}

	if err := a.Destroy(ctx, removeData); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (a *Accessory) Destroy(ctx context.Context, destroyVolumes bool) error {
	containers, err := a.namespace.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	var errs []error
	for _, c := range containers {
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			if a.namespace.containerAccessoryName(name) == a.Settings.Name {
				if err := a.namespace.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					errs = append(errs, fmt.Errorf("removing container: %w", err))
				}
				break
			}
		}
	}

	if destroyVolumes {
		if err := a.removeOwnedVolumes(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (a *Accessory) Verify(ctx context.Context) error {
	name, err := a.ContainerName(ctx)
	if err != nil {
		return err
	}

	info, err := a.namespace.client.ContainerInspect(ctx, name)
	if err != nil {
		return fmt.Errorf("inspecting container: %w", err)
	}
	if info.State == nil || !info.State.Running {
		return fmt.Errorf("container %s is not running", name)
	}

	if a.Settings.HealthCheck.Type == AccessoryHealthCheckNone || a.Settings.HealthCheck.Type == "" {
		return nil
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, accessoryVerifyTimeout)
	defer cancel()

	for {
		info, err := a.namespace.client.ContainerInspect(deadlineCtx, name)
		if err != nil {
			return fmt.Errorf("inspecting container health: %w", err)
		}
		if info.State != nil && info.State.Health != nil && info.State.Health.Status == container.Healthy {
			a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryHealthCheck(a.Settings.Name, nil) })
			return nil
		}
		if deadlineCtx.Err() != nil {
			err := fmt.Errorf("container %s did not become healthy", name)
			a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryHealthCheck(a.Settings.Name, err) })
			return err
		}
		time.Sleep(time.Second)
	}
}

func (a *Accessory) NewLogStreamer(ctx context.Context, settings LogStreamerSettings) (*LogStreamer, error) {
	name, err := a.ContainerName(ctx)
	if err != nil {
		return nil, err
	}

	streamer := NewLogStreamer(a.namespace, settings)
	streamer.Start(ctx, name)
	return streamer, nil
}

// Private

func (a *Accessory) reconcile(ctx context.Context, force bool, progress DeployProgressCallback) (bool, error) {
	effective, runtime, err := a.effectiveRuntime(ctx)
	if err != nil {
		return false, err
	}

	currentName, currentTemplate, currentImageID, err := a.currentContainerState(ctx)
	if err != nil {
		return false, err
	}

	pulledImageChanged, err := a.pullImage(ctx, effective.Image, progress)
	if err != nil {
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
		return false, err
	}

	effectiveImageID, err := a.imageID(ctx, effective.Image)
	if err != nil {
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
		return false, err
	}

	if !force && currentName != "" && currentTemplate.Equal(a.Settings) && currentImageID == effectiveImageID && !pulledImageChanged {
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, nil) })
		return false, nil
	}

	createdName, err := a.deployWithRuntime(ctx, runtime, effective, progress)
	if err != nil {
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
		return false, err
	}

	if err := a.removeContainersExcept(ctx, createdName); err != nil {
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
		return true, err
	}

	if err := a.saveDeployResult(ctx, nil); err != nil {
		slog.Debug("saving accessory deploy result", "accessory", a.Settings.Name, "error", err)
	}

	return true, nil
}

func (a *Accessory) deployWithRuntime(ctx context.Context, runtime accessoryRuntime, effective AccessorySettings, progress DeployProgressCallback) (string, error) {
	if progress != nil {
		progress(DeployProgress{Stage: DeployStageStarting})
	}

	id, err := ContainerRandomID()
	if err != nil {
		return "", fmt.Errorf("generating container id: %w", err)
	}

	containerName := fmt.Sprintf("%s-accessory-%s-%s", a.namespace.name, a.Settings.Name, id)
	hostConfig := a.hostConfig(runtime, effective)

	resp, err := a.namespace.client.ContainerCreate(ctx,
		a.containerConfig(effective),
		hostConfig,
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				a.namespace.name: {},
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}

	if err := a.namespace.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = a.namespace.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("starting container: %w", err)
	}

	if effective.HealthCheck.Type != AccessoryHealthCheckNone && effective.HealthCheck.Type != "" {
		if err := a.waitForHealthy(ctx, resp.ID); err != nil {
			a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryHealthCheck(a.Settings.Name, err) })
			_ = a.namespace.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return "", err
		}
		a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryHealthCheck(a.Settings.Name, nil) })
	}

	shortContainerID := resp.ID[:12]
	if effective.Proxy.Enabled {
		if effective.Proxy.Host == "" {
			_ = a.namespace.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return "", fmt.Errorf("proxy host is required")
		}
		target := shortContainerID
		if port := effective.Proxy.TargetPort; port != 0 && port != 80 {
			target = fmt.Sprintf("%s:%d", shortContainerID, port)
		}
		if err := a.namespace.Proxy().Deploy(ctx, DeployOptions{
			ServiceName: a.Settings.Name,
			Target:      target,
			Host:        effective.Proxy.Host,
			TLS:         !effective.Proxy.DisableTLS,
		}); err != nil {
			_ = a.namespace.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return "", fmt.Errorf("registering with proxy: %w", err)
		}
	}

	if progress != nil {
		progress(DeployProgress{Stage: DeployStageFinished})
	}

	return containerName, nil
}

func (a *Accessory) removeContainersExcept(ctx context.Context, keep string) error {
	containers, err := a.namespace.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	var errs []error
	for _, c := range containers {
		if len(c.Names) == 0 {
			continue
		}
		name := strings.TrimPrefix(c.Names[0], "/")
		if a.namespace.containerAccessoryName(name) == a.Settings.Name && name != keep {
			if err := a.namespace.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (a *Accessory) hostConfig(runtime accessoryRuntime, effective AccessorySettings) *container.HostConfig {
	restartPolicy := container.RestartPolicyMode(effective.RestartPolicy)
	if restartPolicy == "" {
		restartPolicy = container.RestartPolicyAlways
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: restartPolicy},
		LogConfig:     ContainerLogConfig(),
		Mounts:        runtime.mounts,
	}
	hostConfig.Resources = container.Resources{
		Memory:   int64(effective.Resources.MemoryMB) * 1024 * 1024,
		NanoCPUs: int64(effective.Resources.CPUs) * 1e9,
	}

	var portBindings nat.PortMap
	var exposedPorts nat.PortSet
	for _, port := range effective.Ports {
		containerPort := nat.Port(fmt.Sprintf("%d/tcp", port.ContainerPort))
		exposedPorts[containerPort] = struct{}{}

		if port.HostPort != 0 {
			binding := nat.PortBinding{HostPort: fmt.Sprintf("%d", port.HostPort)}
			if port.HostIP != "" {
				binding.HostIP = port.HostIP
			}
			portBindings[containerPort] = append(portBindings[containerPort], binding)
		}
	}
	hostConfig.PortBindings = portBindings
	hostConfig.AutoRemove = false
	hostConfig.Mounts = runtime.mounts

	if len(exposedPorts) > 0 {
		hostConfig.ExtraHosts = nil
	}

	return hostConfig
}

func (a *Accessory) containerConfig(effective AccessorySettings) *container.Config {
	labels := make(map[string]string, len(effective.Labels)+1)
	for k, v := range effective.Labels {
		labels[k] = v
	}
	labels[labelKey] = a.Settings.Marshal()

	var healthcheck *container.HealthConfig
	if cfg := healthcheckConfig(effective); cfg != nil {
		healthcheck = cfg
	}

	return &container.Config{
		Image:        effective.Image,
		Cmd:          effective.Command,
		Labels:       labels,
		Env:          accessoryEnv(effective.EnvVars),
		ExposedPorts: accessoryExposedPorts(effective.Ports),
		Healthcheck:  healthcheck,
	}
}

func (a *Accessory) effectiveRuntime(ctx context.Context) (AccessorySettings, accessoryRuntime, error) {
	settings := a.Settings
	runtime := accessoryRuntime{}

	if settings.Scope == AccessoryScopePerApp && settings.InheritAppRuntime {
		app := a.namespace.Application(settings.OwnerApp)
		if app == nil {
			return AccessorySettings{}, runtime, fmt.Errorf("owner app %q not found", settings.OwnerApp)
		}

		vol, err := app.Volume(ctx)
		if err != nil {
			return AccessorySettings{}, runtime, fmt.Errorf("getting owner app volume: %w", err)
		}

		if settings.Image == "" {
			settings.Image = app.Settings.Image
		}
		if settings.Resources == (ContainerResources{}) {
			settings.Resources = app.Settings.Resources
		}
		env := app.Settings.BuildEnv(vol.Settings)
		settings.EnvVars = mergeEnvMaps(env, settings.EnvVars)

		for _, target := range AppVolumeMountTargets {
			runtime.mounts = append(runtime.mounts, mount.Mount{
				Type:   mount.TypeVolume,
				Source: vol.Name(),
				Target: target,
			})
		}
	}

	for _, mnt := range settings.Mounts {
		switch mnt.Type {
		case AccessoryMountVolume:
			volumeName := a.namespace.accessoryVolumeName(a.Settings.Name, mnt.Name)
			runtime.mounts = append(runtime.mounts, mount.Mount{
				Type:     mount.TypeVolume,
				Source:   volumeName,
				Target:   mnt.Target,
				ReadOnly: mnt.ReadOnly,
			})
		case AccessoryMountBind:
			runtime.mounts = append(runtime.mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   mnt.Source,
				Target:   mnt.Target,
				ReadOnly: mnt.ReadOnly,
			})
		}
	}

	return settings, runtime, nil
}

func (a *Accessory) currentContainerState(ctx context.Context) (string, AccessorySettings, string, error) {
	name, err := a.ContainerName(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "no container found") {
			return "", AccessorySettings{}, "", nil
		}
		return "", AccessorySettings{}, "", err
	}

	info, err := a.namespace.client.ContainerInspect(ctx, name)
	if err != nil {
		return "", AccessorySettings{}, "", fmt.Errorf("inspecting container: %w", err)
	}

	var stored AccessorySettings
	if label := info.Config.Labels[labelKey]; label != "" {
		stored, err = UnmarshalAccessorySettings(label)
		if err != nil {
			return "", AccessorySettings{}, "", fmt.Errorf("parsing accessory settings: %w", err)
		}
	}

	imageID := info.Image
	return name, stored, imageID, nil
}

func (a *Accessory) pullImage(ctx context.Context, imageRef string, progress DeployProgressCallback) (bool, error) {
	reader, err := a.namespace.client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPullFailed, err)
	}
	defer reader.Close()

	if progress != nil {
		tracker := newPullProgressTracker(progress)
		if err := tracker.Track(reader); err != nil {
			return false, fmt.Errorf("%w: %w", ErrPullFailed, err)
		}
	} else {
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return false, fmt.Errorf("%w: %w", ErrPullFailed, err)
		}
	}

	pulledInspect, err := a.namespace.client.ImageInspect(ctx, imageRef)
	if err != nil {
		return false, fmt.Errorf("%w: inspecting image after pull: %w", ErrPullFailed, err)
	}

	return pulledInspect.ID != a.currentImageID(ctx), nil
}

func (a *Accessory) currentImageID(ctx context.Context) string {
	name, err := a.ContainerName(ctx)
	if err != nil {
		return ""
	}
	info, err := a.namespace.client.ContainerInspect(ctx, name)
	if err != nil {
		return ""
	}
	return info.Image
}

func (a *Accessory) imageID(ctx context.Context, imageRef string) (string, error) {
	inspect, err := a.namespace.client.ImageInspect(ctx, imageRef)
	if err != nil {
		return "", err
	}
	return inspect.ID, nil
}

func (a *Accessory) waitForHealthy(ctx context.Context, containerID string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, accessoryVerifyTimeout)
	defer cancel()

	for {
		info, err := a.namespace.client.ContainerInspect(deadlineCtx, containerID)
		if err != nil {
			return fmt.Errorf("inspecting container health: %w", err)
		}
		if info.State != nil && info.State.Health != nil {
			switch info.State.Health.Status {
			case container.Healthy:
				return nil
			case container.Unhealthy:
				return fmt.Errorf("container %s is unhealthy", containerID)
			}
		}
		if deadlineCtx.Err() != nil {
			return fmt.Errorf("container %s did not become healthy", containerID)
		}
		time.Sleep(time.Second)
	}
}

func (a *Accessory) saveOperationResult(ctx context.Context, record func(*State)) {
	state, err := a.namespace.LoadState(ctx)
	if err != nil {
		return
	}
	record(state)
	_ = a.namespace.SaveState(ctx, state)
}

func (a *Accessory) saveDeployResult(ctx context.Context, err error) error {
	a.saveOperationResult(ctx, func(s *State) { s.RecordAccessoryDeploy(a.Settings.Name, err) })
	return nil
}

func (a *Accessory) removeOwnedVolumes(ctx context.Context) error {
	volumes, err := a.namespace.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return err
	}

	var errs []error
	prefix := a.namespace.accessoryVolumePrefix(a.Settings.Name)
	for _, vol := range volumes.Volumes {
		if strings.HasPrefix(vol.Name, prefix) {
			if err := a.namespace.client.VolumeRemove(ctx, vol.Name, true); err != nil && !errdefs.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("removing volume %s: %w", vol.Name, err))
			}
		}
	}
	return errors.Join(errs...)
}

func accessoryExposedPorts(ports []AccessoryPortBinding) nat.PortSet {
	exposed := nat.PortSet{}
	for _, port := range ports {
		exposed[nat.Port(fmt.Sprintf("%d/tcp", port.ContainerPort))] = struct{}{}
	}
	return exposed
}

func accessoryEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	env := make([]string, 0, len(values))
	for k, v := range values {
		env = append(env, k+"="+v)
	}
	return env
}

func mergeEnvMaps(base []string, override map[string]string) map[string]string {
	env := make(map[string]string, len(base)+len(override))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	for k, v := range override {
		env[k] = v
	}
	return env
}

func healthcheckConfig(settings AccessorySettings) *container.HealthConfig {
	switch settings.HealthCheck.Type {
	case AccessoryHealthCheckHTTP:
		port := settings.HealthCheck.Port
		if port == 0 {
			port = 80
		}
		path := settings.HealthCheck.Path
		if path == "" {
			path = "/"
		}
		return &container.HealthConfig{
			Test: []string{"CMD-SHELL", fmt.Sprintf("curl -fsS http://localhost:%d%s || exit 1", port, path)},
		}
	case AccessoryHealthCheckExec:
		if len(settings.HealthCheck.Command) == 0 {
			return nil
		}
		return &container.HealthConfig{
			Test: append([]string{"CMD"}, settings.HealthCheck.Command...),
		}
	default:
		return nil
	}
}

type accessoryRuntime struct {
	mounts []mount.Mount
}
