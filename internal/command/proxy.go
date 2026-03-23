package command

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/basecamp/once/internal/docker"
)

type proxyCommand struct {
	cmd *cobra.Command
}

func newProxyCommand() *proxyCommand {
	p := &proxyCommand{}
	p.cmd = &cobra.Command{
		Use:   "proxy",
		Short: "Manage the Once proxy",
	}
	p.cmd.AddCommand(newProxyShowCommand().cmd)
	p.cmd.AddCommand(newProxyConfigureCommand().cmd)
	return p
}

type proxyShowCommand struct {
	cmd *cobra.Command
}

func newProxyShowCommand() *proxyShowCommand {
	s := &proxyShowCommand{}
	s.cmd = &cobra.Command{
		Use:   "show",
		Short: "Show proxy settings",
		RunE:  WithNamespace(s.run),
	}
	return s
}

func (s *proxyShowCommand) run(_ context.Context, ns *docker.Namespace, _ *cobra.Command, _ []string) error {
	settings := docker.ProxySettings{
		BindAddress: "0.0.0.0",
		HTTPPort:    docker.DefaultHTTPPort,
		HTTPSPort:   docker.DefaultHTTPSPort,
		MetricsPort: docker.DefaultMetricsPort,
	}
	if ns.Proxy().Settings != nil {
		settings = *ns.Proxy().Settings
	}
	fmt.Printf("bind=%s http=%d https=%d metrics=%d\n", settings.BindAddress, settings.HTTPPort, settings.HTTPSPort, settings.MetricsPort)
	return nil
}

type proxyConfigureCommand struct {
	cmd         *cobra.Command
	bind        string
	httpPort    int
	httpsPort   int
	metricsPort int
}

func newProxyConfigureCommand() *proxyConfigureCommand {
	c := &proxyConfigureCommand{}
	c.cmd = &cobra.Command{
		Use:   "configure",
		Short: "Configure proxy ports and bind address",
		RunE:  WithNamespace(c.run),
	}
	c.cmd.Flags().StringVar(&c.bind, "bind", "0.0.0.0", "bind address for HTTP and HTTPS")
	c.cmd.Flags().IntVar(&c.httpPort, "http-port", docker.DefaultHTTPPort, "HTTP port")
	c.cmd.Flags().IntVar(&c.httpsPort, "https-port", docker.DefaultHTTPSPort, "HTTPS port")
	c.cmd.Flags().IntVar(&c.metricsPort, "metrics-port", docker.DefaultMetricsPort, "metrics port")
	return c
}

func (c *proxyConfigureCommand) run(ctx context.Context, ns *docker.Namespace, _ *cobra.Command, _ []string) error {
	if err := ns.EnsureNetwork(ctx); err != nil {
		return err
	}
	if err := ns.Proxy().ApplySettings(ctx, docker.ProxySettings{
		BindAddress: c.bind,
		HTTPPort:    c.httpPort,
		HTTPSPort:   c.httpsPort,
		MetricsPort: c.metricsPort,
	}); err != nil {
		return err
	}
	fmt.Println("Proxy configured")
	return nil
}
