package core

import (
	"strings"
	"testing"

	"mcmodlist/core/model"
)

func TestFilterMods(t *testing.T) {
	mods := []model.Mod{
		{Name: "Alpha", ID: "alpha", Version: "1", Authors: []string{"Alice"}, Jar: "a.jar", Loader: model.LoaderFabric, Side: "client", Status: model.StatusEnabled, Recognized: true},
		{Name: "Beta", ID: "beta", Version: "2", Authors: []string{"Bob"}, Jar: "b.jar", Loader: model.LoaderQuilt, Side: "server", Status: model.StatusEnabled, Recognized: true},
		{Name: "Off", ID: "off", Version: "1", Jar: "off.jar.disabled", Loader: model.LoaderFabric, Status: model.StatusDisabled, Recognized: true},
		{Name: "bad.jar", Jar: "bad.jar", Status: model.StatusEnabled, Recognized: false},
	}

	got := FilterMods(mods, FilterOptions{IncludeDisabled: true})
	if len(got) != 4 {
		t.Fatalf("include disabled: got %d mods", len(got))
	}
	got = FilterMods(mods, FilterOptions{})
	if len(got) != 3 {
		t.Fatalf("default should hide disabled, got %d", len(got))
	}
	got = FilterMods(mods, FilterOptions{Search: "alice"})
	if len(got) != 1 || got[0].ID != "alpha" {
		t.Fatalf("search alice: %+v", got)
	}
	got = FilterMods(mods, FilterOptions{Loader: "fabric"})
	if len(got) != 1 || got[0].ID != "alpha" {
		t.Fatalf("loader fabric: %+v", got)
	}
	got = FilterMods(mods, FilterOptions{Side: "server"})
	if len(got) != 1 || got[0].ID != "beta" {
		t.Fatalf("side server: %+v", got)
	}
	got = FilterMods(mods, FilterOptions{Author: "bob"})
	if len(got) != 1 || got[0].ID != "beta" {
		t.Fatalf("author bob: %+v", got)
	}
	got = FilterMods(mods, FilterOptions{UnrecognisedOnly: true})
	if len(got) != 1 || got[0].Recognized {
		t.Fatalf("unrecognised only: %+v", got)
	}
}

func TestFilterReportRecounts(t *testing.T) {
	report := Report{Mods: []model.Mod{
		{Name: "Alpha", ID: "alpha", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true},
		{Name: "Beta", ID: "beta", Loader: model.LoaderQuilt, Status: model.StatusEnabled, Recognized: true},
		{Name: "Off", ID: "off", Loader: model.LoaderForge, Status: model.StatusDisabled, Recognized: true},
	}}
	view := FilterReport(report, FilterOptions{Loader: "fabric"})
	if view.Counts.Total != 1 || view.Counts.Loaders["fabric"] != 1 {
		t.Fatalf("unexpected recount: %+v", view.Counts)
	}
	if len(view.Mods) != 1 || view.Mods[0].ID != "alpha" {
		t.Fatalf("unexpected filtered mods: %+v", view.Mods)
	}
}

func TestSortMods(t *testing.T) {
	mods := []model.Mod{
		{Name: "Beta", ID: "b", Version: "2", Jar: "b.jar", Loader: model.LoaderFabric, Status: model.StatusEnabled, Recognized: true},
		{Name: "Alpha", ID: "a", Version: "1", Jar: "a.jar", Loader: model.LoaderQuilt, Status: model.StatusEnabled, Recognized: true},
		{Name: "Alpha", ID: "a2", Version: "3", Jar: "c.jar", Loader: model.LoaderForge, Status: model.StatusEnabled, Recognized: true},
	}
	got, err := SortMods(mods, "name,id")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].ID != "a" || got[1].ID != "a2" || got[2].ID != "b" {
		t.Fatalf("name,id sort: %+v", got)
	}
	got, err = SortMods(mods, "name,-version")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Version != "3" || got[2].ID != "b" {
		t.Fatalf("name,-version sort: %+v", got)
	}
	if _, err := SortMods(mods, "bogus"); err == nil || !strings.Contains(err.Error(), "unknown sort key") {
		t.Fatalf("expected unknown sort key error, got %v", err)
	}
}

func TestOrderedLoaderLabels(t *testing.T) {
	counts := model.Counts{Loaders: map[string]int{
		"neoforge":         1,
		"forge":            1,
		"fabric/connector": 1,
		"quilt":            1,
		"fabric":           1,
		"unknown/thing":    1,
		"zero":             0,
	}}
	got := OrderedLoaderLabels(counts)
	want := []string{"fabric", "fabric/connector", "quilt", "forge", "neoforge", "unknown/thing"}
	if len(got) != len(want) {
		t.Fatalf("ordered labels = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordered labels = %v, want %v", got, want)
		}
	}
}
