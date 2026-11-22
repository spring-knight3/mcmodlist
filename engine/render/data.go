package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

type jsonTool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type jsonInstance struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Launcher      string `json:"launcher"`
	Minecraft     string `json:"minecraft"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loaderVersion"`
}

type jsonCounts struct {
	Fabric       int `json:"fabric"`
	Quilt        int `json:"quilt"`
	Forge        int `json:"forge"`
	NeoForge     int `json:"neoforge"`
	Connector    int `json:"connector"`
	Total        int `json:"total"`
	Disabled     int `json:"disabled"`
	Unrecognised int `json:"unrecognised"`
}

type jsonMod struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	Authors    []string `json:"authors"`
	Jar        string   `json:"jar"`
	Loader     string   `json:"loader"`
	Side       string   `json:"side"`
	Status     string   `json:"status"`
	Change     string   `json:"change,omitempty"`
	OldVersion string   `json:"oldVersion,omitempty"`
}

type jsonMessage struct {
	Type    string `json:"type"`
	Mod     string `json:"mod"`
	Message string `json:"message"`
}

type jsonDocument struct {
	Tool     jsonTool      `json:"tool"`
	Instance jsonInstance  `json:"instance"`
	Counts   jsonCounts    `json:"counts"`
	Mods     []jsonMod     `json:"mods"`
	Issues   []jsonMessage `json:"issues"`
	Warnings []jsonMessage `json:"warnings"`
}

func renderJSON(report core.Report, opts Options) (string, error) {
	doc := jsonDocument{
		Tool: jsonTool{Name: "mcmodlist", Version: opts.Version},
		Instance: jsonInstance{
			Name:          report.Instance.Name,
			Path:          report.Instance.Path,
			Launcher:      report.Instance.Launcher,
			Minecraft:     report.Instance.Minecraft,
			Loader:        string(report.Instance.Loader),
			LoaderVersion: report.Instance.LoaderVersion,
		},
		Counts:   jsonCountsFor(report.Counts),
		Mods:     []jsonMod{},
		Issues:   []jsonMessage{},
		Warnings: []jsonMessage{},
	}
	if !opts.SummaryOnly {
		for _, mod := range report.Mods {
			authors := append([]string(nil), mod.Authors...)
			if authors == nil {
				authors = []string{}
			}
			doc.Mods = append(doc.Mods, jsonMod{
				Name:       mod.Name,
				ID:         mod.ID,
				Version:    mod.Version,
				Authors:    authors,
				Jar:        mod.Jar,
				Loader:     modLoaderLabel(mod),
				Side:       mod.Side,
				Status:     string(mod.Status),
				Change:     mod.Change,
				OldVersion: mod.OldVersion,
			})
		}
	}
	for _, issue := range report.Issues {
		doc.Issues = append(doc.Issues, jsonMessage{Type: issue.Type, Mod: issue.Mod, Message: issue.Message})
	}
	for _, warning := range report.Warnings {
		doc.Warnings = append(doc.Warnings, jsonMessage{Type: warning.Type, Mod: warning.Mod, Message: warning.Message})
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func jsonCountsFor(counts model.Counts) jsonCounts {
	out := jsonCounts{
		Fabric:       counts.Loaders["fabric"],
		Quilt:        counts.Loaders["quilt"],
		Forge:        counts.Loaders["forge"],
		NeoForge:     counts.Loaders["neoforge"],
		Total:        counts.Total,
		Disabled:     counts.Disabled,
		Unrecognised: counts.Unrecognised,
	}
	for label, count := range counts.Loaders {
		if strings.HasSuffix(label, "/connector") {
			out.Connector += count
		}
	}
	return out
}

func renderCSV(report core.Report, opts Options) string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	header := headerRow(opts)
	groupBy := strings.ToLower(strings.TrimSpace(opts.GroupBy))
	grouped := groupBy == "loader" || groupBy == "side"
	if grouped {
		header = append([]string{"Group"}, header...)
	}
	if showColumnHeader(opts) {
		_ = writer.Write(header)
	}
	if !opts.SummaryOnly {
		for _, group := range groupedMods(report, opts) {
			for _, mod := range group.Mods {
				row := modRowSelected(mod, opts)
				if grouped {
					row = append([]string{group.Title}, row...)
				}
				_ = writer.Write(row)
			}
		}
	}
	writer.Flush()
	return buf.String()
}
