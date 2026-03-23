package accessorytemplates

import "github.com/basecamp/once/internal/docker"

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
