package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/basecamp/once/internal/accessorytemplates"
)

func TestInstallAccessoryStartsActivityAfterEnvSubmit(t *testing.T) {
	ns := newTestNamespace()
	m := NewInstallAccessory(ns, nil, "")
	m, _ = updateInstallAccessory(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	template, ok := accessorytemplates.ByAlias("cloudflared")
	require.True(t, ok)
	m.template = template
	m.settings = template.Settings
	m.settings.Name = "cloudflared"
	m.state = installAccessoryStateEnvironment

	comp, _ := m.Update(AccessoryEnvironmentSubmitMsg{
		EnvVars: map[string]string{"TUNNEL_TOKEN": "token"},
	})
	updated := comp.(InstallAccessory)

	assert.Equal(t, installAccessoryStateActivity, updated.state)
	require.NotNil(t, updated.activity)
}

func updateInstallAccessory(m InstallAccessory, msg tea.Msg) (InstallAccessory, tea.Cmd) {
	comp, cmd := m.Update(msg)
	return comp.(InstallAccessory), cmd
}
