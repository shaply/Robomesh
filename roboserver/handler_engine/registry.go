package handler_engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"roboserver/shared"
)

// validDeviceType matches only safe device type names: alphanumeric, hyphens, underscores, max 64 chars.
var validDeviceType = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// ResolveHandlerScript returns the absolute path to the handler script for a device type.
// It looks for: {base_path}/{deviceType}/start_handler.sh
func ResolveHandlerScript(deviceType string) (string, error) {
	if !validDeviceType.MatchString(deviceType) {
		return "", fmt.Errorf("invalid device type %q: must be alphanumeric/hyphens/underscores, max 64 chars", deviceType)
	}
	basePath := shared.AppConfig.Handlers.BasePath
	scriptPath := filepath.Join(basePath, deviceType, "start_handler.sh")

	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve handler path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("handler script not found for device type %q: %w", deviceType, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("handler path is a directory, not a script: %s", absPath)
	}

	return absPath, nil
}

// ResolveHandlerDir returns the absolute path to the handler directory for a device type.
func ResolveHandlerDir(deviceType string) (string, error) {
	basePath := shared.AppConfig.Handlers.BasePath
	dirPath := filepath.Join(basePath, deviceType)

	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve handler dir: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("handler directory not found for device type %q: %w", deviceType, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("handler path is not a directory: %s", absPath)
	}

	return absPath, nil
}

// ListHandlerTypes returns all device types that have handler directories.
func ListHandlerTypes() []string {
	basePath := shared.AppConfig.Handlers.BasePath
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	var types []string
	for _, entry := range entries {
		if entry.IsDir() {
			scriptPath := filepath.Join(basePath, entry.Name(), "start_handler.sh")
			if _, err := os.Stat(scriptPath); err == nil {
				types = append(types, entry.Name())
			}
		}
	}
	return types
}
