// Package render turns core scan reports into text output formats.
package render

import (
	"fmt"
	"strings"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

// Options controls rendering.
type Options struct {
	Color           bool
	NoHeader        bool
	NoSummary       bool
	SummaryOnly     bool
	Width           int
	Version         string
	Columns         []string
	GroupBy         string
	Wide            bool
	Truncate        int
	SkippedDisabled int
}

// Render returns report formatted as format.
func Render(format string, report core.Report, opts Options) (string, error) {
	switch strings.ToLower(format) {
	case "pretty":
		return renderPretty(report, opts), nil
	case "markdown":
		return renderMarkdown(report, opts), nil
	case "csv":
		return renderCSV(report, opts), nil
	case "json":
		return renderJSON(report, opts)
	case "html":
		return renderHTML(report, opts), nil
	case "text":
		return renderText(report, opts), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func summaryLines(report core.Report) []string {
	inst := report.Instance
	minecraft := inst.Minecraft
	if minecraft == "" {
		minecraft = "unknown"
	}
	loader := titleLoader(inst.Loader)
	if loader == "" {
		loader = "unknown"
	}
	if inst.LoaderVersion != "" {
		loader += " " + inst.LoaderVersion
	}
	counts := loaderCountsText(report.Counts)
	if counts == "" {
		counts = "none"
	}
	return []string{
		"Instance: " + inst.Name,
		"Minecraft: " + minecraft,
		"Loader: " + loader,
		"Loader counts: " + counts,
		fmt.Sprintf("Total: %d", report.Counts.Total),
	}
}

func showSummary(opts Options) bool {
	return !opts.NoSummary && !opts.NoHeader
}

func showColumnHeader(opts Options) bool {
	return !opts.NoHeader
}

func footerNote(report core.Report, opts Options) string {
	disabled := opts.SkippedDisabled + report.Counts.Disabled
	unrecognised := report.Counts.Unrecognised
	if disabled == 0 && unrecognised == 0 {
		return ""
	}
	parts := make([]string, 0, 2)
	if disabled > 0 {
		parts = append(parts, fmt.Sprintf("%d disabled mod(s)", disabled))
	}
	if unrecognised > 0 {
		parts = append(parts, fmt.Sprintf("%d unrecognised jar(s)", unrecognised))
	}
	return "Note: loader counts exclude " + strings.Join(parts, " and ") + "."
}

func loaderCountsText(counts model.Counts) string {
	labels := orderedLabels(counts)
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("%s(%d)", displayLabel(label), counts.Loaders[label]))
	}
	return strings.Join(parts, " ")
}

func orderedLabels(counts model.Counts) []string {
	return core.OrderedLoaderLabels(counts)
}

func splitLabel(label string) (model.Loader, string) {
	parts := strings.SplitN(label, "/", 2)
	loader := model.ParseLoader(parts[0])
	if len(parts) == 1 {
		return loader, ""
	}
	return loader, parts[1]
}

func displayLabel(label string) string {
	loader, layer := splitLabel(label)
	text := titleLoader(loader)
	if text == "" {
		text = label
	}
	if layer != "" {
		text += "/" + titleWord(layer)
	}
	return text
}

// DisplayLoaderLabel returns the user-facing loader/layer label.
func DisplayLoaderLabel(label string) string { return displayLabel(label) }

func titleLoader(loader model.Loader) string {
	switch loader {
	case model.LoaderFabric:
		return "Fabric"
	case model.LoaderQuilt:
		return "Quilt"
	case model.LoaderForge:
		return "Forge"
	case model.LoaderNeoForge:
		return "NeoForge"
	default:
		return ""
	}
}

func titleWord(word string) string {
	if word == "" {
		return ""
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func modLoaderLabel(mod model.Mod) string {
	if mod.Layer != "" {
		return string(mod.Loader) + "/" + mod.Layer
	}
	return string(mod.Loader)
}

func modLoaderDisplay(mod model.Mod) string {
	if mod.Layer != "" {
		return displayLabel(modLoaderLabel(mod))
	}
	if text := titleLoader(mod.Loader); text != "" {
		return text
	}
	return ""
}

func statusDisplay(mod model.Mod) string {
	if mod.Change != "" {
		return mod.Change
	}
	if !mod.Recognized {
		return "unrecognised"
	}
	return string(mod.Status)
}

func versionDisplay(mod model.Mod) string {
	if mod.OldVersion != "" {
		return mod.OldVersion + " -> " + mod.Version
	}
	return mod.Version
}

func nameDisplay(mod model.Mod) string {
	if !mod.Recognized {
		return mod.Name + " (unrecognised)"
	}
	return mod.Name
}

func authorsDisplay(authors []string, sep string) string {
	return strings.Join(authors, sep)
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
