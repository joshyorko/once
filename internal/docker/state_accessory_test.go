package docker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateBackwardCompatibility(t *testing.T) {
	raw := []byte(`{"apps":{"app":{"lastBackup":{"at":"2025-01-01T00:00:00Z","error":""},"lastUpdate":{"at":"2025-01-01T00:00:00Z","error":""}}}}`)

	var state State
	require.NoError(t, json.Unmarshal(raw, &state))
	require.NotNil(t, state.Apps)
	assert.Nil(t, state.Accessories)
	assert.NotNil(t, state.AppState("app"))
}

func TestAccessoryStateRecorders(t *testing.T) {
	state := &State{}
	state.RecordAccessoryDeploy("minio", nil)
	state.RecordAccessoryHealthCheck("minio", nil)

	require.NotNil(t, state.Accessories["minio"])
	assert.False(t, state.Accessories["minio"].LastDeploy.At.IsZero())
	assert.False(t, state.Accessories["minio"].LastHealthCheck.At.IsZero())
}
