package core

import (
	"path/filepath"
	"testing"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func TestScanConnectorReclassifiesForgeMods(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "connector.jar", map[string]string{
		"fabric.mod.json": `{"id":"connector","name":"Sinytra Connector","version":"1"}`,
	})
	addJar(t, root, "forge-mod.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-via-connector\"\nversion = \"1\"\ndisplayName = \"Forge Via Connector\"\n",
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Instance.Loader != model.LoaderFabric {
		t.Fatalf("expected inferred fabric loader, got %q", report.Instance.Loader)
	}
	connector := findMod(report.Mods, "connector")
	if connector == nil || connector.Loader != model.LoaderFabric || connector.Layer != "" {
		t.Fatalf("unexpected connector mod: %+v", connector)
	}
	forge := findMod(report.Mods, "forge-via-connector")
	if forge == nil {
		t.Fatal("missing forge mod")
	}
	if forge.Loader != model.LoaderFabric || forge.Layer != model.LayerConnector {
		t.Fatalf("expected fabric/connector classification, got %+v", forge)
	}
	if report.Counts.Loaders["fabric"] != 1 || report.Counts.Loaders["fabric/connector"] != 1 {
		t.Fatalf("unexpected counts: %+v", report.Counts.Loaders)
	}
	if report.Counts.Loaders["forge"] != 0 {
		t.Fatalf("forge mod should not count as plain forge: %+v", report.Counts.Loaders)
	}
}

func TestScanForgeModWithoutConnectorStaysForge(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "fabric.jar", map[string]string{
		"fabric.mod.json": `{"id":"fabric-mod","name":"Fabric Mod","version":"1"}`,
	})
	addJar(t, root, "forge.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-mod\"\nversion = \"1\"\ndisplayName = \"Forge Mod\"\n",
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	forge := findMod(report.Mods, "forge-mod")
	if forge == nil || forge.Loader != model.LoaderForge || forge.Layer != "" {
		t.Fatalf("expected plain forge classification, got %+v", forge)
	}
	if report.Counts.Loaders["fabric"] != 1 || report.Counts.Loaders["forge"] != 1 {
		t.Fatalf("unexpected counts: %+v", report.Counts.Loaders)
	}
}

func TestScanNestedJarsOption(t *testing.T) {
	inner := testutil.JarBytes(t, testutil.TextEntries(map[string]string{
		"fabric.mod.json": `{"id":"inner-mod","name":"Inner Mod","version":"1"}`,
	}))
	root := newInstance(t)
	testutil.BuildJar(t, filepath.Join(root, "mods", "outer.jar"), map[string][]byte{
		"fabric.mod.json":           []byte(`{"id":"outer-mod","name":"Outer Mod","version":"1"}`),
		"META-INF/jarjar/inner.jar": inner,
	})

	defaultReport, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if findMod(defaultReport.Mods, "inner-mod") != nil {
		t.Fatalf("nested mod should be ignored by default: %+v", defaultReport.Mods)
	}
	if findMod(defaultReport.Mods, "outer-mod") == nil {
		t.Fatalf("missing outer mod: %+v", defaultReport.Mods)
	}

	nestedReport, err := Scan(root, Options{IncludeNested: true})
	if err != nil {
		t.Fatal(err)
	}
	innerMod := findMod(nestedReport.Mods, "inner-mod")
	if innerMod == nil || innerMod.Status != model.StatusNested {
		t.Fatalf("expected nested inner mod, got %+v", nestedReport.Mods)
	}
	if nestedReport.Counts.Loaders["fabric"] != 1 {
		t.Fatalf("nested mod should not add to enabled loader counts: %+v", nestedReport.Counts.Loaders)
	}
	if nestedReport.Counts.Total != 2 {
		t.Fatalf("expected total 2, got %+v", nestedReport.Counts)
	}
}

func TestScanUnrecognisedJarName(t *testing.T) {
	root := newInstance(t)
	addBytes(t, root, "mystery.jar", []byte("not a zip"))

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mods) != 1 {
		t.Fatalf("expected one mod, got %+v", report.Mods)
	}
	mod := report.Mods[0]
	if mod.Recognized || mod.Name != "mystery.jar" || mod.Version != "unknown" {
		t.Fatalf("unexpected unrecognised mod: %+v", mod)
	}
	if report.Counts.Unrecognised != 1 {
		t.Fatalf("unexpected counts: %+v", report.Counts)
	}
}
