package core

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"mcmodlist/core/instance"
	"mcmodlist/core/model"
)

// ListInstances returns Minecraft instances found under known launcher
// directories for the current user and OS.
func ListInstances() ([]model.Instance, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return ListInstancesUnder(home), nil
}

// ListInstancesUnder searches launcher directories below home. It is
// exported for tests and for callers that manage their own home directory.
func ListInstancesUnder(home string) []model.Instance {
	seen := make(map[string]bool)
	var out []model.Instance
	for _, root := range launcherRoots(home) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if seen[path] {
				continue
			}
			det, err := instance.Detect(path)
			if err != nil {
				continue
			}
			seen[path] = true
			out = append(out, det.Info)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func launcherRoots(home string) []string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(appData, "PrismLauncher", "instances"),
			filepath.Join(appData, "MultiMC", "instances"),
			filepath.Join(home, "CurseForge", "Minecraft", "Instances"),
			filepath.Join(appData, "Modrinth App", "profiles"),
			filepath.Join(appData, "gdlauncher_carbon", "instances"),
			filepath.Join(home, "ATLauncher", "instances"),
		}
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		return []string{
			filepath.Join(base, "PrismLauncher", "instances"),
			filepath.Join(base, "multimc", "instances"),
			filepath.Join(home, "CurseForge", "Minecraft", "Instances"),
			filepath.Join(base, "Modrinth App", "profiles"),
			filepath.Join(base, "gdlauncher_carbon", "instances"),
			filepath.Join(home, "ATLauncher", "instances"),
		}
	default:
		data := filepath.Join(home, ".local", "share")
		return []string{
			filepath.Join(data, "PrismLauncher", "instances"),
			filepath.Join(data, "multimc", "instances"),
			filepath.Join(home, "CurseForge", "Minecraft", "Instances"),
			filepath.Join(data, "Modrinth App", "profiles"),
			filepath.Join(data, "gdlauncher_carbon", "instances"),
			filepath.Join(home, "ATLauncher", "instances"),
		}
	}
}
