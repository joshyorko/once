package ui

import (
	"strings"

	"github.com/basecamp/once/internal/docker"
)

func accessoryCommandField(command []string) string {
	return strings.Join(command, " ")
}

func parseAccessoryCommandField(input string) []string {
	return strings.Fields(strings.TrimSpace(input))
}

func accessoryMountField(mounts []docker.AccessoryMount, mountType docker.AccessoryMountType) string {
	values := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		if mount.Type != mountType {
			continue
		}
		values = append(values, docker.FormatAccessoryMount(mount))
	}
	return strings.Join(values, ", ")
}

func parseAccessoryMountField(input string, mountType docker.AccessoryMountType) ([]docker.AccessoryMount, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	items := splitAccessoryFieldList(input)
	mounts := make([]docker.AccessoryMount, 0, len(items))
	for _, item := range items {
		var (
			mount docker.AccessoryMount
			err   error
		)
		switch mountType {
		case docker.AccessoryMountBind:
			mount, err = docker.ParseAccessoryBindMount(item)
		default:
			mount, err = docker.ParseAccessoryVolumeMount(item)
		}
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
}

func accessoryPortField(ports []docker.AccessoryPortBinding) string {
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		values = append(values, docker.FormatAccessoryPortBinding(port))
	}
	return strings.Join(values, ", ")
}

func parseAccessoryPortField(input string) ([]docker.AccessoryPortBinding, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	items := splitAccessoryFieldList(input)
	ports := make([]docker.AccessoryPortBinding, 0, len(items))
	for _, item := range items {
		port, err := docker.ParseAccessoryPortBinding(item)
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}
	return ports, nil
}

func splitAccessoryFieldList(input string) []string {
	parts := strings.Split(input, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}
