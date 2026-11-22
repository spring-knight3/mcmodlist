package jar

import (
	"archive/zip"
	"encoding/json"
	"path"
	"strconv"
	"strings"

	"mcmodlist/core/model"
)

func parseMetadata(zr *zip.Reader, jarName string, status model.Status) []model.Mod {
	var all []model.Mod
	if data, ok := readFile(zr, "META-INF/neoforge.mods.toml"); ok {
		all = append(all, parseTOMLMods(data, model.LoaderNeoForge)...)
	}
	if data, ok := readFile(zr, "META-INF/mods.toml"); ok {
		all = append(all, parseTOMLMods(data, model.LoaderForge)...)
	}
	if data, ok := readFile(zr, "mcmod.info"); ok {
		all = append(all, parseMCModInfo(data)...)
	}
	if data, ok := readFile(zr, "META-INF/mcmod.info"); ok {
		all = append(all, parseMCModInfo(data)...)
	}
	if data, ok := readFile(zr, "fabric.mod.json"); ok {
		all = append(all, parseFabricModJSON(data)...)
	}
	if data, ok := readFile(zr, "quilt.mod.json"); ok {
		all = append(all, parseQuiltModJSON(data)...)
	}
	return finalizeCandidates(all, jarName, status)
}

func finalizeCandidates(all []model.Mod, jarName string, status model.Status) []model.Mod {
	seen := make(map[string]struct{})
	out := make([]model.Mod, 0, len(all))
	for _, mod := range all {
		mod.Jar = jarName
		mod.Status = status
		mod.Recognized = true
		if mod.ID == "" {
			mod.ID = mod.Name
		}
		if mod.Name == "" {
			mod.Name = mod.ID
		}
		if mod.Name == "" && mod.ID == "" {
			mod.Name = jarDisplayName(jarName)
			mod.ID = mod.Name
		}
		key := string(mod.Loader) + "|" + mod.ID
		if mod.ID == "" {
			key = string(mod.Loader) + "|name|" + mod.Name
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, mod)
	}
	return out
}

func jarDisplayName(jarName string) string {
	base := path.Base(strings.ReplaceAll(jarName, "\\", "/"))
	base = strings.TrimSuffix(base, ".disabled")
	base = strings.TrimSuffix(base, ".jar")
	if base == "" {
		return jarName
	}
	return base
}

type jsonModMetadata struct {
	ID          string `json:"id"`
	ModID       string `json:"modid"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	Authors     any    `json:"authors"`
	AuthorList  any    `json:"authorList"`
	AuthorsStr  string `json:"author"`
	Description string `json:"description"`
	Side        string `json:"side"`
	Depends     any    `json:"depends"`
	License     string `json:"license"`
	Contact     any    `json:"contact"`
}

func parseFabricModJSON(data []byte) []model.Mod {
	var meta jsonModMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return []model.Mod{jsonModToModel(meta, model.LoaderFabric)}
}

func parseQuiltModJSON(data []byte) []model.Mod {
	var meta jsonModMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return []model.Mod{jsonModToModel(meta, model.LoaderQuilt)}
}

func jsonModToModel(meta jsonModMetadata, loader model.Loader) model.Mod {
	authorValue := meta.Authors
	if authorValue == nil {
		authorValue = meta.AuthorList
	}
	if authorValue == nil && meta.AuthorsStr != "" {
		authorValue = meta.AuthorsStr
	}
	depends, minecraft := dependsFromAny(meta.Depends)
	return model.Mod{
		Name:        firstNonEmpty(meta.Name, meta.DisplayName, meta.ID, meta.ModID),
		ID:          firstNonEmpty(meta.ID, meta.ModID),
		Version:     meta.Version,
		Authors:     normalizeAuthors(authorValue),
		Loader:      loader,
		Side:        meta.Side,
		Depends:     depends,
		Minecraft:   minecraft,
		Description: meta.Description,
		License:     meta.License,
		Links:       linksFromAny(meta.Contact),
	}
}

func parseMCModInfo(data []byte) []model.Mod {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	items := mcmodItems(raw)
	out := make([]model.Mod, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		authorValue := m["authorList"]
		if authorValue == nil {
			authorValue = m["authors"]
		}
		if authorValue == nil {
			authorValue = m["author"]
		}
		links := map[string]string{}
		if s := scalarJSON(m["url"]); s != "" {
			links["url"] = s
		}
		if s := scalarJSON(m["updateUrl"]); s != "" {
			links["updateUrl"] = s
		}
		if len(links) == 0 {
			links = nil
		}
		out = append(out, model.Mod{
			Name:        firstNonEmpty(scalarJSON(m["name"]), scalarJSON(m["displayName"])),
			ID:          firstNonEmpty(scalarJSON(m["modid"]), scalarJSON(m["modID"]), scalarJSON(m["id"])),
			Version:     scalarJSON(m["version"]),
			Authors:     normalizeAuthors(authorValue),
			Loader:      model.LoaderForge,
			Side:        scalarJSON(m["side"]),
			Description: scalarJSON(m["description"]),
			License:     scalarJSON(m["license"]),
			Links:       links,
		})
	}
	return out
}

func mcmodItems(raw any) []any {
	switch v := raw.(type) {
	case []any:
		return v
	case map[string]any:
		if mods, ok := v["mods"].([]any); ok {
			return mods
		}
		return []any{v}
	default:
		return nil
	}
}

func scalarJSON(v any) string {
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

func linksFromAny(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for key, value := range m {
		if s := scalarJSON(value); s != "" {
			out[key] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func dependsFromAny(v any) ([]string, string) {
	var deps []string
	minecraft := ""
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		lower := strings.ToLower(id)
		if lower == "minecraft" {
			return
		}
		for _, existing := range deps {
			if strings.EqualFold(existing, id) {
				return
			}
		}
		deps = append(deps, id)
	}
	switch t := v.(type) {
	case map[string]any:
		for key, value := range t {
			lower := strings.ToLower(key)
			if lower == "minecraft" {
				minecraft = scalarJSON(value)
				continue
			}
			if lower == "java" || strings.HasSuffix(lower, "loader") || lower == "loader" {
				continue
			}
			if scalarJSON(value) != "" {
				add(key)
			}
		}
	case []any:
		for _, item := range t {
			add(scalarJSON(item))
		}
	case string:
		for _, item := range strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ' ' }) {
			add(item)
		}
	}
	return deps, minecraft
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
