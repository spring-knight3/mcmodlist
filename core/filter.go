package core

import (
	"strings"

	"mcmodlist/core/model"
)

// FilterOptions selects which mods are included in a report view.
type FilterOptions struct {
	Search           string
	Loader           string
	Side             string
	Author           string
	UnrecognisedOnly bool
	IncludeDisabled  bool
}

// FilterReport returns a copy of report containing only matching mods, with
// counts recomputed for the filtered list.
func FilterReport(report Report, f FilterOptions) Report {
	out := report
	out.Mods = FilterMods(report.Mods, f)
	out.Counts = CountMods(out.Mods)
	return out
}

// FilterMods returns the subset of mods matching f.
func FilterMods(mods []model.Mod, f FilterOptions) []model.Mod {
	out := make([]model.Mod, 0, len(mods))
	for _, mod := range mods {
		if !f.IncludeDisabled && mod.Status == model.StatusDisabled {
			continue
		}
		if f.UnrecognisedOnly && mod.Recognized {
			continue
		}
		if !matchesSearch(mod, f.Search) || !matchesLoader(mod, f.Loader) || !matchesSide(mod, f.Side) || !matchesAuthor(mod, f.Author) {
			continue
		}
		out = append(out, mod)
	}
	return out
}

func loaderLabel(mod model.Mod) string {
	label := string(mod.Loader)
	if mod.Layer != "" {
		label += "/" + mod.Layer
	}
	return label
}

func matchesSearch(mod model.Mod, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	fields := []string{mod.Name, mod.ID, mod.Version, mod.Jar, loaderLabel(mod), mod.Side, strings.Join(mod.Authors, " ")}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

func matchesLoader(mod model.Mod, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" {
		return true
	}
	label := strings.ToLower(loaderLabel(mod))
	base := strings.ToLower(string(mod.Loader))
	layer := strings.ToLower(mod.Layer)
	return q == label || q == base || q == layer || strings.HasPrefix(label, q+"/")
}

func matchesSide(mod model.Mod, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(mod.Side), q)
}

func matchesAuthor(mod model.Mod, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" {
		return true
	}
	for _, author := range mod.Authors {
		if strings.Contains(strings.ToLower(author), q) {
			return true
		}
	}
	return false
}
