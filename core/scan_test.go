package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func newInstance(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func addJar(t *testing.T, root, filename string, entries map[string]string) {
	t.Helper()
	testutil.BuildTextJar(t, filepath.Join(root, "mods", filename), entries)
}

func addBytes(t *testing.T, root, filename string, data []byte) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(root, "mods", filename), data)
}

func findMod(mods []model.Mod, id string) *model.Mod {
	for i := range mods {
		if mods[i].ID == id {
			return &mods[i]
		}
	}
	return nil
}

func TestScanOneJarPerLoader(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "fabric.jar", map[string]string{
		"fabric.mod.json": `{"id":"fabric-mod","name":"Fabric Mod","version":"1","authors":["A"]}`,
	})
	addJar(t, root, "quilt.jar", map[string]string{
		"quilt.mod.json": `{"id":"quilt-mod","name":"Quilt Mod","version":"1"}`,
	})
	addJar(t, root, "forge.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-mod\"\nversion = \"1\"\ndisplayName = \"Forge Mod\"\n",
	})
	addJar(t, root, "neoforge.jar", map[string]string{
		"META-INF/neoforge.mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"neoforge-mod\"\nversion = \"1\"\ndisplayName = \"NeoForge Mod\"\n",
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Instance.Loader != model.LoaderFabric {
		t.Fatalf("expected inferred fabric loader, got %q", report.Instance.Loader)
	}
	for _, id := range []string{"fabric-mod", "quilt-mod", "forge-mod", "neoforge-mod"} {
		if findMod(report.Mods, id) == nil {
			t.Fatalf("missing mod %s in %+v", id, report.Mods)
		}
	}
	for _, label := range []string{"fabric", "quilt", "forge", "neoforge"} {
		if report.Counts.Loaders[label] != 1 {
			t.Fatalf("expected %s count 1, got %+v", label, report.Counts.Loaders)
		}
	}
	if report.Counts.Total != 4 || report.Counts.Unrecognised != 0 {
		t.Fatalf("unexpected counts: %+v", report.Counts)
	}
}

func TestScanMultipleModsAndMultiLoaderOnce(t *testing.T) {
	root := newInstance(t)
	testutil.WriteTextFile(t, filepath.Join(root, "instance.cfg"), "[General]\nname=Fabric Pack\n")
	testutil.WriteTextFile(t, filepath.Join(root, "mmc-pack.json"), `{"components":[{"uid":"net.minecraft","version":"1.20.1"},{"uid":"net.fabricmc.fabric-loader","version":"0.15.0"}]}`)

	addJar(t, root, "multi.toml.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-a\"\nversion = \"1\"\ndisplayName = \"Forge A\"\n[[mods]]\nmodId = \"forge-b\"\nversion = \"1\"\ndisplayName = \"Forge B\"\n",
	})
	addJar(t, root, "multiloader.jar", map[string]string{
		"fabric.mod.json":    `{"id":"multi-fabric","name":"Multi Fabric","version":"1"}`,
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"multi-forge\"\nversion = \"1\"\ndisplayName = \"Multi Forge\"\n",
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Instance.Loader != model.LoaderFabric {
		t.Fatalf("expected fabric instance loader, got %q", report.Instance.Loader)
	}
	if findMod(report.Mods, "forge-a") == nil || findMod(report.Mods, "forge-b") == nil {
		t.Fatalf("expected both forge mods, got %+v", report.Mods)
	}
	if findMod(report.Mods, "multi-forge") != nil {
		t.Fatalf("multi-loader jar should count once under fabric, got %+v", report.Mods)
	}
	multi := findMod(report.Mods, "multi-fabric")
	if multi == nil || multi.Loader != model.LoaderFabric || multi.Jar != "multiloader.jar" {
		t.Fatalf("unexpected multi-loader selection: %+v", multi)
	}
	if report.Counts.Loaders["forge"] != 2 || report.Counts.Loaders["fabric"] != 1 {
		t.Fatalf("unexpected loader counts: %+v", report.Counts.Loaders)
	}
}

func TestScanDisabledCorruptAndMetadataLess(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "disabled.jar.disabled", map[string]string{
		"fabric.mod.json": `{"id":"disabled-mod","name":"Disabled Mod","version":"1"}`,
	})
	addBytes(t, root, "corrupt.jar", []byte("not a zip"))
	addJar(t, root, "empty.jar", map[string]string{"README.txt": "no metadata"})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Total != 3 {
		t.Fatalf("expected total 3, got %+v", report.Counts)
	}
	if report.Counts.Disabled != 1 {
		t.Fatalf("expected disabled count 1, got %+v", report.Counts)
	}
	if report.Counts.Unrecognised != 2 {
		t.Fatalf("expected unrecognised count 2, got %+v", report.Counts)
	}
	if report.Counts.Loaders["fabric"] != 0 {
		t.Fatalf("disabled mod should not count in loader totals: %+v", report.Counts.Loaders)
	}
	if len(report.Warnings) < 2 {
		t.Fatalf("expected warnings for corrupt and metadata-less jars, got %+v", report.Warnings)
	}
	disabled := findMod(report.Mods, "disabled-mod")
	if disabled == nil || disabled.Status != model.StatusDisabled {
		t.Fatalf("expected disabled status, got %+v", disabled)
	}
}

func TestScanDeterministicOrder(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "b.jar", map[string]string{"fabric.mod.json": `{"id":"zeta","name":"Zeta","version":"1"}`})
	addJar(t, root, "a.jar", map[string]string{"fabric.mod.json": `{"id":"alpha","name":"Alpha","version":"1"}`})
	addJar(t, root, "c.jar", map[string]string{"quilt.mod.json": `{"id":"middle","name":"Middle","version":"1"}`})

	first, err := Scan(root, Options{Jobs: 3})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan(root, Options{Jobs: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Mods, second.Mods) {
		t.Fatalf("mod order differs:\nfirst:  %+v\nsecond: %+v", first.Mods, second.Mods)
	}
	want := []string{"Alpha", "Middle", "Zeta"}
	for i, name := range want {
		if first.Mods[i].Name != name {
			t.Fatalf("position %d: got %q want %q", i, first.Mods[i].Name, name)
		}
	}
}

type fileSnapshot struct {
	data    []byte
	modTime time.Time
}

func snapshotTree(t *testing.T, root string) map[string]fileSnapshot {
	t.Helper()
	out := make(map[string]fileSnapshot)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[path] = fileSnapshot{data: data, modTime: info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestScanDoesNotModifyInstance(t *testing.T) {
	root := newInstance(t)
	addJar(t, root, "fabric.jar", map[string]string{
		"fabric.mod.json": `{"id":"fabric-mod","name":"Fabric Mod","version":"1"}`,
	})
	addJar(t, root, "forge.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-mod\"\nversion = \"1\"\ndisplayName = \"Forge Mod\"\n",
	})
	testutil.WriteTextFile(t, filepath.Join(root, "instance.cfg"), "[General]\nname=Read Only\n")

	before := snapshotTree(t, root)
	if _, err := Scan(root, Options{}); err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("scan modified files inside the instance")
	}
}
