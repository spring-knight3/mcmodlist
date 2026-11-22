package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mcmodlist/core/model"
)

type launcherParser func(root string) (model.Instance, bool)

var launcherParsers = []launcherParser{
	parsePrismMultiMC,
	parseCurseForge,
	parseModrinth,
	parseATLauncher,
	parseGDLauncher,
}

func readLauncherMetadata(root string) model.Instance {
	for _, parse := range launcherParsers {
		if info, ok := parse(root); ok {
			return info
		}
	}
	return model.Instance{}
}

func parsePrismMultiMC(root string) (model.Instance, bool) {
	cfgPath := filepath.Join(root, "instance.cfg")
	packPath := filepath.Join(root, "mmc-pack.json")
	_, cfgErr := os.Stat(cfgPath)
	_, packErr := os.Stat(packPath)
	if cfgErr != nil && packErr != nil {
		return model.Instance{}, false
	}

	info := model.Instance{Launcher: "multimc"}
	if packErr == nil {
		info.Launcher = "prism"
	}
	if cfgErr == nil {
		if values, err := parseKeyValueFile(cfgPath); err == nil {
			info.Name = values["name"]
		}
	}
	if packErr == nil {
		if pack, err := readJSONFile(packPath); err == nil {
			for _, component := range mmcComponents(pack) {
				uid := strings.ToLower(component.UID)
				version := component.Version
				if loader := loaderFromMMCUID(uid); loader.Valid() {
					info.Loader = loader
					info.LoaderVersion = version
					continue
				}
				if info.Minecraft == "" && (strings.Contains(uid, "net.minecraft") || strings.Contains(uid, "mojang.minecraft")) {
					info.Minecraft = version
				}
			}
		}
	}
	return info, true
}

type mmcComponent struct {
	UID     string `json:"uid"`
	Version string `json:"version"`
}

func mmcComponents(pack map[string]any) []mmcComponent {
	raw, ok := pack["components"].([]any)
	if !ok {
		return nil
	}
	out := make([]mmcComponent, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, mmcComponent{
			UID:     scalarString(m["uid"]),
			Version: scalarString(m["version"]),
		})
	}
	return out
}

func loaderFromMMCUID(uid string) model.Loader {
	switch {
	case strings.Contains(uid, "neoforged") || strings.Contains(uid, "neoforge"):
		return model.LoaderNeoForge
	case strings.Contains(uid, "quilt"):
		return model.LoaderQuilt
	case strings.Contains(uid, "fabric"):
		return model.LoaderFabric
	case strings.Contains(uid, "minecraftforge") || strings.Contains(uid, "forge"):
		return model.LoaderForge
	default:
		return model.LoaderUnknown
	}
}

func parseCurseForge(root string) (model.Instance, bool) {
	path := filepath.Join(root, "minecraftinstance.json")
	if _, err := os.Stat(path); err != nil {
		return model.Instance{}, false
	}
	data, err := readJSONFile(path)
	if err != nil {
		return model.Instance{Launcher: "curseforge"}, true
	}
	return model.Instance{
		Launcher:      "curseforge",
		Name:          stringAt(data, "name", "instanceName", "instance.name"),
		Minecraft:     stringAt(data, "mcGameVersion", "minecraftVersion", "gameVersion", "baseModLoader.minecraftVersion", "manifest.minecraftVersion"),
		Loader:        loaderAt(data, "baseModLoader.name", "modLoader", "loader", "baseModLoader.loader"),
		LoaderVersion: stringAt(data, "baseModLoader.forgeVersion", "baseModLoader.loaderVersion", "loaderVersion", "modLoaderVersion", "baseModLoader.version"),
	}, true
}

func parseModrinth(root string) (model.Instance, bool) {
	path := filepath.Join(root, "profile.json")
	if _, err := os.Stat(path); err != nil {
		return model.Instance{}, false
	}
	data, err := readJSONFile(path)
	if err != nil {
		return model.Instance{Launcher: "modrinth"}, true
	}
	return model.Instance{
		Launcher:      "modrinth",
		Name:          stringAt(data, "name", "instanceName"),
		Minecraft:     stringAt(data, "game_version", "minecraft_version", "gameVersion", "mcVersion"),
		Loader:        loaderAt(data, "loader", "modLoader", "mod_loader"),
		LoaderVersion: stringAt(data, "version_id", "loaderVersion", "loader_version"),
	}, true
}

func parseATLauncher(root string) (model.Instance, bool) {
	path := filepath.Join(root, "instance.json")
	if _, err := os.Stat(path); err != nil {
		return model.Instance{}, false
	}
	data, err := readJSONFile(path)
	if err != nil {
		return model.Instance{Launcher: "atlauncher"}, true
	}
	return model.Instance{
		Launcher:      "atlauncher",
		Name:          stringAt(data, "name", "instanceName"),
		Minecraft:     stringAt(data, "minecraftVersion", "gameVersion", "mcVersion", "version"),
		Loader:        loaderAt(data, "loader", "modLoader", "mod_loader"),
		LoaderVersion: stringAt(data, "loaderVersion", "loader_version", "modLoaderVersion"),
	}, true
}

func parseGDLauncher(root string) (model.Instance, bool) {
	path := filepath.Join(root, "config.json")
	if _, err := os.Stat(path); err != nil {
		return model.Instance{}, false
	}
	data, err := readJSONFile(path)
	if err != nil {
		return model.Instance{Launcher: "gdlauncher"}, true
	}
	return model.Instance{
		Launcher:      "gdlauncher",
		Name:          stringAt(data, "name", "instanceName", "InstanceName"),
		Minecraft:     stringAt(data, "gameVersion", "minecraftVersion", "mcVersion", "version"),
		Loader:        loaderAt(data, "loader", "modLoader", "mod_loader", "Loader"),
		LoaderVersion: stringAt(data, "loaderVersion", "loader_version", "modLoaderVersion", "LoaderVersion"),
	}, true
}

func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(map[string]any)
	}
	return out, nil
}

func parseKeyValueFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func lookup(m map[string]any, path string) any {
	current := any(m)
	for _, part := range strings.Split(path, ".") {
		mm, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = mm[part]
	}
	return current
}

func stringAt(m map[string]any, paths ...string) string {
	for _, path := range paths {
		if s := scalarString(lookup(m, path)); s != "" {
			return s
		}
	}
	return ""
}

func loaderAt(m map[string]any, paths ...string) model.Loader {
	for _, path := range paths {
		v := lookup(m, path)
		if mm, ok := v.(map[string]any); ok {
			name := scalarString(mm["name"])
			if name == "" {
				name = scalarString(mm["id"])
			}
			if loader := model.ParseLoader(name); loader.Valid() {
				return loader
			}
		}
		if loader := model.ParseLoader(scalarString(v)); loader.Valid() {
			return loader
		}
	}
	return model.LoaderUnknown
}

func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}
