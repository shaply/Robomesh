package handler_engine

import (
	"fmt"
	"os"
	"path/filepath"
	"roboserver/shared"
)

// ResolveHandlerScript returns the absolute path to the handler script for a device type.
// It looks for: {base_path}/{deviceType}.sh
func ResolveHandlerScript(deviceType string) (string, error) {
	basePath := shared.AppConfig.Handlers.BasePath
	scriptPath := filepath.Join(basePath, deviceType+".sh")

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
