// Package engine implements the mcmodlist GUI engine protocol.
package engine

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

// InstanceOption is one selectable instance for the start screen.
type InstanceOption struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Launcher      string `json:"launcher"`
	Minecraft     string `json:"minecraft"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loaderVersion"`
}

// ModRow is one table row in the GUI.
type ModRow struct {
	RowID       string            `json:"rowId"`
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Authors     string            `json:"authors"`
	Jar         string            `json:"jar"`
	Loader      string            `json:"loader"`
	Side        string            `json:"side"`
	Status      string            `json:"status"`
	OldVersion  string            `json:"oldVersion,omitempty"`
	Description string            `json:"description"`
	License     string            `json:"license"`
	Links       map[string]string `json:"links"`
	Group       string            `json:"group,omitempty"`
	GroupTitle  string            `json:"groupTitle,omitempty"`
}

// LoaderCount is one loader summary entry.
type LoaderCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Group is one group boundary in a grouped view.
type Group struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Start int    `json:"start"`
	Count int    `json:"count"`
}

// State is the GUI view state for one scanned instance.
type State struct {
	Instance      InstanceOption  `json:"instance"`
	Rows          []ModRow        `json:"rows"`
	Groups        []Group         `json:"groups,omitempty"`
	Loaders       []LoaderCount   `json:"loaders"`
	LoaderOptions []string        `json:"loaderOptions"`
	Total         int             `json:"total"`
	Disabled      int             `json:"disabled"`
	Warnings      []model.Warning `json:"warnings"`
	Issues        []model.Issue   `json:"issues"`
}

// Details is the details-pane payload for one mod.
type Details struct {
	Mod      ModRow          `json:"mod"`
	Warnings []model.Warning `json:"warnings"`
}

// ViewOptions controls GUI filtering and sorting.
type ViewOptions struct {
	Search           string
	Loader           string
	Side             string
	Author           string
	Sort             string
	GroupBy          string
	ShowDisabled     bool
	ShowNested       bool
	UnrecognisedOnly bool
}

// ListInstances returns instances known to installed launchers.
func ListInstances() ([]InstanceOption, error) {
	instances, err := core.ListInstances()
	if err != nil {
		return nil, err
	}
	out := make([]InstanceOption, 0, len(instances))
	for _, inst := range instances {
		out = append(out, instanceOption(inst))
	}
	return out, nil
}

// ScanOptions returns the core options used by the GUI.
func ScanOptions(progress func(done, total int)) core.Options {
	return core.Options{IncludeNested: true, Progress: progress}
}

// BuildState converts a core report into GUI state.
func BuildState(report core.Report, opts ViewOptions) State {
	mods := core.FilterMods(report.Mods, core.FilterOptions{
		Search:           opts.Search,
		Loader:           opts.Loader,
		Side:             opts.Side,
		Author:           opts.Author,
		UnrecognisedOnly: opts.UnrecognisedOnly,
		IncludeDisabled:  opts.ShowDisabled,
	})
	if !opts.ShowNested {
		filtered := mods[:0]
		for _, mod := range mods {
			if mod.Status != model.StatusNested {
				filtered = append(filtered, mod)
			}
		}
		mods = filtered
	}
	if strings.TrimSpace(opts.Sort) != "" {
		if sorted, err := core.SortMods(mods, opts.Sort); err == nil {
			mods = sorted
		}
	}
	rows := make([]ModRow, 0, len(mods))
	for _, mod := range mods {
		rows = append(rows, rowFromMod(mod))
	}
	counts := core.CountMods(mods)
	return State{
		Instance:      instanceOption(report.Instance),
		Rows:          rows,
		Loaders:       loaderCounts(counts),
		LoaderOptions: loaderFilterOptions(report.Mods),
		Total:         counts.Total,
		Disabled:      counts.Disabled,
		Warnings:      report.Warnings,
		Issues:        report.Issues,
	}
}

// BuildViewReport applies the same filters/sort as BuildState and returns the
// underlying core report for export and diagnostics.
func BuildViewReport(report core.Report, opts ViewOptions) (core.Report, error) {
	view := report
	if !opts.ShowNested {
		mods := make([]model.Mod, 0, len(report.Mods))
		for _, mod := range report.Mods {
			if mod.Status != model.StatusNested {
				mods = append(mods, mod)
			}
		}
		view.Mods = mods
		view.Counts = core.CountMods(mods)
	}
	view = core.FilterReport(view, core.FilterOptions{
		Search:           opts.Search,
		Loader:           opts.Loader,
		Side:             opts.Side,
		Author:           opts.Author,
		UnrecognisedOnly: opts.UnrecognisedOnly,
		IncludeDisabled:  opts.ShowDisabled,
	})
	if strings.TrimSpace(opts.Sort) != "" {
		sorted, err := core.SortMods(view.Mods, opts.Sort)
		if err != nil {
			return core.Report{}, err
		}
		view.Mods = sorted
		view.Counts = core.CountMods(view.Mods)
	}
	return view, nil
}

// CheckReport returns diagnostics for a scanned report.
func CheckReport(report core.Report) []model.Issue {
	return core.Check(report)
}

// DiffReports returns the added/removed/version-changed view between two reports.
func DiffReports(a, b core.Report) core.Report {
	return core.Diff(a, b)
}

// DetailsForMod returns details for one mod in a report.
func DetailsForMod(report core.Report, id, jar string) Details {
	for _, mod := range report.Mods {
		if mod.ID == id && mod.Jar == jar {
			return Details{Mod: rowFromMod(mod), Warnings: warningsForJar(report.Warnings, jar)}
		}
	}
	return Details{}
}

// ExportState renders state in a simple GUI export format.
func ExportState(state State, format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	case "csv":
		return exportCSV(state), nil
	case "markdown", "md":
		return exportMarkdown(state), nil
	case "text":
		return exportText(state), nil
	case "html":
		return exportHTML(state), nil
	default:
		return "", fmt.Errorf("unknown export format %q", format)
	}
}

func instanceOption(inst model.Instance) InstanceOption {
	return InstanceOption{
		Name:          inst.Name,
		Path:          inst.Path,
		Launcher:      inst.Launcher,
		Minecraft:     inst.Minecraft,
		Loader:        string(inst.Loader),
		LoaderVersion: inst.LoaderVersion,
	}
}

func rowFromMod(mod model.Mod) ModRow {
	return ModRow{
		RowID:       rowID(mod),
		Name:        mod.Name,
		ID:          mod.ID,
		Version:     mod.Version,
		Authors:     strings.Join(mod.Authors, ", "),
		Jar:         mod.Jar,
		Loader:      loaderLabel(mod),
		Side:        mod.Side,
		Status:      statusLabel(mod),
		OldVersion:  mod.OldVersion,
		Description: mod.Description,
		License:     mod.License,
		Links:       mod.Links,
	}
}

func loaderLabel(mod model.Mod) string {
	if mod.Layer != "" {
		return string(mod.Loader) + "/" + mod.Layer
	}
	return string(mod.Loader)
}

func statusLabel(mod model.Mod) string {
	if mod.Change != "" {
		return mod.Change
	}
	if !mod.Recognized {
		return "unrecognised"
	}
	return string(mod.Status)
}

// LoaderLabel returns the display loader label for a mod.
func LoaderLabel(mod model.Mod) string { return loaderLabel(mod) }

// StatusLabel returns the display status label for a mod.
func StatusLabel(mod model.Mod) string { return statusLabel(mod) }

func rowID(mod model.Mod) string {
	return strings.Join([]string{
		mod.Jar, mod.ID, mod.Name, mod.Version, string(mod.Loader), mod.Layer,
		mod.Side, string(mod.Status), mod.Change, mod.OldVersion, fmt.Sprint(mod.Recognized),
	}, "\x1f")
}

func loaderCounts(counts model.Counts) []LoaderCount {
	labels := core.OrderedLoaderLabels(counts)
	out := make([]LoaderCount, 0, len(labels))
	for _, label := range labels {
		out = append(out, LoaderCount{Label: label, Count: counts.Loaders[label]})
	}
	return out
}

func loaderFilterOptions(mods []model.Mod) []string {
	counts := model.NewCounts()
	for _, mod := range mods {
		if !mod.Recognized {
			continue
		}
		label := string(mod.Loader)
		if mod.Layer != "" {
			label += "/" + mod.Layer
		}
		if label != "" {
			counts.Loaders[label]++
		}
	}
	return core.OrderedLoaderLabels(counts)
}

func warningsForJar(warnings []model.Warning, jar string) []model.Warning {
	var out []model.Warning
	for _, warning := range warnings {
		if warning.Mod == jar || warning.Mod == "" {
			out = append(out, warning)
		}
	}
	return out
}

func exportCSV(state State) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"Name", "ID", "Version", "Authors", "Jar", "Loader", "Side", "Status"})
	for _, row := range state.Rows {
		_ = w.Write([]string{row.Name, row.ID, row.Version, row.Authors, row.Jar, row.Loader, row.Side, row.Status})
	}
	w.Flush()
	return b.String()
}

func exportMarkdown(state State) string {
	var b strings.Builder
	b.WriteString("| Name | ID | Version | Authors | Jar | Loader | Side | Status |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, row := range state.Rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			row.Name, row.ID, row.Version, row.Authors, row.Jar, row.Loader, row.Side, row.Status)
	}
	return b.String()
}

func exportText(state State) string {
	var b strings.Builder
	for _, row := range state.Rows {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Name, row.ID, row.Version, row.Authors, row.Jar, row.Loader, row.Side, row.Status)
	}
	return b.String()
}

func exportHTML(state State) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>mcmodlist</title></head><body><table><thead><tr>")
	for _, h := range []string{"Name", "ID", "Version", "Authors", "Jar", "Loader", "Side", "Status"} {
		b.WriteString("<th>" + h + "</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range state.Rows {
		b.WriteString("<tr>")
		for _, cell := range []string{row.Name, row.ID, row.Version, row.Authors, row.Jar, row.Loader, row.Side, row.Status} {
			b.WriteString("<td>" + htmlEscape(cell) + "</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>\n")
	return b.String()
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}
