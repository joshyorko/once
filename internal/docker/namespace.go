package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const DefaultNamespace = "once"

var ErrInvalidNamespace = errors.New("invalid namespace: must contain only lowercase letters, digits, and hyphens, and must not start with a hyphen")

type Namespace struct {
	name         string
	client       *client.Client
	proxy        *Proxy
	applications []*Application
	accessories  []*Accessory
}

type NamespaceOption func(*Namespace)

func WithApplications(apps ...ApplicationSettings) NamespaceOption {
	return func(ns *Namespace) {
		for _, s := range apps {
			ns.addApplication(s)
		}
	}
}

func WithAccessories(accessories ...AccessorySettings) NamespaceOption {
	return func(ns *Namespace) {
		for _, s := range accessories {
			ns.addAccessory(s)
		}
	}
}

func NewNamespace(name string, opts ...NamespaceOption) (*Namespace, error) {
	if name == "" {
		name = DefaultNamespace
	}

	if !validNamespace.MatchString(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidNamespace, name)
	}

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ns := &Namespace{
		name:   name,
		client: c,
	}
	ns.proxy = NewProxy(ns)

	for _, opt := range opts {
		opt(ns)
	}

	return ns, nil
}

func RestoreNamespace(ctx context.Context, name string) (*Namespace, error) {
	ns, err := NewNamespace(name)
	if err != nil {
		return nil, err
	}

	if err := ns.restoreState(ctx); err != nil {
		return nil, err
	}

	return ns, nil
}

func (n *Namespace) Name() string {
	return n.name
}

func (n *Namespace) addApplication(settings ApplicationSettings) *Application {
	app := NewApplication(n, settings)
	n.applications = append(n.applications, app)
	n.sortApplications()
	return app
}

func (n *Namespace) addAccessory(settings AccessorySettings) *Accessory {
	accessory := NewAccessory(n, settings)
	n.accessories = append(n.accessories, accessory)
	n.sortAccessories()
	return accessory
}

func (n *Namespace) Proxy() *Proxy {
	return n.proxy
}

func (n *Namespace) Application(name string) *Application {
	for _, app := range n.applications {
		if app.Settings.Name == name {
			return app
		}
	}
	return nil
}

func (n *Namespace) Accessory(name string) *Accessory {
	for _, accessory := range n.accessories {
		if accessory.Settings.Name == name {
			return accessory
		}
	}
	return nil
}

func (n *Namespace) Applications() []*Application {
	return n.applications
}

func (n *Namespace) Accessories() []*Accessory {
	return n.accessories
}

func (n *Namespace) SharedAccessories() []*Accessory {
	var accessories []*Accessory
	for _, accessory := range n.accessories {
		if accessory.Settings.Scope == AccessoryScopeShared {
			accessories = append(accessories, accessory)
		}
	}
	return accessories
}

func (n *Namespace) AccessoriesForApp(appName string) []*Accessory {
	var accessories []*Accessory
	for _, accessory := range n.accessories {
		if accessory.Settings.Scope == AccessoryScopePerApp && accessory.Settings.OwnerApp == appName {
			accessories = append(accessories, accessory)
		}
	}
	return accessories
}
func (n *Namespace) ApplicationByHost(host string) *Application {
	for _, app := range n.applications {
		if app.Settings.Host == host {
			return app
		}
	}
	return nil
}

func (n *Namespace) HostInUse(host string) bool {
	if n.ApplicationByHost(host) != nil {
		return true
	}
	for _, accessory := range n.accessories {
		if accessory.Settings.Proxy.Enabled && accessory.Settings.Proxy.Host == host {
			return true
		}
	}
	return false
}

func (n *Namespace) HostInUseByAnother(host string, excludeApp string) bool {
	for _, app := range n.applications {
		if app.Settings.Host == host && app.Settings.Name != excludeApp {
			return true
		}
	}
	for _, accessory := range n.accessories {
		if accessory.Settings.Proxy.Enabled && accessory.Settings.Proxy.Host == host {
			return true
		}
	}
	return false
}

func (n *Namespace) UniqueName(base string) (string, error) {
	for {
		id, err := randomID(6)
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("%s.%s", base, id)
		if !n.serviceNameInUse(candidate) {
			return candidate, nil
		}
	}
}

func (n *Namespace) Setup(ctx context.Context) error {
	if err := n.EnsureNetwork(ctx); err != nil {
		return err
	}

	return n.proxy.Boot(ctx, ProxySettings{})
}

func (n *Namespace) EnsureNetwork(ctx context.Context) error {
	networks, err := n.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}

	for _, net := range networks {
		if net.Name == n.name {
			return nil
		}
	}

	_, err = n.client.NetworkCreate(ctx, n.name, network.CreateOptions{
		Driver: "bridge",
	})
	return err
}

func (n *Namespace) Teardown(ctx context.Context, destroyVolumes bool) error {
	var errs []error
	for _, accessory := range n.accessories {
		if err := accessory.Destroy(ctx, destroyVolumes); err != nil {
			errs = append(errs, err)
		}
	}
	for _, app := range n.applications {
		if err := app.Destroy(ctx, destroyVolumes); err != nil {
			errs = append(errs, err)
		}
	}

	if err := n.proxy.Destroy(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := n.client.NetworkRemove(ctx, n.name); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (n *Namespace) Refresh(ctx context.Context) error {
	n.applications = nil
	n.accessories = nil
	return n.restoreState(ctx)
}

func (n *Namespace) DockerRootDir(ctx context.Context) (string, error) {
	info, err := n.client.Info(ctx)
	if err != nil {
		return "", err
	}
	return info.DockerRootDir, nil
}

func (n *Namespace) EventWatcher() *EventWatcher {
	return NewEventWatcher(n.client, n.name)
}

func (n *Namespace) ApplicationExists(ctx context.Context, name string) (bool, error) {
	containers, err := n.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return false, err
	}

	for _, c := range containers {
		for _, cname := range c.Names {
			cname = strings.TrimPrefix(cname, "/")
			if n.containerAppName(cname) == name {
				return true, nil
			}
		}
	}

	return false, nil
}

func (n *Namespace) LoadState(ctx context.Context) (*State, error) {
	return n.proxy.LoadState(ctx)
}

func (n *Namespace) SaveState(ctx context.Context, state *State) error {
	return n.proxy.SaveState(ctx, state)
}

func (n *Namespace) Restore(ctx context.Context, r io.ReadSeeker) (*Application, error) {
	appSettings, volSettings, err := readBackupSettings(r)
	if err != nil {
		return nil, fmt.Errorf("parsing backup: %w", err)
	}

	if n.HostInUse(appSettings.Host) {
		return nil, ErrHostnameInUse
	}

	name, err := n.UniqueName(NameFromImageRef(appSettings.Image))
	if err != nil {
		return nil, fmt.Errorf("generating app name: %w", err)
	}
	appSettings.Name = name

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewinding backup: %w", err)
	}

	app := NewApplication(n, appSettings)
	if err := app.Restore(ctx, volSettings, r); err != nil {
		if cleanupErr := app.Destroy(context.Background(), true); cleanupErr != nil {
			slog.Error("Failed to clean up after restore failure", "app", appSettings.Name, "error", cleanupErr)
		}
		return nil, err
	}

	if err := n.Refresh(ctx); err != nil {
		slog.Error("Failed to refresh namespace after restore", "app", appSettings.Name, "error", err)
	}

	if restored := n.Application(appSettings.Name); restored != nil {
		return restored, nil
	}
	return app, nil
}

// containerAppName extracts the application name from a container name
// matching the pattern {namespace}-app-{appName}-{id}. Returns "" if the
// container name doesn't match.
func (n *Namespace) containerAppName(containerName string) string {
	after, ok := strings.CutPrefix(containerName, n.name+"-app-")
	if !ok {
		return ""
	}
	appName, _, ok := cutLast(after, "-")
	if !ok {
		return ""
	}
	return appName
}

// containerAccessoryName extracts the accessory name from a container name
// matching the pattern {namespace}-accessory-{accessoryName}-{id}. Returns ""
// if the container name doesn't match.
func (n *Namespace) containerAccessoryName(containerName string) string {
	after, ok := strings.CutPrefix(containerName, n.name+"-accessory-")
	if !ok {
		return ""
	}
	accessoryName, _, ok := cutLast(after, "-")
	if !ok {
		return ""
	}
	return accessoryName
}

// Private

type appCandidate struct {
	app     *Application
	created int64
}

type accessoryCandidate struct {
	accessory *Accessory
	created   int64
}

func (n *Namespace) restoreState(ctx context.Context) error {
	containers, err := n.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	proxyPrefix := n.name + "-proxy"
	appPrefix := n.name + "-app-"
	accessoryPrefix := n.name + "-accessory-"

	// Use a map to deduplicate apps by name, preferring the most recently created container
	appsByName := make(map[string]appCandidate)
	accessoriesByName := make(map[string]accessoryCandidate)

	for _, c := range containers {
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")

			if name == proxyPrefix {
				label := c.Labels[labelKey]
				if label != "" {
					settings, err := UnmarshalProxySettings(label)
					if err != nil {
						return err
					}
					n.proxy.Settings = &settings
				}
				break
			}

			if strings.HasPrefix(name, appPrefix) {
				label := c.Labels[labelKey]
				if label != "" {
					settings, err := UnmarshalApplicationSettings(label)
					if err != nil {
						return err
					}
					app := NewApplication(n, settings)
					app.Running = c.State == "running"
					if app.Running {
						info, err := n.client.ContainerInspect(ctx, c.ID)
						if err == nil && info.State != nil {
							if t, err := time.Parse(time.RFC3339Nano, info.State.StartedAt); err == nil {
								app.RunningSince = t
							}
						}
					}

					existing, found := appsByName[settings.Name]
					if !found || c.Created > existing.created {
						appsByName[settings.Name] = appCandidate{app: app, created: c.Created}
					}
				}
				break
			}

			if strings.HasPrefix(name, accessoryPrefix) {
				label := c.Labels[labelKey]
				if label != "" {
					settings, err := UnmarshalAccessorySettings(label)
					if err != nil {
						return err
					}
					accessory := NewAccessory(n, settings)
					accessory.Running = c.State == "running"
					if accessory.Running {
						info, err := n.client.ContainerInspect(ctx, c.ID)
						if err == nil && info.State != nil {
							if t, err := time.Parse(time.RFC3339Nano, info.State.StartedAt); err == nil {
								accessory.RunningSince = t
							}
						}
					}

					existing, found := accessoriesByName[settings.Name]
					if !found || c.Created > existing.created {
						accessoriesByName[settings.Name] = accessoryCandidate{accessory: accessory, created: c.Created}
					}
				}
				break
			}
		}
	}

	for _, candidate := range appsByName {
		n.applications = append(n.applications, candidate.app)
	}
	for _, candidate := range accessoriesByName {
		n.accessories = append(n.accessories, candidate.accessory)
	}

	n.sortApplications()
	n.sortAccessories()
	return nil
}

func (n *Namespace) sortApplications() {
	slices.SortFunc(n.applications, func(a, b *Application) int {
		return strings.Compare(a.Settings.Host, b.Settings.Host)
	})
}

func (n *Namespace) sortAccessories() {
	slices.SortFunc(n.accessories, func(a, b *Accessory) int {
		if a.Settings.Scope != b.Settings.Scope {
			return strings.Compare(string(a.Settings.Scope), string(b.Settings.Scope))
		}
		if a.Settings.OwnerApp != b.Settings.OwnerApp {
			return strings.Compare(a.Settings.OwnerApp, b.Settings.OwnerApp)
		}
		return strings.Compare(a.Settings.Name, b.Settings.Name)
	})
}

func (n *Namespace) serviceNameInUse(name string) bool {
	if n.Application(name) != nil || n.Accessory(name) != nil {
		return true
	}
	return false
}

func (n *Namespace) accessoryVolumePrefix(accessoryName string) string {
	return fmt.Sprintf("%s-accessory-%s-", n.name, accessoryName)
}

func (n *Namespace) accessoryVolumeName(accessoryName, mountName string) string {
	return fmt.Sprintf("%s-accessory-%s-%s", n.name, accessoryName, mountName)
}

// Helpers

var validNamespace = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
