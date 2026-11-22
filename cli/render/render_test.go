package render

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcmodlist/core"
	"mcmodlist/core/model"
)

var update = flag.Bool("update", false, "update golden files")

func fixtureReport() core.Report {
	return core.Report{
		Instance: model.Instance{
			Name:          "Test Instance",
			Path:          "/tmp/test-instance",
			Launcher:      "prism",
			Minecraft:     "1.20.1",
			Loader:        model.LoaderFabric,
			LoaderVersion: "0.15.0",
		},
		Mods: []model.Mod{
			{Name: "Fabric Mod", ID: "fabric-mod", Version: "1.0.0", Authors: []string{"Alice", "Bob"}, Jar: "fabric.jar", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true},
			{Name: "Connector Forge", ID: "forge-mod", Version: "2.0.0", Authors: []string{"Carol"}, Jar: "forge.jar", Loader: model.LoaderFabric, Layer: model.LayerConnector, Status: model.StatusEnabled, Recognized: true},
			{Name: "Old Forge", ID: "old-forge", Version: "0.1.0", Jar: "old.jar.disabled", Loader: model.LoaderForge, Status: model.StatusDisabled, Recognized: true},
			{Name: "mystery.jar", Version: "unknown", Jar: "mystery.jar", Status: model.StatusEnabled, Recognized: false},
		},
		Counts: model.Counts{
			Total:        4,
			Disabled:     1,
			Unrecognised: 1,
			Loaders: map[string]int{
				"fabric":           1,
				"fabric/connector": 1,
			},
		},
		Issues: []model.Issue{},
		Warnings: []model.Warning{
			{Type: "corrupt", Mod: "mystery.jar", Message: "cannot read jar"},
		},
	}
}

func TestGoldenFormats(t *testing.T) {
	report := fixtureReport()
	formats := []string{"pretty", "markdown", "csv", "json", "html", "text"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			got, err := Render(format, report, Options{Width: 200, Version: "test"})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", format+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to create)", path, err)
			}
			if got != string(want) {
				t.Fatalf("format %s mismatch\n--- got ---\n%s\n--- want ---\n%s", format, got, string(want))
			}
		})
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := Render("yaml", fixtureReport(), Options{}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestDiffRendering(t *testing.T) {
	report := fixtureReport()
	report.Mods = []model.Mod{
		{Name: "Added Mod", ID: "added", Version: "1.0.0", Jar: "added.jar", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true, Change: "added"},
		{Name: "Removed Mod", ID: "removed", Version: "1.0.0", Jar: "removed.jar", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true, Change: "removed"},
		{Name: "Changed Mod", ID: "changed", Version: "2.0.0", OldVersion: "1.0.0", Jar: "changed.jar", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true, Change: "version-changed"},
	}
	report.Counts = core.CountMods(report.Mods)

	for _, format := range []string{"pretty", "markdown", "csv", "json", "html", "text"} {
		got, err := Render(format, report, Options{Width: 200, Version: "test", Wide: true})
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		for _, want := range []string{"added", "removed", "version-changed"} {
			if !strings.Contains(got, want) {
				t.Fatalf("format %s missing %q:\n%s", format, want, got)
			}
		}
		if format == "html" {
			if !strings.Contains(got, "1.0.0") || !strings.Contains(got, "2.0.0") {
				t.Fatalf("format html missing version change:\n%s", got)
			}
		} else if !strings.Contains(got, "1.0.0 -> 2.0.0") && format != "json" {
			t.Fatalf("format %s missing old -> new version:\n%s", format, got)
		}
	}

	jsonOut, err := Render("json", report, Options{Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Mods []struct {
			ID         string `json:"id"`
			Change     string `json:"change"`
			OldVersion string `json:"oldVersion"`
		} `json:"mods"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &doc); err != nil {
		t.Fatal(err)
	}
	byID := map[string]struct {
		Change     string
		OldVersion string
	}{}
	for _, mod := range doc.Mods {
		byID[mod.ID] = struct {
			Change     string
			OldVersion string
		}{mod.Change, mod.OldVersion}
	}
	if byID["changed"].Change != "version-changed" || byID["changed"].OldVersion != "1.0.0" {
		t.Fatalf("json missing additive diff fields: %+v", byID["changed"])
	}
}
