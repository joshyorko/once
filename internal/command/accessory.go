package command

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/basecamp/once/internal/accessorytemplates"
	"github.com/basecamp/once/internal/docker"
)

type accessoryCommand struct {
	cmd *cobra.Command
}

func newAccessoryCommand() *accessoryCommand {
	a := &accessoryCommand{}
	a.cmd = &cobra.Command{
		Use:   "accessory",
		Short: "Manage accessories",
	}
	a.cmd.AddCommand(newAccessoryListCommand().cmd)
	a.cmd.AddCommand(newAccessoryDeployCommand().cmd)
	a.cmd.AddCommand(newAccessoryStartCommand().cmd)
	a.cmd.AddCommand(newAccessoryStopCommand().cmd)
	a.cmd.AddCommand(newAccessoryRemoveCommand().cmd)
	a.cmd.AddCommand(newAccessoryLogsCommand().cmd)
	return a
}

type accessoryListCommand struct {
	cmd *cobra.Command
}

func newAccessoryListCommand() *accessoryListCommand {
	l := &accessoryListCommand{}
	l.cmd = &cobra.Command{
		Use:   "list",
		Short: "List installed accessories",
		RunE:  WithNamespace(l.run),
	}
	return l
}

func (l *accessoryListCommand) run(_ context.Context, ns *docker.Namespace, _ *cobra.Command, _ []string) error {
	for _, accessory := range ns.Accessories() {
		switch accessory.Settings.Scope {
		case docker.AccessoryScopePerApp:
			fmt.Printf("%s\t%s\n", accessory.Settings.Name, accessory.Settings.OwnerApp)
		default:
			fmt.Println(accessory.Settings.Name)
		}
	}
	return nil
}

type accessoryDeployCommand struct {
	cmd            *cobra.Command
	name           string
	image          string
	template       string
	appName        string
	cmdArgs        []string
	envVars        []string
	labels         []string
	volumes        []string
	binds          []string
	publishes      []string
	restart        string
	proxyHost      string
	proxyPort      int
	disableTLS     bool
	healthHTTPPort int
	healthHTTPPath string
	healthCmd      []string
}

func newAccessoryDeployCommand() *accessoryDeployCommand {
	d := &accessoryDeployCommand{}
	d.cmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an accessory",
		RunE:  WithNamespace(d.run),
	}
	d.cmd.Flags().StringVar(&d.name, "name", "", "accessory name")
	d.cmd.Flags().StringVar(&d.image, "image", "", "accessory image")
	d.cmd.Flags().StringVar(&d.template, "template", "", "built-in template alias")
	d.cmd.Flags().StringVar(&d.appName, "app", "", "owner app name")
	d.cmd.Flags().StringArrayVar(&d.cmdArgs, "cmd", nil, "container command")
	d.cmd.Flags().StringArrayVar(&d.envVars, "env", nil, "environment variable")
	d.cmd.Flags().StringArrayVar(&d.labels, "label", nil, "container label")
	d.cmd.Flags().StringArrayVar(&d.volumes, "volume", nil, "named volume mount")
	d.cmd.Flags().StringArrayVar(&d.binds, "bind", nil, "bind mount")
	d.cmd.Flags().StringArrayVar(&d.publishes, "publish", nil, "publish port")
	d.cmd.Flags().StringVar(&d.restart, "restart", "always", "restart policy")
	d.cmd.Flags().StringVar(&d.proxyHost, "proxy-host", "", "proxy host")
	d.cmd.Flags().IntVar(&d.proxyPort, "proxy-port", 0, "proxy target port")
	d.cmd.Flags().BoolVar(&d.disableTLS, "disable-tls", false, "disable proxy TLS")
	d.cmd.Flags().IntVar(&d.healthHTTPPort, "health-http-port", 0, "HTTP health check port")
	d.cmd.Flags().StringVar(&d.healthHTTPPath, "health-http-path", "", "HTTP health check path")
	d.cmd.Flags().StringArrayVar(&d.healthCmd, "health-cmd", nil, "exec health check command")
	return d
}

func (d *accessoryDeployCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, _ []string) error {
	if err := ns.Setup(ctx); err != nil {
		return fmt.Errorf("%w: %w", docker.ErrSetupFailed, err)
	}

	settings := docker.AccessorySettings{
		Name:          d.name,
		RestartPolicy: d.restart,
	}
	if d.name == "" {
		return fmt.Errorf("accessory name is required")
	}

	if template, ok := accessorytemplates.ByAlias(d.template); ok {
		settings = template.Settings
		settings.Name = d.name
	}

	if d.appName != "" {
		settings.Scope = docker.AccessoryScopePerApp
		settings.OwnerApp = d.appName
		settings.InheritAppRuntime = true
	}

	if d.image != "" {
		settings.Image = d.image
	}
	if len(d.cmdArgs) > 0 {
		settings.Command = append([]string(nil), d.cmdArgs...)
	}
	if settings.Scope == "" {
		settings.Scope = docker.AccessoryScopeShared
	}
	if settings.RestartPolicy == "" {
		settings.RestartPolicy = d.restart
	}

	if settings.EnvVars == nil {
		settings.EnvVars = map[string]string{}
	}
	for _, item := range d.envVars {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return fmt.Errorf("invalid env %q", item)
		}
		settings.EnvVars[key] = value
	}

	if settings.Labels == nil {
		settings.Labels = map[string]string{}
	}
	for _, item := range d.labels {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return fmt.Errorf("invalid label %q", item)
		}
		settings.Labels[key] = value
	}

	for _, item := range d.volumes {
		mount, err := docker.ParseAccessoryVolumeMount(item)
		if err != nil {
			return err
		}
		settings.Mounts = append(settings.Mounts, mount)
	}
	for _, item := range d.binds {
		mount, err := docker.ParseAccessoryBindMount(item)
		if err != nil {
			return err
		}
		settings.Mounts = append(settings.Mounts, mount)
	}

	for _, item := range d.publishes {
		port, err := docker.ParseAccessoryPortBinding(item)
		if err != nil {
			return err
		}
		settings.Ports = append(settings.Ports, port)
	}

	if d.proxyHost != "" || d.proxyPort != 0 || d.disableTLS {
		settings.Proxy.Enabled = true
		settings.Proxy.Host = d.proxyHost
		settings.Proxy.TargetPort = d.proxyPort
		settings.Proxy.DisableTLS = d.disableTLS
	}

	if d.healthHTTPPort != 0 || d.healthHTTPPath != "" {
		settings.HealthCheck = docker.AccessoryHealthCheckSettings{
			Type: docker.AccessoryHealthCheckHTTP,
			Port: d.healthHTTPPort,
			Path: d.healthHTTPPath,
		}
	}
	if len(d.healthCmd) > 0 {
		settings.HealthCheck = docker.AccessoryHealthCheckSettings{
			Type:    docker.AccessoryHealthCheckExec,
			Command: append([]string(nil), d.healthCmd...),
		}
	}

	accessory := docker.NewAccessory(ns, settings)
	if err := accessory.Deploy(ctx, nil); err != nil {
		return err
	}

	fmt.Printf("Deployed %s\n", settings.Name)
	return nil
}

type accessoryStartCommand struct {
	cmd *cobra.Command
}

func newAccessoryStartCommand() *accessoryStartCommand {
	s := &accessoryStartCommand{}
	s.cmd = &cobra.Command{
		Use:   "start <accessory>",
		Short: "Start an accessory",
		Args:  cobra.ExactArgs(1),
		RunE:  WithNamespace(s.run),
	}
	return s
}

func (s *accessoryStartCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, args []string) error {
	return withAccessory(ns, args[0], "starting", func(accessory *docker.Accessory) error {
		return accessory.Start(ctx)
	})
}

type accessoryStopCommand struct {
	cmd *cobra.Command
}

func newAccessoryStopCommand() *accessoryStopCommand {
	s := &accessoryStopCommand{}
	s.cmd = &cobra.Command{
		Use:   "stop <accessory>",
		Short: "Stop an accessory",
		Args:  cobra.ExactArgs(1),
		RunE:  WithNamespace(s.run),
	}
	return s
}

func (s *accessoryStopCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, args []string) error {
	return withAccessory(ns, args[0], "stopping", func(accessory *docker.Accessory) error {
		return accessory.Stop(ctx)
	})
}

type accessoryRemoveCommand struct {
	cmd        *cobra.Command
	removeData bool
}

func newAccessoryRemoveCommand() *accessoryRemoveCommand {
	r := &accessoryRemoveCommand{}
	r.cmd = &cobra.Command{
		Use:   "remove <accessory>",
		Short: "Remove an accessory",
		Args:  cobra.ExactArgs(1),
		RunE:  WithNamespace(r.run),
	}
	r.cmd.Flags().BoolVar(&r.removeData, "remove-data", false, "Also remove accessory data volumes")
	return r
}

func (r *accessoryRemoveCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, args []string) error {
	return withAccessory(ns, args[0], "removing", func(accessory *docker.Accessory) error {
		return accessory.Remove(ctx, r.removeData)
	})
}

type accessoryLogsCommand struct {
	cmd    *cobra.Command
	lines  int
	follow bool
}

func newAccessoryLogsCommand() *accessoryLogsCommand {
	l := &accessoryLogsCommand{}
	l.cmd = &cobra.Command{
		Use:   "logs <accessory>",
		Short: "Show accessory logs",
		Args:  cobra.ExactArgs(1),
		RunE:  WithNamespace(l.run),
	}
	l.cmd.Flags().IntVar(&l.lines, "lines", 200, "number of log lines to show")
	l.cmd.Flags().BoolVarP(&l.follow, "follow", "f", false, "follow log output")
	return l
}

func (l *accessoryLogsCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, args []string) error {
	return withAccessory(ns, args[0], "streaming logs for", func(accessory *docker.Accessory) error {
		streamer, err := accessory.NewLogStreamer(ctx, docker.LogStreamerSettings{})
		if err != nil {
			return err
		}
		defer streamer.Stop()

		lastVersion := uint64(0)
		printedCount := 0
		printFetched := func() {
			lines := streamer.Fetch(l.lines)
			for _, line := range lines {
				fmt.Fprintln(os.Stdout, line.Content)
			}
			lastVersion = streamer.Version()
			printedCount = len(lines)
		}

		deadline := time.Now().Add(5 * time.Second)
		for !streamer.Ready() {
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		printFetched()
		if !l.follow {
			return nil
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
				if streamer.Version() == lastVersion {
					continue
				}
				lines := streamer.Fetch(streamer.Count())
				start := printedCount
				if printedCount > len(lines) {
					start = 0
				}
				for _, line := range lines[start:] {
					fmt.Fprintln(os.Stdout, line.Content)
				}
				lastVersion = streamer.Version()
				printedCount = len(lines)
			}
		}
	})
}

// Private

func withAccessory(ns *docker.Namespace, name string, action string, fn func(*docker.Accessory) error) error {
	accessory := ns.Accessory(name)
	if accessory == nil {
		return fmt.Errorf("accessory %q not found", name)
	}
	if err := fn(accessory); err != nil {
		return fmt.Errorf("%s accessory: %w", action, err)
	}
	return nil
}
