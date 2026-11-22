package core

import (
	"fmt"
	"sort"
	"strings"

	"mcmodlist/core/model"
)

var sortFields = map[string]bool{
	"name":    true,
	"id":      true,
	"version": true,
	"author":  true,
	"authors": true,
	"jar":     true,
	"loader":  true,
	"side":    true,
	"status":  true,
}

// SortMods returns mods ordered by a comma-separated spec. A leading '-'
// reverses that key. An empty spec preserves the incoming order.
func SortMods(mods []model.Mod, spec string) ([]model.Mod, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return mods, nil
	}
	type key struct {
		field string
		desc  bool
	}
	var keys []key
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := strings.HasPrefix(part, "-")
		field := strings.ToLower(strings.TrimPrefix(part, "-"))
		if !sortFields[field] {
			return nil, fmt.Errorf("unknown sort key %q", field)
		}
		keys = append(keys, key{field: field, desc: desc})
	}
	if len(keys) == 0 {
		return mods, nil
	}
	out := append([]model.Mod(nil), mods...)
	sort.SliceStable(out, func(i, j int) bool {
		for _, k := range keys {
			cmp := compareModField(out[i], out[j], k.field)
			if cmp == 0 {
				continue
			}
			if k.desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	return out, nil
}

func compareModField(a, b model.Mod, field string) int {
	av, bv := sortFieldValue(a, field), sortFieldValue(b, field)
	switch {
	case av < bv:
		return -1
	case av > bv:
		return 1
	default:
		return 0
	}
}

func sortFieldValue(mod model.Mod, field string) string {
	switch field {
	case "name":
		return strings.ToLower(mod.Name)
	case "id":
		return strings.ToLower(mod.ID)
	case "version":
		return strings.ToLower(mod.Version)
	case "author", "authors":
		return strings.ToLower(strings.Join(mod.Authors, ", "))
	case "jar":
		return strings.ToLower(mod.Jar)
	case "loader":
		return strings.ToLower(loaderLabel(mod))
	case "side":
		return strings.ToLower(mod.Side)
	case "status":
		return strings.ToLower(string(mod.Status))
	default:
		return ""
	}
}
