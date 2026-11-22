package jar

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mcmodlist/core/model"
)

// Options controls jar scanning behaviour.
type Options struct {
	// IncludeNested scans .jar entries inside jars as nested mods.
	IncludeNested bool
	// Name overrides the jar display name (used for recursive scans).
	Name string
}

// ParseFile scans one jar file and returns mod candidates plus warnings.
// It never returns an error for a bad jar; unreadable or unrecognised jars are
// reported as unrecognised entries with a warning.
func ParseFile(path string, opts Options) ([]model.Mod, []model.Warning) {
	name := opts.Name
	if name == "" {
		name = filepath.Base(path)
	}
	status := model.StatusEnabled
	if strings.HasSuffix(name, ".jar.disabled") {
		status = model.StatusDisabled
	}

	f, err := os.Open(path)
	if err != nil {
		return unrecognised(name, status), []model.Warning{{Type: "unreadable", Mod: name, Message: fmt.Sprintf("cannot open jar: %v", err)}}
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return unrecognised(name, status), []model.Warning{{Type: "unreadable", Mod: name, Message: fmt.Sprintf("cannot stat jar: %v", err)}}
	}
	zr, err := zip.NewReader(f, st.Size())
	if err != nil {
		return unrecognised(name, status), []model.Warning{{Type: "corrupt", Mod: name, Message: fmt.Sprintf("cannot read jar: %v", err)}}
	}
	return parseArchive(zr, name, status, opts, 0)
}

func parseArchive(zr *zip.Reader, jarName string, status model.Status, opts Options, depth int) ([]model.Mod, []model.Warning) {
	mods := parseMetadata(zr, jarName, status)
	var warnings []model.Warning
	var nested []model.Mod
	if opts.IncludeNested && depth == 0 {
		nested, warnings = scanNested(zr, jarName, opts, warnings)
	}
	if len(mods) == 0 && len(nested) == 0 {
		mods = unrecognised(jarName, status)
		warnings = append(warnings, model.Warning{Type: "unrecognised", Mod: jarName, Message: "no recognised mod metadata"})
	}
	mods = append(mods, nested...)
	resolveVersions(zr, mods)
	return mods, warnings
}

func scanNested(zr *zip.Reader, outerName string, opts Options, warnings []model.Warning) ([]model.Mod, []model.Warning) {
	var out []model.Mod
	for _, file := range zr.File {
		name := file.Name
		if file.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(name), ".jar") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			warnings = append(warnings, model.Warning{Type: "nested", Mod: outerName, Message: fmt.Sprintf("cannot open nested jar %s: %v", name, err)})
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			warnings = append(warnings, model.Warning{Type: "nested", Mod: outerName, Message: fmt.Sprintf("cannot read nested jar %s: %v", name, err)})
			continue
		}
		nzr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			warnings = append(warnings, model.Warning{Type: "nested", Mod: outerName, Message: fmt.Sprintf("cannot parse nested jar %s: %v", name, err)})
			continue
		}
		nestedMods, nestedWarnings := parseArchive(nzr, outerName+"!/"+name, model.StatusNested, opts, 1)
		out = append(out, nestedMods...)
		warnings = append(warnings, nestedWarnings...)
	}
	return out, warnings
}

func resolveVersions(zr *zip.Reader, mods []model.Mod) {
	needsManifest := false
	for _, mod := range mods {
		if strings.Contains(mod.Version, "${") {
			needsManifest = true
			break
		}
	}
	var manifest map[string]string
	if needsManifest {
		manifest = manifestAttributes(zr)
	}
	for i := range mods {
		mods[i].Version = resolveVersion(mods[i].Version, manifest)
	}
}

func unrecognised(jarName string, status model.Status) []model.Mod {
	return []model.Mod{{
		Name:       jarName,
		Version:    unknownVersion,
		Jar:        jarName,
		Status:     status,
		Recognized: false,
	}}
}

func findFile(zr *zip.Reader, name string) *zip.File {
	for _, file := range zr.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func readFile(zr *zip.Reader, name string) ([]byte, bool) {
	file := findFile(zr, name)
	if file == nil {
		return nil, false
	}
	rc, err := file.Open()
	if err != nil {
		return nil, false
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, false
	}
	return data, true
}
