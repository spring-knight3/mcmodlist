package jar

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"mcmodlist/core/model"
)

type tomlModsFile struct {
	Mods         []tomlMod                   `toml:"mods"`
	Dependencies map[string][]tomlDependency `toml:"dependencies"`
}

type tomlDependency struct {
	ModID        string `toml:"modId"`
	Type         string `toml:"type"`
	VersionRange string `toml:"versionRange"`
	Mandatory    bool   `toml:"mandatory"`
}

type tomlMod struct {
	ModID           string `toml:"modId"`
	ModIDLower      string `toml:"modid"`
	ModIDCaps       string `toml:"modID"`
	Name            string `toml:"name"`
	DisplayName     string `toml:"displayName"`
	Version         any    `toml:"version"`
	Authors         any    `toml:"authors"`
	Side            string `toml:"side"`
	Description     string `toml:"description"`
	License         string `toml:"license"`
	DisplayURL      string `toml:"displayURL"`
	IssueTrackerURL string `toml:"issueTrackerURL"`
	UpdateJSONURL   string `toml:"updateJSONURL"`
}

func parseTOMLMods(data []byte, loader model.Loader) []model.Mod {
	var file tomlModsFile
	if err := toml.NewDecoder(bytes.NewReader(data)).Decode(&file); err != nil {
		return nil
	}
	deps, minecraft := tomlDependencies(file.Dependencies)
	out := make([]model.Mod, 0, len(file.Mods))
	for _, mod := range file.Mods {
		id := firstNonEmpty(mod.ModID, mod.ModIDLower, mod.ModIDCaps)
		out = append(out, model.Mod{
			Name:        firstNonEmpty(mod.DisplayName, mod.Name, id),
			ID:          id,
			Version:     scalarTOML(mod.Version),
			Authors:     normalizeAuthors(mod.Authors),
			Loader:      loader,
			Side:        mod.Side,
			Depends:     deps,
			Minecraft:   minecraft,
			Description: mod.Description,
			License:     mod.License,
			Links:       tomlLinks(mod),
		})
	}
	return out
}

func tomlLinks(mod tomlMod) map[string]string {
	links := map[string]string{}
	if mod.DisplayURL != "" {
		links["displayURL"] = mod.DisplayURL
	}
	if mod.IssueTrackerURL != "" {
		links["issueTrackerURL"] = mod.IssueTrackerURL
	}
	if mod.UpdateJSONURL != "" {
		links["updateJSONURL"] = mod.UpdateJSONURL
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

func tomlDependencies(in map[string][]tomlDependency) ([]string, string) {
	var deps []string
	minecraft := ""
	seen := make(map[string]bool)
	for key, entries := range in {
		for _, dep := range entries {
			id := firstNonEmpty(dep.ModID, key)
			lower := strings.ToLower(id)
			if lower == "minecraft" {
				if minecraft == "" {
					minecraft = dep.VersionRange
				}
				continue
			}
			if lower == "java" || lower == "forge" || lower == "fml" || strings.HasSuffix(lower, "loader") || lower == "loader" || lower == "neoforged" || lower == "neoforge" {
				continue
			}
			if !dep.Mandatory && dep.Type != "" && !strings.EqualFold(dep.Type, "required") {
				continue
			}
			if id == "" || seen[lower] {
				continue
			}
			seen[lower] = true
			deps = append(deps, id)
		}
	}
	return deps, minecraft
}

func scalarTOML(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}
