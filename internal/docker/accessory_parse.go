package docker

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseAccessoryVolumeMount(input string) (AccessoryMount, error) {
	parts := strings.Split(input, ":")
	if len(parts) < 2 {
		return AccessoryMount{}, fmt.Errorf("invalid volume mount %q", input)
	}
	mount := AccessoryMount{
		Type:   AccessoryMountVolume,
		Name:   parts[0],
		Target: parts[1],
	}
	if len(parts) > 2 {
		mount.ReadOnly = parts[2] == "ro"
	}
	return mount, nil
}

func ParseAccessoryBindMount(input string) (AccessoryMount, error) {
	parts := strings.Split(input, ":")
	if len(parts) < 2 {
		return AccessoryMount{}, fmt.Errorf("invalid bind mount %q", input)
	}
	mount := AccessoryMount{
		Type:   AccessoryMountBind,
		Source: parts[0],
		Target: parts[1],
	}
	if len(parts) > 2 {
		mount.ReadOnly = parts[2] == "ro"
	}
	return mount, nil
}

func ParseAccessoryPortBinding(input string) (AccessoryPortBinding, error) {
	parts := strings.Split(input, ":")
	switch len(parts) {
	case 2:
		containerPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return AccessoryPortBinding{}, fmt.Errorf("invalid publish port %q", input)
		}
		hostPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return AccessoryPortBinding{}, fmt.Errorf("invalid publish port %q", input)
		}
		return AccessoryPortBinding{HostPort: hostPort, ContainerPort: containerPort}, nil
	case 3:
		containerPort, err := strconv.Atoi(parts[2])
		if err != nil {
			return AccessoryPortBinding{}, fmt.Errorf("invalid publish port %q", input)
		}
		hostPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return AccessoryPortBinding{}, fmt.Errorf("invalid publish port %q", input)
		}
		return AccessoryPortBinding{HostIP: parts[0], HostPort: hostPort, ContainerPort: containerPort}, nil
	default:
		return AccessoryPortBinding{}, fmt.Errorf("invalid publish port %q", input)
	}
}

func FormatAccessoryMount(mount AccessoryMount) string {
	base := mount.Name
	if mount.Type == AccessoryMountBind {
		base = mount.Source
	}
	value := base + ":" + mount.Target
	if mount.ReadOnly {
		value += ":ro"
	}
	return value
}

func FormatAccessoryPortBinding(binding AccessoryPortBinding) string {
	if binding.HostIP != "" {
		return fmt.Sprintf("%s:%d:%d", binding.HostIP, binding.HostPort, binding.ContainerPort)
	}
	return fmt.Sprintf("%d:%d", binding.HostPort, binding.ContainerPort)
}

func MergeAccessoryMounts(base, overrides []AccessoryMount) []AccessoryMount {
	merged := append([]AccessoryMount(nil), base...)

	for _, override := range overrides {
		replaced := false
		for i, existing := range merged {
			if existing.Target != override.Target {
				continue
			}
			merged[i] = override
			replaced = true
			break
		}
		if !replaced {
			merged = append(merged, override)
		}
	}

	return merged
}
