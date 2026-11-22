package core

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mcmodlist/core/model"
)

// Check returns deterministic diagnostics for a scanned report.
func Check(report Report) []model.Issue {
	var issues []model.Issue
	issues = append(issues, duplicateIDIssues(report.Mods)...)
	issues = append(issues, loaderMismatchIssues(report)...)
	issues = append(issues, dependencyIssues(report.Mods)...)
	issues = append(issues, minecraftIssues(report)...)
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Type != issues[j].Type {
			return issues[i].Type < issues[j].Type
		}
		if issues[i].Mod != issues[j].Mod {
			return issues[i].Mod < issues[j].Mod
		}
		return issues[i].Message < issues[j].Message
	})
	return issues
}

func checkMods(mods []model.Mod) []model.Mod {
	out := make([]model.Mod, 0, len(mods))
	for _, mod := range mods {
		if mod.Recognized && mod.Status != model.StatusNested {
			out = append(out, mod)
		}
	}
	return out
}

func activeMods(mods []model.Mod) []model.Mod {
	out := make([]model.Mod, 0, len(mods))
	for _, mod := range mods {
		if mod.Recognized && mod.Status == model.StatusEnabled {
			out = append(out, mod)
		}
	}
	return out
}

func duplicateIDIssues(mods []model.Mod) []model.Issue {
	type entry struct {
		id   string
		jars map[string]bool
	}
	byID := make(map[string]*entry)
	for _, mod := range checkMods(mods) {
		id := strings.TrimSpace(mod.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		e := byID[key]
		if e == nil {
			e = &entry{id: id, jars: make(map[string]bool)}
			byID[key] = e
		}
		if mod.Jar != "" {
			e.jars[mod.Jar] = true
		}
	}
	var issues []model.Issue
	for _, e := range byID {
		if len(e.jars) < 2 {
			continue
		}
		jars := make([]string, 0, len(e.jars))
		for jar := range e.jars {
			jars = append(jars, jar)
		}
		sort.Strings(jars)
		issues = append(issues, model.Issue{
			Type:    "duplicate-id",
			Mod:     e.id,
			Message: fmt.Sprintf("duplicate mod id %q in %s", e.id, strings.Join(jars, ", ")),
		})
	}
	return issues
}

func loaderMismatchIssues(report Report) []model.Issue {
	if !report.Instance.Loader.Valid() {
		return nil
	}
	var issues []model.Issue
	for _, mod := range activeMods(report.Mods) {
		if !mod.Loader.Valid() || mod.Loader == report.Instance.Loader {
			continue
		}
		name := firstNonEmptyLocal(mod.ID, mod.Name, mod.Jar)
		issues = append(issues, model.Issue{
			Type:    "loader-mismatch",
			Mod:     name,
			Message: fmt.Sprintf("mod loader %s does not match instance loader %s", mod.Loader, report.Instance.Loader),
		})
	}
	return issues
}

func firstNonEmptyLocal(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func dependencyIssues(mods []model.Mod) []model.Issue {
	active := activeMods(mods)
	present := make(map[string]bool, len(active))
	for _, mod := range active {
		if id := strings.ToLower(strings.TrimSpace(mod.ID)); id != "" {
			present[id] = true
		}
	}
	var issues []model.Issue
	for _, mod := range active {
		for _, dep := range mod.Depends {
			dep = strings.TrimSpace(dep)
			if dep == "" || present[strings.ToLower(dep)] {
				continue
			}
			name := firstNonEmptyLocal(mod.ID, mod.Name, mod.Jar)
			issues = append(issues, model.Issue{
				Type:    "missing-dependency",
				Mod:     name,
				Message: fmt.Sprintf("missing required dependency %q", dep),
			})
		}
	}
	return issues
}

func minecraftIssues(report Report) []model.Issue {
	if strings.TrimSpace(report.Instance.Minecraft) == "" {
		return nil
	}
	var issues []model.Issue
	for _, mod := range activeMods(report.Mods) {
		constraint := strings.TrimSpace(mod.Minecraft)
		if constraint == "" || minecraftSatisfies(constraint, report.Instance.Minecraft) {
			continue
		}
		name := firstNonEmptyLocal(mod.ID, mod.Name, mod.Jar)
		issues = append(issues, model.Issue{
			Type:    "minecraft-version",
			Mod:     name,
			Message: fmt.Sprintf("requires Minecraft %s but instance is %s", constraint, report.Instance.Minecraft),
		})
	}
	return issues
}

func minecraftSatisfies(constraint, version string) bool {
	constraint = strings.TrimSpace(constraint)
	version = strings.TrimSpace(version)
	if constraint == "" || version == "" {
		return true
	}
	for _, alternative := range strings.Split(constraint, "||") {
		if minecraftAlternativeSatisfies(strings.TrimSpace(alternative), version) {
			return true
		}
	}
	return false
}

func minecraftAlternativeSatisfies(alternative, version string) bool {
	if alternative == "" {
		return true
	}
	fields := strings.FieldsFunc(alternative, func(r rune) bool { return r == ',' || r == ' ' })
	for _, term := range fields {
		if !minecraftTermSatisfies(term, version) {
			return false
		}
	}
	return true
}

func minecraftTermSatisfies(term, version string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return true
	}
	for _, op := range []string{"==", ">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(term, op) {
			want := strings.TrimSpace(strings.TrimPrefix(term, op))
			cmp := compareVersions(version, want)
			switch op {
			case ">=":
				return cmp >= 0
			case "<=":
				return cmp <= 0
			case ">":
				return cmp > 0
			case "<":
				return cmp < 0
			default:
				return cmp == 0
			}
		}
	}
	if strings.ContainsAny(term, "xX*") {
		prefix := strings.TrimRight(strings.NewReplacer("x", ".", "X", ".", "*", ".").Replace(term), ".")
		return strings.HasPrefix(version, prefix)
	}
	return compareVersions(version, term) == 0
}

func compareVersions(a, b string) int {
	av := versionParts(a)
	bv := versionParts(b)
	for i := 0; i < len(av) || i < len(bv); i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func versionParts(s string) []int {
	s = strings.TrimSpace(s)
	if idx := strings.IndexAny(s, "-+"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
