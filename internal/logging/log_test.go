package logging

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateDirUsesXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/once-state")

	dir, err := stateDir()

	require.NoError(t, err)
	assert.Equal(t, "/tmp/once-state/once", dir)
}

func TestToLogFileWritesLogAndReturnsError(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	wantErr := errors.New("task failed")

	err := ToLogFile(func() error {
		slog.Info("background task ran", "result", "failed")
		return wantErr
	})

	assert.ErrorIs(t, err, wantErr)
	contents, readErr := os.ReadFile(filepath.Join(stateHome, "once", "once.log"))
	require.NoError(t, readErr)
	assert.Contains(t, string(contents), "background task ran")
	assert.Contains(t, string(contents), "result=failed")
}
