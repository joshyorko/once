package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/basecamp/once/internal/docker"
)

type deployCommand struct {
	cmd        *cobra.Command
	host       string
	disableTLS bool
}

func newDeployCommand() *deployCommand {
	d := &deployCommand{}
	d.cmd = &cobra.Command{
		Use:   "deploy <image>",
		Short: "Deploy an application",
		Args:  cobra.ExactArgs(1),
		RunE:  WithNamespace(d.run),
	}
	d.cmd.Flags().StringVar(&d.host, "host", "", "hostname for the application (defaults to <name>.localhost)")
	d.cmd.Flags().BoolVar(&d.disableTLS, "disable-tls", false, "disable TLS for the application")
	return d
}

// Private

func (d *deployCommand) run(ctx context.Context, ns *docker.Namespace, cmd *cobra.Command, args []string) error {
	imageRef := args[0]

	if err := ns.Setup(ctx); err != nil {
		return fmt.Errorf("%w: %w", docker.ErrSetupFailed, err)
	}

	baseName := docker.NameFromImageRef(imageRef)
	name, err := ns.UniqueName(baseName)
	if err != nil {
		return fmt.Errorf("generating app name: %w", err)
	}

	host := d.host
	if host == "" {
		host = baseName + ".localhost"
	}

	if ns.HostInUse(host) {
		return docker.ErrHostnameInUse
	}

	app := docker.NewApplication(ns, docker.ApplicationSettings{
		Name:       name,
		Image:      imageRef,
		Host:       host,
		DisableTLS: d.disableTLS,
		AutoUpdate: true,
	})

	progress := func(p docker.DeployProgress) {
		switch p.Stage {
		case docker.DeployStageDownloading:
			fmt.Printf("Downloading: %d%%\n", p.Percentage)
		case docker.DeployStageStarting:
			fmt.Println("Starting...")
		case docker.DeployStageFinished:
			fmt.Println("Finished")
		}
	}

	if err := app.Deploy(ctx, progress); err != nil {
		if cleanupErr := cleanupFailedDeploy(app); cleanupErr != nil {
			slog.Error("Failed to clean up after deploy failure", "app", name, "error", cleanupErr)
		}
		return fmt.Errorf("%w: %w", docker.ErrDeployFailed, err)
	}

	fmt.Println("Verifying...")
	if err := app.VerifyHTTP(ctx); err != nil {
		if cleanupErr := cleanupFailedDeploy(app); cleanupErr != nil {
			slog.Error("Failed to clean up after verification failure", "app", name, "error", cleanupErr)
		}
		return err
	}

	fmt.Printf("Deployed %s\n", name)
	return nil
}

func cleanupFailedDeploy(app *docker.Application) error {
	if err := app.Remove(context.Background(), true); err != nil {
		return err
	}
	return nil
}
