// Package render turns core scan reports into text output formats.
package render

import (
	"mcmodlist/core"
	enginerender "mcmodlist/engine/render"
)

// Options controls rendering.
type Options = enginerender.Options

// Render returns report formatted as format.
func Render(format string, report core.Report, opts Options) (string, error) {
	return enginerender.Render(format, report, opts)
}
