package service

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReturnsPlatformService(t *testing.T) {
	service, err := New()

	switch runtime.GOOS {
	case "linux":
		require.NoError(t, err)
		assert.IsType(t, &Systemd{}, service)
	case "darwin":
		require.NoError(t, err)
		assert.IsType(t, &Launchd{}, service)
	default:
		require.Error(t, err)
		assert.Nil(t, service)
	}
}

func TestSystemdNames(t *testing.T) {
	service := &Systemd{}

	assert.Equal(t, "once.service", service.ServiceName("once"))
	assert.Equal(t, "/etc/systemd/system/once.service", service.unitFilePath("once"))
}

func TestLaunchdNames(t *testing.T) {
	service := &Launchd{}

	assert.Equal(t, "com.basecamp.once", service.ServiceName("once"))
	assert.Equal(t, "/Library/LaunchDaemons/com.basecamp.once.plist", service.plistPath("once"))
}
