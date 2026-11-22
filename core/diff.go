package core

import (
	"strings"

	"mcmodlist/core/model"
)

// Diff returns a report describing mods added, removed, or version-changed
// between a (old) and b (new). Changed entries keep the new mod and record the
// old version in OldVersion.
func Diff(a, b Report) Report {
	oldMods := diffableMods(a.Mods)
	newMods := diffableMods(b.Mods)
	oldByKey := make(map[string]model.Mod, len(oldMods))
	for _, mod := range oldMods {
		oldByKey[diffKey(mod)] = mod
	}
	newByKey := make(map[string]model.Mod, len(newMods))
	for _, mod := range newMods {
		newByKey[diffKey(mod)] = mod
	}

	var out []model.Mod
	for key, old := range oldByKey {
		newMod, ok := newByKey[key]
		if !ok {
			removed := old
			removed.Change = "removed"
			out = append(out, removed)
			continue
		}
		if !strings.EqualFold(old.Version, newMod.Version) {
			changed := newMod
			changed.Change = "version-changed"
			changed.OldVersion = old.Version
			out = append(out, changed)
		}
	}
	for key, newMod := range newByKey {
		if _, ok := oldByKey[key]; !ok {
			added := newMod
			added.Change = "added"
			out = append(out, added)
		}
	}
	if sorted, err := SortMods(out, "name,id,jar"); err == nil {
		out = sorted
	}
	return Report{
		Instance: a.Instance,
		Mods:     out,
		Counts:   CountMods(out),
		Issues:   []model.Issue{},
		Warnings: []model.Warning{},
	}
}

func diffableMods(mods []model.Mod) []model.Mod {
	out := make([]model.Mod, 0, len(mods))
	for _, mod := range mods {
		if mod.Recognized && mod.Status != model.StatusNested {
			out = append(out, mod)
		}
	}
	return out
}

func diffKey(mod model.Mod) string {
	if id := strings.ToLower(strings.TrimSpace(mod.ID)); id != "" {
		return "id:" + id
	}
	return "name:" + strings.ToLower(mod.Name) + "|" + mod.Jar
}
