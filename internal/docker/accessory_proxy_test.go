package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessoryProxyDeployOptions_HTTPHealthCheck(t *testing.T) {
	accessory := &Accessory{
		Settings: AccessorySettings{Name: "prometheus"},
	}

	opts := accessory.proxyDeployOptions("container123:9090", AccessorySettings{
		Proxy: AccessoryProxySettings{
			Enabled:    true,
			Host:       "prom.localhost",
			TargetPort: 9090,
		},
		HealthCheck: AccessoryHealthCheckSettings{
			Type: AccessoryHealthCheckHTTP,
			Path: "/-/healthy",
			Port: 9090,
		},
	})

	assert.False(t, opts.Force)
	assert.Equal(t, "/-/healthy", opts.HealthCheckPath)
	assert.Equal(t, 9090, opts.HealthCheckPort)
}

func TestAccessoryProxyDeployOptions_ForceWithoutHTTPHealthCheck(t *testing.T) {
	accessory := &Accessory{
		Settings: AccessorySettings{Name: "worker"},
	}

	opts := accessory.proxyDeployOptions("container123", AccessorySettings{
		Proxy: AccessoryProxySettings{
			Enabled: true,
			Host:    "worker.localhost",
		},
	})

	assert.True(t, opts.Force)
	assert.Empty(t, opts.HealthCheckPath)
	assert.Zero(t, opts.HealthCheckPort)
}

func TestHealthcheckConfigHTTPUsesCurlOrWget(t *testing.T) {
	cfg := healthcheckConfig(AccessorySettings{
		HealthCheck: AccessoryHealthCheckSettings{
			Type: AccessoryHealthCheckHTTP,
			Port: 9090,
			Path: "/-/healthy",
		},
	})

	require.NotNil(t, cfg)
	require.Len(t, cfg.Test, 2)
	assert.Equal(t, "CMD-SHELL", cfg.Test[0])
	assert.Contains(t, cfg.Test[1], "command -v curl")
	assert.Contains(t, cfg.Test[1], "command -v wget")
	assert.Contains(t, cfg.Test[1], "http://localhost:9090/-/healthy")
}
