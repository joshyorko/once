package accessorytemplates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/basecamp/once/internal/docker"
)

func TestTemplateValidateRequiresEnv(t *testing.T) {
	template, ok := ByAlias("cloudflared")
	require.True(t, ok)

	err := template.Validate(template.Settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TUNNEL_TOKEN")
}

func TestTemplateValidateRequiresConcreteBindMount(t *testing.T) {
	template, ok := ByAlias("prometheus")
	require.True(t, ok)

	err := template.Validate(template.Settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/etc/prometheus/prometheus.yml")

	settings := template.Settings
	settings.Mounts = []docker.AccessoryMount{
		{Type: docker.AccessoryMountBind, Source: "/tmp/prometheus.yml", Target: "/etc/prometheus/prometheus.yml", ReadOnly: true},
		{Type: docker.AccessoryMountVolume, Name: "data", Target: "/prometheus"},
	}

	require.NoError(t, template.Validate(settings))
}
