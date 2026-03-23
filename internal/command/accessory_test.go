package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/basecamp/once/internal/docker"
)

func TestParseVolumeMount(t *testing.T) {
	mount, err := docker.ParseAccessoryVolumeMount("data:/data:ro")
	require.NoError(t, err)
	assert.Equal(t, "data", mount.Name)
	assert.Equal(t, "/data", mount.Target)
	assert.True(t, mount.ReadOnly)
}

func TestParseBindMount(t *testing.T) {
	mount, err := docker.ParseAccessoryBindMount("/tmp/prometheus.yml:/etc/prometheus/prometheus.yml")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/prometheus.yml", mount.Source)
	assert.Equal(t, "/etc/prometheus/prometheus.yml", mount.Target)
}

func TestParsePortBinding(t *testing.T) {
	port, err := docker.ParseAccessoryPortBinding("127.0.0.1:9001:9000")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", port.HostIP)
	assert.Equal(t, 9001, port.HostPort)
	assert.Equal(t, 9000, port.ContainerPort)
}
