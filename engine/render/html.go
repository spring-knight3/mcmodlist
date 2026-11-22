package render

import (
	"fmt"
	"html"
	"strings"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

func renderHTML(report core.Report, opts Options) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>mcmodlist</title>\n<style>\n")
	b.WriteString("body{font-family:system-ui,sans-serif;margin:2rem;color:#1b1b1b;background:#fff}\n")
	b.WriteString("table{border-collapse:collapse;width:100%}\n")
	b.WriteString("th,td{border:1px solid #ccc;padding:.35rem .5rem;text-align:left;vertical-align:top}\n")
	b.WriteString("th{background:#f3f3f3}\n.status-disabled{color:#8a4b00}.status-nested{color:#555}.status-unrecognised{color:#9b0000}\n")
	b.WriteString(".note{color:#555}\n")
	b.WriteString("</style>\n</head>\n<body>\n")
	b.WriteString("<h1>mcmodlist</h1>\n")
	if showSummary(opts) {
		b.WriteString("<dl>\n")
		for _, line := range summaryLines(report) {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				fmt.Fprintf(&b, "<dt>%s</dt><dd>%s</dd>\n", html.EscapeString(parts[0]), html.EscapeString(parts[1]))
			}
		}
		b.WriteString("</dl>\n")
	}
	if !opts.SummaryOnly {
		groups := groupedMods(report, opts)
		for _, group := range groups {
			if group.Title != "" {
				fmt.Fprintf(&b, "<h2>%s</h2>\n", html.EscapeString(group.Title))
			}
			writeHTMLTable(&b, group.Mods, opts)
		}
		if note := footerNote(report, opts); note != "" {
			fmt.Fprintf(&b, "<p class=\"note\">%s</p>\n", html.EscapeString(note))
		}
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func writeHTMLTable(b *strings.Builder, mods []model.Mod, opts Options) {
	header := headerRow(opts)
	statusIdx := indexOfString(header, "Status")
	b.WriteString("<table>\n")
	if showColumnHeader(opts) {
		b.WriteString("<thead><tr>")
		for _, column := range header {
			fmt.Fprintf(b, "<th>%s</th>", html.EscapeString(column))
		}
		b.WriteString("</tr></thead>\n")
	}
	b.WriteString("<tbody>\n")
	for _, mod := range mods {
		row := modRowSelected(mod, opts)
		b.WriteString("<tr>")
		for i, cell := range row {
			class := ""
			if i == statusIdx {
				class = " class=\"status-" + html.EscapeString(statusDisplay(mod)) + "\""
			}
			fmt.Fprintf(b, "<td%s>%s</td>", class, html.EscapeString(cell))
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
}

func indexOfString(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return -1
}
