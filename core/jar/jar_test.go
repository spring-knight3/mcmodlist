package jar

import (
	"path/filepath"
	"testing"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func TestParseFabricMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fabric.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"fabric.mod.json": `{"id":"fabric-example","name":"Fabric Example","version":"1.0.0","authors":["Alice","Bob"],"description":"test"}`,
	})
	mods, warnings := ParseFile(path, Options{})
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	mod := mods[0]
	if !mod.Recognized || mod.Loader != model.LoaderFabric {
		t.Fatalf("unexpected mod: %+v", mod)
	}
	if mod.Name != "Fabric Example" || mod.ID != "fabric-example" || mod.Version != "1.0.0" {
		t.Fatalf("unexpected fields: %+v", mod)
	}
	if len(mod.Authors) != 2 || mod.Authors[0] != "Alice" || mod.Authors[1] != "Bob" {
		t.Fatalf("unexpected authors: %v", mod.Authors)
	}
}

func TestParseQuiltMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quilt.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"quilt.mod.json": `{"id":"quilt-example","name":"Quilt Example","version":"2.0.0","authors":{"name":"Quilt Author"}}`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	if mods[0].Loader != model.LoaderQuilt || mods[0].ID != "quilt-example" {
		t.Fatalf("unexpected mod: %+v", mods[0])
	}
	if len(mods[0].Authors) != 1 || mods[0].Authors[0] != "Quilt Author" {
		t.Fatalf("unexpected authors: %v", mods[0].Authors)
	}
}

func TestParseForgeTOMLWithManifestVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\nImplementation-Version: 1.2.3\n\n",
		"META-INF/mods.toml": `modLoader = "javafml"
loaderVersion = "[47,)"
[[mods]]
modId = "forge-example"
version = "${file.jarVersion}"
displayName = "Forge Example"
authors = "Alice, Bob"
side = "BOTH"
`,
	})
	mods, warnings := ParseFile(path, Options{})
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	mod := mods[0]
	if mod.Loader != model.LoaderForge || mod.ID != "forge-example" || mod.Version != "1.2.3" {
		t.Fatalf("unexpected mod: %+v", mod)
	}
	if mod.Name != "Forge Example" || len(mod.Authors) != 2 {
		t.Fatalf("unexpected fields: %+v", mod)
	}
}

func TestParseNeoForgeTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "neoforge.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"META-INF/neoforge.mods.toml": `modLoader = "javafml"
[[mods]]
modId = "neoforge-example"
version = "20.0.0"
displayName = "NeoForge Example"
authors = "Carol"
`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Loader != model.LoaderNeoForge || mods[0].ID != "neoforge-example" {
		t.Fatalf("unexpected mods: %+v", mods)
	}
}

func TestParseLegacyMCModInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"mcmod.info": `[{"modid":"legacy-example","name":"Legacy Example","version":"0.1","authorList":["Dave","Erin"],"side":"CLIENT"}]`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	mod := mods[0]
	if mod.Loader != model.LoaderForge || mod.ID != "legacy-example" || mod.Side != "CLIENT" {
		t.Fatalf("unexpected mod: %+v", mod)
	}
	if len(mod.Authors) != 2 || mod.Authors[0] != "Dave" {
		t.Fatalf("unexpected authors: %v", mod.Authors)
	}
}

func TestMultipleModsPerJar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"META-INF/mods.toml": `modLoader = "javafml"
[[mods]]
modId = "first"
version = "1"
displayName = "First"
[[mods]]
modId = "second"
version = "2"
displayName = "Second"
`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods, got %d: %+v", len(mods), mods)
	}
	if mods[0].ID != "first" || mods[1].ID != "second" {
		t.Fatalf("unexpected mods: %+v", mods)
	}
	if mods[0].Jar != "multi.jar" || mods[1].Jar != "multi.jar" {
		t.Fatalf("mods must carry jar name: %+v", mods)
	}
}

func TestMultiLoaderJarReturnsCandidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multiloader.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"fabric.mod.json": `{"id":"multi","name":"Multi","version":"1"}`,
		"META-INF/mods.toml": `modLoader = "javafml"
[[mods]]
modId = "multi"
version = "1"
displayName = "Multi Forge"
`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %+v", len(mods), mods)
	}
	loaders := map[model.Loader]bool{}
	for _, mod := range mods {
		loaders[mod.Loader] = true
	}
	if !loaders[model.LoaderFabric] || !loaders[model.LoaderForge] {
		t.Fatalf("expected fabric and forge candidates: %+v", mods)
	}
}

func TestCorruptJar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.jar")
	testutil.WriteFile(t, path, []byte("this is not a zip"))
	mods, warnings := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Recognized {
		t.Fatalf("expected one unrecognised mod: %+v", mods)
	}
	if mods[0].Name != "corrupt.jar" || mods[0].Status != model.StatusEnabled {
		t.Fatalf("unexpected unrecognised mod: %+v", mods[0])
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for corrupt jar")
	}
}

func TestMetadataLessJar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.jar")
	testutil.BuildTextJar(t, path, map[string]string{"README.txt": "hello"})
	mods, warnings := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Recognized {
		t.Fatalf("expected one unrecognised mod: %+v", mods)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for metadata-less jar")
	}
}

func TestJarInJarIgnoredByDefault(t *testing.T) {
	inner := testutil.JarBytes(t, map[string][]byte{
		"fabric.mod.json": []byte(`{"id":"inner","name":"Inner","version":"1"}`),
	})
	path := filepath.Join(t.TempDir(), "outer.jar")
	testutil.BuildJar(t, path, map[string][]byte{
		"META-INF/jarjar/inner.jar": inner,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Recognized {
		t.Fatalf("expected outer jar to be unrecognised by default: %+v", mods)
	}

	mods, _ = ParseFile(path, Options{IncludeNested: true})
	if len(mods) != 1 {
		t.Fatalf("expected one nested mod, got %d: %+v", len(mods), mods)
	}
	if mods[0].Status != model.StatusNested || mods[0].Loader != model.LoaderFabric || mods[0].ID != "inner" {
		t.Fatalf("unexpected nested mod: %+v", mods[0])
	}
}

func TestDisabledJarStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disabled.jar.disabled")
	testutil.BuildTextJar(t, path, map[string]string{
		"fabric.mod.json": `{"id":"disabled-example","name":"Disabled Example","version":"1"}`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Status != model.StatusDisabled {
		t.Fatalf("expected disabled status: %+v", mods)
	}
	if mods[0].Jar != "disabled.jar.disabled" {
		t.Fatalf("unexpected jar name: %q", mods[0].Jar)
	}
}

func TestVersionPlaceholderUnknownWithoutManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.jar")
	testutil.BuildTextJar(t, path, map[string]string{
		"META-INF/mods.toml": `modLoader = "javafml"
[[mods]]
modId = "forge-example"
version = "${file.jarVersion}"
displayName = "Forge Example"
`,
	})
	mods, _ := ParseFile(path, Options{})
	if len(mods) != 1 || mods[0].Version != "unknown" {
		t.Fatalf("expected unknown version, got %+v", mods)
	}
}
