package accessorytemplates

import (
	"fmt"
	"strings"

	"github.com/basecamp/once/internal/docker"
)

type Template struct {
	Alias          string
	Name           string
	Description    string
	Settings       docker.AccessorySettings
	RequiredEnv    []string
	RequiredMounts []string
}

func Builtins() []Template {
	return []Template{
		{
			Alias:       "cloudflared",
			Name:        "Cloudflare Tunnel",
			Description: "Runs cloudflared with a tunnel token.",
			Settings: docker.AccessorySettings{
				Scope:             docker.AccessoryScopeShared,
				Image:             "cloudflare/cloudflared:latest",
				Command:           []string{"tunnel", "run"},
				Proxy:             docker.AccessoryProxySettings{Enabled: false},
				InheritAppRuntime: false,
			},
			RequiredEnv: []string{"TUNNEL_TOKEN"},
		},
		{
			Alias:       "minio",
			Name:        "MinIO",
			Description: "Object storage with an optional console proxy.",
			Settings: docker.AccessorySettings{
				Scope:             docker.AccessoryScopeShared,
				Image:             "minio/minio:latest",
				Command:           []string{"server", "/data", "--console-address", ":9001"},
				Mounts:            []docker.AccessoryMount{{Type: docker.AccessoryMountVolume, Name: "data", Target: "/data"}},
				Proxy:             docker.AccessoryProxySettings{Enabled: false, TargetPort: 9001},
				InheritAppRuntime: false,
			},
			RequiredEnv: []string{"MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD"},
		},
		{
			Alias:       "prometheus",
			Name:        "Prometheus",
			Description: "Metrics collection scaffold with a config mount placeholder.",
			Settings: docker.AccessorySettings{
				Scope:             docker.AccessoryScopeShared,
				Image:             "prom/prometheus:latest",
				Mounts:            []docker.AccessoryMount{{Type: docker.AccessoryMountBind, Source: "/path/to/prometheus.yml", Target: "/etc/prometheus/prometheus.yml", ReadOnly: true}, {Type: docker.AccessoryMountVolume, Name: "data", Target: "/prometheus"}},
				Proxy:             docker.AccessoryProxySettings{Enabled: false},
				InheritAppRuntime: false,
			},
			RequiredMounts: []string{"/etc/prometheus/prometheus.yml"},
		},
		{
			Alias:       "alertmanager",
			Name:        "Alertmanager",
			Description: "Alertmanager scaffold with a config mount placeholder.",
			Settings: docker.AccessorySettings{
				Scope:             docker.AccessoryScopeShared,
				Image:             "prom/alertmanager:latest",
				Mounts:            []docker.AccessoryMount{{Type: docker.AccessoryMountBind, Source: "/path/to/alertmanager.yml", Target: "/etc/alertmanager/alertmanager.yml", ReadOnly: true}},
				Proxy:             docker.AccessoryProxySettings{Enabled: false},
				InheritAppRuntime: false,
			},
			RequiredMounts: []string{"/etc/alertmanager/alertmanager.yml"},
		},
	}
}

func ByAlias(alias string) (Template, bool) {
	for _, template := range Builtins() {
		if template.Alias == alias {
			return template, true
		}
	}
	return Template{}, false
}

func (t Template) Validate(settings docker.AccessorySettings) error {
	for _, key := range t.RequiredEnv {
		if strings.TrimSpace(settings.EnvVars[key]) == "" {
			return fmt.Errorf("missing required environment variable %q", key)
		}
	}

	for _, target := range t.RequiredMounts {
		found := false
		for _, mount := range settings.Mounts {
			if mount.Target != target {
				continue
			}
			if mount.Type == docker.AccessoryMountBind && mount.Source != "" && !strings.HasPrefix(mount.Source, "/path/to/") {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required bind mount for %q", target)
		}
	}

	return nil
}
