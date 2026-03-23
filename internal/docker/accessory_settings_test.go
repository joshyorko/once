package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessorySettingsMarshalRoundTrip(t *testing.T) {
	original := AccessorySettings{
		Name:              "minio",
		Image:             "minio/minio:latest",
		Scope:             AccessoryScopePerApp,
		OwnerApp:          "app",
		InheritAppRuntime: true,
		Command:           []string{"server", "/data"},
		EnvVars: map[string]string{
			"A": "1",
		},
		Mounts: []AccessoryMount{{Type: AccessoryMountVolume, Name: "data", Target: "/data"}},
		Ports:  []AccessoryPortBinding{{ContainerPort: 9001, HostPort: 9001}},
		Labels: map[string]string{"team": "platform"},
		Resources: ContainerResources{
			CPUs:     2,
			MemoryMB: 512,
		},
		RestartPolicy: "always",
		Proxy: AccessoryProxySettings{
			Enabled:    true,
			Host:       "minio.example.com",
			TargetPort: 9001,
		},
		HealthCheck: AccessoryHealthCheckSettings{
			Type: AccessoryHealthCheckHTTP,
			Port: 9001,
			Path: "/health",
		},
	}

	restored, err := UnmarshalAccessorySettings(original.Marshal())
	require.NoError(t, err)
	assert.True(t, original.Equal(restored))
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.Proxy.Host, restored.Proxy.Host)
}
