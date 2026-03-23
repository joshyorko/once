package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessoryDisplayNameFallsBackToImageName(t *testing.T) {
	accessory := &Accessory{
		Settings: AccessorySettings{
			Image: "cloudflare/cloudflared:latest",
		},
	}

	assert.Equal(t, "cloudflared", accessory.DisplayName())
	assert.Equal(t, "cloudflared", accessory.StatsName())
}

func TestAccessoryDisplayNameFallsBackToProxyHost(t *testing.T) {
	accessory := &Accessory{
		Settings: AccessorySettings{
			Proxy: AccessoryProxySettings{Host: "tunnel.example.com"},
		},
	}

	assert.Equal(t, "tunnel.example.com", accessory.DisplayName())
	assert.Empty(t, accessory.StatsName())
}
