package docker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerAccessoryName(t *testing.T) {
	ns := &Namespace{name: "once"}

	t.Run("standard accessory", func(t *testing.T) {
		assert.Equal(t, "minio", ns.containerAccessoryName("once-accessory-minio-a1b2c3"))
	})

	t.Run("dotted unique name", func(t *testing.T) {
		assert.Equal(t, "minio.a1b2c3", ns.containerAccessoryName("once-accessory-minio.a1b2c3-d4e5f6"))
	})

	t.Run("dashed accessory name", func(t *testing.T) {
		assert.Equal(t, "my-accessory", ns.containerAccessoryName("once-accessory-my-accessory-abcdef"))
	})
}

func TestNamespaceHostCollisionIncludesAccessories(t *testing.T) {
	ns := &Namespace{name: "once"}
	ns.applications = []*Application{
		NewApplication(ns, ApplicationSettings{Name: "app", Host: "app.example.com"}),
	}
	ns.accessories = []*Accessory{
		NewAccessory(ns, AccessorySettings{Name: "minio", Proxy: AccessoryProxySettings{Enabled: true, Host: "minio.example.com"}}),
	}

	assert.True(t, ns.HostInUse("app.example.com"))
	assert.True(t, ns.HostInUse("minio.example.com"))
	assert.False(t, ns.HostInUse("other.example.com"))
}

func TestNamespaceUniqueNameAvoidsAccessories(t *testing.T) {
	ns := &Namespace{name: "once"}
	ns.applications = []*Application{NewApplication(ns, ApplicationSettings{Name: "app"})}
	ns.accessories = []*Accessory{NewAccessory(ns, AccessorySettings{Name: "app.abcdef"})}

	name, err := ns.UniqueName("app")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(name, "app."))
	assert.NotEqual(t, "app.abcdef", name)
}
