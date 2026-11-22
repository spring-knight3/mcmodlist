package render

import (
	"sort"
	"strings"
	"unicode/utf8"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

var columns = []string{"Name", "ID", "Version", "Authors", "Jar", "Loader", "Side", "Status"}

func modRow(mod model.Mod) []string {
	return []string{
		nameDisplay(mod),
		mod.ID,
		versionDisplay(mod),
		authorsDisplay(mod.Authors, ", "),
		mod.Jar,
		dashIfEmpty(modLoaderDisplay(mod)),
		dashIfEmpty(mod.Side),
		statusDisplay(mod),
	}
}

func selectedColumnIndexes(opts Options) []int {
	all := make([]int, len(columns))
	for i := range all {
		all[i] = i
	}
	if len(opts.Columns) == 0 {
		return all
	}
	wanted := make(map[string]int, len(columns))
	for i, name := range columns {
		wanted[strings.ToLower(name)] = i
	}
	var out []int
	for _, name := range opts.Columns {
		if idx, ok := wanted[strings.ToLower(strings.TrimSpace(name))]; ok {
			out = append(out, idx)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

func headerRow(opts Options) []string {
	idxs := selectedColumnIndexes(opts)
	out := make([]string, 0, len(idxs))
	for _, idx := range idxs {
		out = append(out, columns[idx])
	}
	return out
}

func modRowSelected(mod model.Mod, opts Options) []string {
	row := modRow(mod)
	idxs := selectedColumnIndexes(opts)
	out := make([]string, 0, len(idxs))
	for _, idx := range idxs {
		cell := row[idx]
		if opts.Truncate > 0 {
			cell = truncate(cell, opts.Truncate)
		}
		out = append(out, cell)
	}
	return out
}

type modGroup struct {
	Title string
	Mods  []model.Mod
}

func groupedMods(report core.Report, opts Options) []modGroup {
	by := strings.ToLower(strings.TrimSpace(opts.GroupBy))
	if by != "loader" && by != "side" {
		return []modGroup{{Mods: report.Mods}}
	}
	mods := append([]model.Mod(nil), report.Mods...)
	sort.SliceStable(mods, func(i, j int) bool {
		ka, kb := groupKey(mods[i], by), groupKey(mods[j], by)
		if ka != kb {
			return ka < kb
		}
		if an, bn := strings.ToLower(mods[i].Name), strings.ToLower(mods[j].Name); an != bn {
			return an < bn
		}
		if mods[i].ID != mods[j].ID {
			return mods[i].ID < mods[j].ID
		}
		return mods[i].Jar < mods[j].Jar
	})
	var groups []modGroup
	for _, mod := range mods {
		key := groupKey(mod, by)
		if len(groups) == 0 || groupKey(groups[len(groups)-1].Mods[0], by) != key {
			groups = append(groups, modGroup{Title: groupTitle(mod, by)})
		}
		groups[len(groups)-1].Mods = append(groups[len(groups)-1].Mods, mod)
	}
	return groups
}

func groupKey(mod model.Mod, by string) string {
	if by == "loader" {
		return strings.ToLower(modLoaderLabel(mod))
	}
	return strings.ToLower(mod.Side)
}

func groupTitle(mod model.Mod, by string) string {
	if by == "loader" {
		label := modLoaderLabel(mod)
		if label == "" {
			return "Loader: -"
		}
		return "Loader: " + displayLabel(label)
	}
	return "Side: " + dashIfEmpty(mod.Side)
}

func capsForSelected(opts Options) []int {
	idxs := selectedColumnIndexes(opts)
	full := defaultCaps()
	if opts.Wide {
		full = make([]int, len(columns))
		for i := range full {
			full[i] = 1000
		}
	} else {
		full = fitCaps(full, effectiveWidth(opts))
	}
	out := make([]int, 0, len(idxs))
	for _, idx := range idxs {
		cap := full[idx]
		if opts.Truncate > 0 && opts.Truncate < cap {
			cap = opts.Truncate
		}
		out = append(out, cap)
	}
	return out
}

func renderPretty(report core.Report, opts Options) string {
	if opts.SummaryOnly {
		return strings.Join(summaryLines(report), "\n") + "\n"
	}
	lines := make([]string, 0)
	if showSummary(opts) {
		summary := summaryLines(report)
		if opts.Color {
			summary = colorLines(summary)
		}
		lines = append(lines, summary...)
		lines = append(lines, "")
	}
	groups := groupedMods(report, opts)
	caps := capsForSelected(opts)
	header := headerRow(opts)
	for gi, group := range groups {
		if group.Title != "" {
			if gi > 0 || len(groups) > 1 {
				lines = append(lines, "")
			}
			lines = append(lines, group.Title)
		}
		if showColumnHeader(opts) {
			lines = append(lines, formatPrettyRow(header, caps))
		}
		for _, mod := range group.Mods {
			lines = append(lines, formatPrettyRow(modRowSelected(mod, opts), caps))
		}
	}
	if note := footerNote(report, opts); note != "" {
		lines = append(lines, "")
		lines = append(lines, note)
	}
	return joinLines(lines)
}

func renderText(report core.Report, opts Options) string {
	if opts.SummaryOnly {
		return strings.Join(summaryLines(report), "\n") + "\n"
	}
	lines := make([]string, 0)
	if showSummary(opts) {
		lines = append(lines, summaryLines(report)...)
		lines = append(lines, "")
	}
	header := strings.Join(headerRow(opts), " | ")
	for gi, group := range groupedMods(report, opts) {
		if group.Title != "" {
			if gi > 0 || len(groupedMods(report, opts)) > 1 {
				lines = append(lines, "")
			}
			lines = append(lines, group.Title)
		}
		if showColumnHeader(opts) {
			lines = append(lines, header)
		}
		for _, mod := range group.Mods {
			lines = append(lines, strings.Join(modRowSelected(mod, opts), " | "))
		}
	}
	if note := footerNote(report, opts); note != "" {
		lines = append(lines, "")
		lines = append(lines, note)
	}
	return joinLines(lines)
}

func renderMarkdown(report core.Report, opts Options) string {
	if opts.SummaryOnly {
		return strings.Join(summaryLines(report), "\n") + "\n"
	}
	lines := make([]string, 0)
	if showSummary(opts) {
		lines = append(lines, summaryLines(report)...)
		lines = append(lines, "")
	}
	if !opts.NoHeader {
		lines = append(lines, "## Mods")
	}
	header := headerRow(opts)
	for _, group := range groupedMods(report, opts) {
		if group.Title != "" {
			lines = append(lines, "", "### "+group.Title)
		}
		if showColumnHeader(opts) {
			lines = append(lines, "| "+strings.Join(header, " | ")+" |")
			lines = append(lines, "|"+strings.Repeat(" --- |", len(header)))
			for _, mod := range group.Mods {
				lines = append(lines, "| "+strings.Join(escapeMarkdownRow(modRowSelected(mod, opts)), " | ")+" |")
			}
		} else {
			for _, mod := range group.Mods {
				lines = append(lines, strings.Join(escapeMarkdownRow(modRowSelected(mod, opts)), " | "))
			}
		}
	}
	if note := footerNote(report, opts); note != "" {
		lines = append(lines, "")
		lines = append(lines, note)
	}
	return joinLines(lines)
}

func escapeMarkdownRow(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		out[i] = strings.ReplaceAll(cell, "|", "\\|")
	}
	return out
}

func defaultCaps() []int {
	return []int{28, 24, 14, 24, 30, 18, 8, 12}
}

func fitCaps(caps []int, width int) []int {
	out := append([]int(nil), caps...)
	if width <= 0 {
		return out
	}
	sep := 2 * (len(out) - 1)
	avail := width - sep
	if avail < len(out) {
		avail = len(out)
	}
	sum := 0
	for _, cap := range out {
		sum += cap
	}
	for sum > avail {
		reduced := false
		for _, i := range []int{3, 4, 0, 1, 5} {
			if i < len(out) && out[i] > 8 {
				out[i]--
				sum--
				reduced = true
				if sum <= avail {
					break
				}
			}
		}
		if !reduced {
			break
		}
	}
	return out
}

func formatPrettyRow(row []string, caps []int) string {
	cells := make([]string, len(row))
	for i, cell := range row {
		max := 0
		if i < len(caps) {
			max = caps[i]
		}
		cells[i] = truncate(cell, max)
	}
	for i := 0; i < len(cells)-1; i++ {
		width := 0
		if i < len(caps) {
			width = caps[i]
		}
		cells[i] = padRight(cells[i], width)
	}
	return strings.Join(cells, "  ")
}

func padRight(s string, width int) string {
	for utf8.RuneCountInString(s) < width {
		s += " "
	}
	return s
}

func colorLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = "\x1b[1m" + line + "\x1b[0m"
	}
	return out
}

func effectiveWidth(opts Options) int {
	if opts.Wide {
		return 10000
	}
	if opts.Width > 0 {
		return opts.Width
	}
	return 100
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
