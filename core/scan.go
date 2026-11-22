// Package core provides the mcmodlist scanning library.
package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"mcmodlist/core/instance"
	"mcmodlist/core/jar"
	"mcmodlist/core/model"
)

// Options controls Scan behaviour.
type Options struct {
	// Jobs bounds concurrent jar scanning. Non-positive values use a default.
	Jobs int
	// IncludeNested scans jars embedded inside other jars.
	IncludeNested bool
	// Recursive scans jar files in subdirectories of the mods directory.
	Recursive bool
	// Progress is called as jar files complete when non-nil. It may be called
	// concurrently from scan workers.
	Progress func(done, total int)
}

// Report is the structured result of scanning an instance.
type Report struct {
	Instance model.Instance  `json:"instance"`
	Mods     []model.Mod     `json:"mods"`
	Counts   model.Counts    `json:"counts"`
	Issues   []model.Issue   `json:"issues"`
	Warnings []model.Warning `json:"warnings"`
}

// Scan inspects path as a Minecraft instance and returns structured mod data.
func Scan(path string, opts Options) (Report, error) {
	det, err := instance.Detect(path)
	if err != nil {
		return Report{}, err
	}

	files, err := jarFiles(det.ModsDir, opts.Recursive)
	if err != nil {
		return Report{}, err
	}

	parsed := scanJars(files, opts)
	mods, warnings, activeLoader := classify(parsed, det.Info.Loader)
	sortMods(mods)

	report := Report{
		Instance: det.Info,
		Mods:     mods,
		Counts:   CountMods(mods),
		Issues:   []model.Issue{},
		Warnings: warnings,
	}
	if !report.Instance.Loader.Valid() {
		report.Instance.Loader = activeLoader
	}
	return report, nil
}

type jarResult struct {
	mods     []model.Mod
	warnings []model.Warning
}

type jarFile struct {
	path string
	name string
}

func jarFiles(modsDir string, recursive bool) ([]jarFile, error) {
	if modsDir == "" {
		return nil, nil
	}
	files := []jarFile{}
	if recursive {
		err := filepath.WalkDir(modsDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".jar") && !strings.HasSuffix(lower, ".jar.disabled") {
				return nil
			}
			rel, err := filepath.Rel(modsDir, path)
			if err != nil {
				return err
			}
			files = append(files, jarFile{path: path, name: filepath.ToSlash(rel)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(modsDir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			lower := strings.ToLower(name)
			if strings.HasSuffix(lower, ".jar") || strings.HasSuffix(lower, ".jar.disabled") {
				files = append(files, jarFile{path: filepath.Join(modsDir, name), name: name})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, nil
}

func scanJars(files []jarFile, opts Options) []jarResult {
	results := make([]jarResult, len(files))
	if len(files) == 0 {
		return results
	}
	if opts.Progress != nil {
		opts.Progress(0, len(files))
	}
	var done int64
	finish := func(i int, mods []model.Mod, warnings []model.Warning) {
		results[i] = jarResult{mods: mods, warnings: warnings}
		if opts.Progress != nil {
			opts.Progress(int(atomic.AddInt64(&done, 1)), len(files))
		}
	}
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = 4
	}
	if jobs > len(files) {
		jobs = len(files)
	}
	if jobs < 1 {
		jobs = 1
	}
	if jobs == 1 || runtime.NumCPU() == 1 {
		for i, file := range files {
			mods, warnings := jar.ParseFile(file.path, jar.Options{IncludeNested: opts.IncludeNested, Name: file.name})
			finish(i, mods, warnings)
		}
		return results
	}

	var wg sync.WaitGroup
	paths := make(chan int)
	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range paths {
				mods, warnings := jar.ParseFile(files[i].path, jar.Options{IncludeNested: opts.IncludeNested, Name: files[i].name})
				finish(i, mods, warnings)
			}
		}()
	}
	for i := range files {
		paths <- i
	}
	close(paths)
	wg.Wait()
	return results
}

func classify(results []jarResult, instanceLoader model.Loader) ([]model.Mod, []model.Warning, model.Loader) {
	var all []model.Mod
	var warnings []model.Warning
	for _, result := range results {
		all = append(all, result.mods...)
		warnings = append(warnings, result.warnings...)
	}

	selected, unrecognised := selectLoaderGroups(all, instanceLoader)
	activeLoader := instanceLoader
	if !activeLoader.Valid() {
		activeLoader = inferLoader(selected)
	}
	connector := hasConnector(selected)
	for i := range selected {
		selected[i].Loader, selected[i].Layer = effectiveLoader(selected[i].Loader, activeLoader, connector)
	}
	return append(unrecognised, selected...), warnings, activeLoader
}

func selectLoaderGroups(all []model.Mod, preferred model.Loader) ([]model.Mod, []model.Mod) {
	type group struct {
		byLoader map[model.Loader][]model.Mod
	}
	groups := make(map[string]*group)
	var order []string
	var unrecognised []model.Mod
	for _, mod := range all {
		if !mod.Recognized {
			unrecognised = append(unrecognised, mod)
			continue
		}
		key := mod.Jar
		if key == "" {
			key = mod.Name
		}
		g := groups[key]
		if g == nil {
			g = &group{byLoader: make(map[model.Loader][]model.Mod)}
			groups[key] = g
			order = append(order, key)
		}
		g.byLoader[mod.Loader] = append(g.byLoader[mod.Loader], mod)
	}

	loaderOrder := []model.Loader{model.LoaderFabric, model.LoaderQuilt, model.LoaderForge, model.LoaderNeoForge}
	var selected []model.Mod
	for _, key := range order {
		g := groups[key]
		chosen := model.LoaderUnknown
		if preferred.Valid() {
			if _, ok := g.byLoader[preferred]; ok {
				chosen = preferred
			}
		}
		if chosen == model.LoaderUnknown {
			for _, loader := range loaderOrder {
				if _, ok := g.byLoader[loader]; ok {
					chosen = loader
					break
				}
			}
		}
		for _, mods := range g.byLoader[chosen] {
			selected = append(selected, mods)
		}
	}
	return selected, unrecognised
}

func hasConnector(mods []model.Mod) bool {
	for _, mod := range mods {
		if !mod.Recognized || mod.Status != model.StatusEnabled {
			continue
		}
		id := strings.ToLower(mod.ID)
		name := strings.ToLower(mod.Name)
		if id == "connector" || strings.Contains(name, "sinytra connector") {
			return true
		}
	}
	return false
}

func effectiveLoader(loader model.Loader, active model.Loader, connector bool) (model.Loader, string) {
	if !loader.Valid() || !active.Valid() {
		return loader, ""
	}
	if connector && (active == model.LoaderFabric || active == model.LoaderQuilt) && (loader == model.LoaderForge || loader == model.LoaderNeoForge) {
		return active, model.LayerConnector
	}
	return loader, ""
}

func sortMods(mods []model.Mod) {
	sort.SliceStable(mods, func(i, j int) bool {
		a, b := mods[i], mods[j]
		if an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name); an != bn {
			return an < bn
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Jar != b.Jar {
			return a.Jar < b.Jar
		}
		if a.Loader != b.Loader {
			return a.Loader < b.Loader
		}
		if a.Layer != b.Layer {
			return a.Layer < b.Layer
		}
		return a.Status < b.Status
	})
}

// CountMods returns loader/status counts for a mod list.
func CountMods(mods []model.Mod) model.Counts {
	counts := model.NewCounts()
	for _, mod := range mods {
		counts.Total++
		if mod.Status == model.StatusDisabled {
			counts.Disabled++
		}
		if !mod.Recognized {
			counts.Unrecognised++
			continue
		}
		if mod.Status != model.StatusEnabled {
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
	return counts
}

// OrderedLoaderLabels returns loader count labels in display order: Fabric,
// Quilt, Forge, NeoForge, with compatibility layers after their base loader.
func OrderedLoaderLabels(counts model.Counts) []string {
	baseOrder := []model.Loader{model.LoaderFabric, model.LoaderQuilt, model.LoaderForge, model.LoaderNeoForge}
	layers := make(map[model.Loader][]string)
	var unknown []string
	for label, count := range counts.Loaders {
		if count <= 0 {
			continue
		}
		loaderPart, _, _ := strings.Cut(label, "/")
		loader := model.ParseLoader(loaderPart)
		if !loader.Valid() {
			unknown = append(unknown, label)
			continue
		}
		layers[loader] = append(layers[loader], label)
	}
	var out []string
	for _, loader := range baseOrder {
		var plain, withLayer []string
		for _, label := range layers[loader] {
			if _, _, ok := strings.Cut(label, "/"); ok {
				withLayer = append(withLayer, label)
			} else {
				plain = append(plain, label)
			}
		}
		sort.Strings(plain)
		sort.Strings(withLayer)
		out = append(out, plain...)
		out = append(out, withLayer...)
	}
	sort.Strings(unknown)
	return append(out, unknown...)
}

func inferLoader(mods []model.Mod) model.Loader {
	counts := make(map[model.Loader]int)
	for _, mod := range mods {
		if !mod.Recognized || mod.Status != model.StatusEnabled || mod.Layer != "" {
			continue
		}
		if mod.Loader.Valid() {
			counts[mod.Loader]++
		}
	}
	order := []model.Loader{model.LoaderFabric, model.LoaderQuilt, model.LoaderForge, model.LoaderNeoForge}
	best := model.LoaderUnknown
	bestCount := 0
	for _, loader := range order {
		if counts[loader] > bestCount {
			best = loader
			bestCount = counts[loader]
		}
	}
	return best
}
