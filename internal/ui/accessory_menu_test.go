package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/basecamp/once/internal/docker"
)

func TestAccessoryActionsMenu_SelectRemove(t *testing.T) {
	accessory := &docker.Accessory{Settings: docker.AccessorySettings{Name: "minio"}}
	m := NewAccessoryActionsMenu(accessory)

	_, cmd := updateAccessoryActionsMenu(m, keyPressMsg("r"))
	require.NotNil(t, cmd)

	_, cmd = updateAccessoryActionsMenu(m, cmd())
	require.NotNil(t, cmd)

	msg := cmd()
	selectMsg, ok := msg.(AccessoryActionsMenuSelectMsg)
	require.True(t, ok, "expected AccessoryActionsMenuSelectMsg, got %T", msg)
	assert.Equal(t, AccessoryActionsMenuRemove, selectMsg.action)
}

func TestAccessorySettingsMenu_SelectGeneral(t *testing.T) {
	accessory := &docker.Accessory{Settings: docker.AccessorySettings{Name: "minio"}}
	m := NewAccessorySettingsMenu(accessory)

	_, cmd := updateAccessorySettingsMenu(m, keyPressMsg("g"))
	require.NotNil(t, cmd)

	_, cmd = updateAccessorySettingsMenu(m, cmd())
	require.NotNil(t, cmd)

	msg := cmd()
	selectMsg, ok := msg.(AccessorySettingsMenuSelectMsg)
	require.True(t, ok, "expected AccessorySettingsMenuSelectMsg, got %T", msg)
	assert.Equal(t, AccessorySettingsSectionGeneral, selectMsg.section)
}

func updateAccessoryActionsMenu(m AccessoryActionsMenu, msg tea.Msg) (AccessoryActionsMenu, tea.Cmd) {
	comp, cmd := m.Update(msg)
	return comp.(AccessoryActionsMenu), cmd
}

func updateAccessorySettingsMenu(m AccessorySettingsMenu, msg tea.Msg) (AccessorySettingsMenu, tea.Cmd) {
	comp, cmd := m.Update(msg)
	return comp.(AccessorySettingsMenu), cmd
}
