package docker

import (
	"encoding/json"
	"reflect"
)

type AccessoryScope string

const (
	AccessoryScopeShared AccessoryScope = "shared"
	AccessoryScopePerApp AccessoryScope = "per_app"
)

type AccessoryMountType string

const (
	AccessoryMountVolume AccessoryMountType = "volume"
	AccessoryMountBind   AccessoryMountType = "bind"
)

type AccessoryMount struct {
	Type     AccessoryMountType `json:"type"`
	Name     string             `json:"name,omitempty"`
	Source   string             `json:"source,omitempty"`
	Target   string             `json:"target"`
	ReadOnly bool               `json:"readOnly,omitempty"`
}

type AccessoryPortBinding struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	HostIP        string `json:"hostIP,omitempty"`
}

type AccessoryProxySettings struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Host       string `json:"host,omitempty"`
	DisableTLS bool   `json:"disableTLS,omitempty"`
	TargetPort int    `json:"targetPort,omitempty"`
}

type AccessoryHealthCheckType string

const (
	AccessoryHealthCheckNone AccessoryHealthCheckType = "none"
	AccessoryHealthCheckHTTP AccessoryHealthCheckType = "http"
	AccessoryHealthCheckExec AccessoryHealthCheckType = "exec"
)

type AccessoryHealthCheckSettings struct {
	Type    AccessoryHealthCheckType `json:"type,omitempty"`
	Port    int                      `json:"port,omitempty"`
	Path    string                   `json:"path,omitempty"`
	Command []string                 `json:"command,omitempty"`
}

type AccessorySettings struct {
	Name              string                       `json:"name"`
	Image             string                       `json:"image,omitempty"`
	Scope             AccessoryScope               `json:"scope"`
	OwnerApp          string                       `json:"ownerApp,omitempty"`
	InheritAppRuntime bool                         `json:"inheritAppRuntime,omitempty"`
	Command           []string                     `json:"command,omitempty"`
	EnvVars           map[string]string            `json:"env,omitempty"`
	Mounts            []AccessoryMount             `json:"mounts,omitempty"`
	Ports             []AccessoryPortBinding       `json:"ports,omitempty"`
	Labels            map[string]string            `json:"labels,omitempty"`
	Resources         ContainerResources           `json:"resources,omitempty"`
	RestartPolicy     string                       `json:"restartPolicy,omitempty"`
	Proxy             AccessoryProxySettings       `json:"proxy,omitempty"`
	HealthCheck       AccessoryHealthCheckSettings `json:"healthCheck,omitempty"`
}

func UnmarshalAccessorySettings(s string) (AccessorySettings, error) {
	var settings AccessorySettings
	err := json.Unmarshal([]byte(s), &settings)
	return settings, err
}

func (s AccessorySettings) Marshal() string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (s AccessorySettings) Equal(other AccessorySettings) bool {
	return reflect.DeepEqual(s, other)
}
