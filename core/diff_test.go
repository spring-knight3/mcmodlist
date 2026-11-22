package core

import (
	"testing"

	"mcmodlist/core/model"
)

func diffMod(id, version string) model.Mod {
	return model.Mod{
		Name:       id,
		ID:         id,
		Version:    version,
		Jar:        id + ".jar",
		Loader:     model.LoaderFabric,
		Status:     model.StatusEnabled,
		Recognized: true,
	}
}

func diffReport(mods ...model.Mod) Report {
	return Report{Mods: mods, Counts: CountMods(mods)}
}

func diffByID(report Report) map[string]model.Mod {
	out := make(map[string]model.Mod, len(report.Mods))
	for _, mod := range report.Mods {
		out[mod.ID] = mod
	}
	return out
}

func TestDiffAddedRemovedChanged(t *testing.T) {
	oldReport := diffReport(
		diffMod("shared", "1.0.0"),
		diffMod("removed", "1.0.0"),
		diffMod("changed", "1.0.0"),
	)
	newReport := diffReport(
		diffMod("shared", "1.0.0"),
		diffMod("added", "1.0.0"),
		diffMod("changed", "2.0.0"),
	)

	got := Diff(oldReport, newReport)
	byID := diffByID(got)
	if len(got.Mods) != 3 {
		t.Fatalf("expected 3 diff entries, got %+v", got.Mods)
	}
	if byID["added"].Change != "added" {
		t.Fatalf("added entry missing change: %+v", byID["added"])
	}
	if byID["removed"].Change != "removed" {
		t.Fatalf("removed entry missing change: %+v", byID["removed"])
	}
	changed := byID["changed"]
	if changed.Change != "version-changed" || changed.OldVersion != "1.0.0" || changed.Version != "2.0.0" {
		t.Fatalf("unexpected version-changed entry: %+v", changed)
	}
	if _, ok := byID["shared"]; ok {
		t.Fatalf("unchanged mod should not appear in diff: %+v", byID["shared"])
	}
}

func TestDiffSymmetric(t *testing.T) {
	oldReport := diffReport(
		diffMod("only-old", "1.0.0"),
		diffMod("changed", "1.0.0"),
	)
	newReport := diffReport(
		diffMod("only-new", "1.0.0"),
		diffMod("changed", "2.0.0"),
	)

	forward := diffByID(Diff(oldReport, newReport))
	reverse := diffByID(Diff(newReport, oldReport))

	if forward["only-new"].Change != "added" || reverse["only-new"].Change != "removed" {
		t.Fatalf("added/removed not symmetric for only-new: %+v %+v", forward["only-new"], reverse["only-new"])
	}
	if forward["only-old"].Change != "removed" || reverse["only-old"].Change != "added" {
		t.Fatalf("added/removed not symmetric for only-old: %+v %+v", forward["only-old"], reverse["only-old"])
	}
	if forward["changed"].Change != "version-changed" || reverse["changed"].Change != "version-changed" {
		t.Fatalf("version change should be symmetric: %+v %+v", forward["changed"], reverse["changed"])
	}
	if forward["changed"].OldVersion != "1.0.0" || reverse["changed"].OldVersion != "2.0.0" {
		t.Fatalf("old versions should swap: %+v %+v", forward["changed"], reverse["changed"])
	}
}
