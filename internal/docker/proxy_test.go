package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyContainerName(t *testing.T) {
	ns := &Namespace{name: "once"}
	proxy := NewProxy(ns)
	assert.Equal(t, "once-proxy", proxy.containerName())

	ns2 := &Namespace{name: "staging"}
	proxy2 := NewProxy(ns2)
	assert.Equal(t, "staging-proxy", proxy2.containerName())
}

func TestDeployArgs(t *testing.T) {
	proxy := &Proxy{}

	t.Run("basic deploy includes timeout", func(t *testing.T) {
		args := proxy.deployArgs(DeployOptions{ServiceName: "chat", Target: "localhost:3000"})

		assert.Equal(t, []string{
			"kamal-proxy", "deploy", "chat",
			"--target", "localhost:3000",
			"--deploy-timeout", "120s",
		}, args)
	})

	t.Run("with host", func(t *testing.T) {
		args := proxy.deployArgs(DeployOptions{ServiceName: "chat", Target: "localhost:3000", Host: "chat.example.com"})

		assert.Contains(t, args, "--host")
		assert.Contains(t, args, "chat.example.com")
	})

	t.Run("with TLS", func(t *testing.T) {
		args := proxy.deployArgs(DeployOptions{ServiceName: "chat", Target: "localhost:3000", TLS: true})

		assert.Contains(t, args, "--tls")
	})

	t.Run("with host and TLS", func(t *testing.T) {
		args := proxy.deployArgs(DeployOptions{
			ServiceName: "chat",
			Target:      "localhost:3000",
			Host:        "chat.example.com",
			TLS:         true,
		})

		assert.Equal(t, []string{
			"kamal-proxy", "deploy", "chat",
			"--target", "localhost:3000",
			"--deploy-timeout", "120s",
			"--host", "chat.example.com",
			"--tls",
		}, args)
	})

	t.Run("with force and health check", func(t *testing.T) {
		args := proxy.deployArgs(DeployOptions{
			ServiceName:     "metrics",
			Target:          "localhost:9090",
			Host:            "metrics.example.com",
			Force:           true,
			HealthCheckPath: "/-/healthy",
			HealthCheckPort: 9090,
		})

		assert.Equal(t, []string{
			"kamal-proxy", "deploy", "metrics",
			"--target", "localhost:9090",
			"--deploy-timeout", "120s",
			"--host", "metrics.example.com",
			"--force",
			"--health-check-path", "/-/healthy",
			"--health-check-port", "9090",
		}, args)
	})
}
