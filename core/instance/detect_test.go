package instance

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func TestDetectInvalidPath(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "missing"))
	var detectErr *Error
	if !errors.As(err, &detectErr) || detectErr.Kind != KindNotDirectory {
		t.Fatalf("expected KindNotDirectory, got %v", err)
	}
}

func TestDetectNotInstance(t *testing.T) {
	root := t.TempDir()
	_, err := Detect(root)
	var detectErr *Error
	if !errors.As(err, &detectErr) || detectErr.Kind != KindNotInstance {
		t.Fatalf("expected KindNotInstance, got %v", err)
	}
}

func TestDetectModsFolderSuggestsParent(t *testing.T) {
	root := t.TempDir()
	mods := filepath.Join(root, "instance", "mods")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteTextFile(t, filepath.Join(mods, "example.jar"), "not really a jar")

	_, err := Detect(mods)
	var detectErr *Error
	if !errors.As(err, &detectErr) || detectErr.Kind != KindModsFolder {
		t.Fatalf("expected KindModsFolder, got %v", err)
	}
	if detectErr.Suggestion != filepath.Dir(mods) {
		t.Fatalf("expected suggestion %s, got %s", filepath.Dir(mods), detectErr.Suggestion)
	}
}

func TestDetectPlainInstance(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if det.ModsDir != filepath.Join(root, "mods") {
		t.Fatalf("unexpected mods dir %s", det.ModsDir)
	}
	if det.Info.Name != filepath.Base(root) {
		t.Fatalf("expected directory name fallback, got %q", det.Info.Name)
	}
	if det.Info.Loader != model.LoaderUnknown {
		t.Fatalf("expected unknown loader, got %q", det.Info.Loader)
	}
}

func TestLauncherMetadata(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		launcher  string
		instName  string
		minecraft string
		loader    model.Loader
		loaderVer string
	}{
		{
			name: "prism",
			files: map[string]string{
				"instance.cfg": "[General]\nInstanceType=OneSix\nname=Prism Pack\n",
				"mmc-pack.json": `{"components":[
					{"uid":"net.minecraft","version":"1.20.1"},
					{"uid":"net.fabricmc.fabric-loader","version":"0.15.11"}
				]}`,
			},
			launcher: "prism", instName: "Prism Pack", minecraft: "1.20.1", loader: model.LoaderFabric, loaderVer: "0.15.11",
		},
		{
			name: "curseforge",
			files: map[string]string{
				"minecraftinstance.json": `{"name":"CF Pack","mcGameVersion":"1.19.2","baseModLoader":{"name":"forge","forgeVersion":"47.1.3","minecraftVersion":"1.19.2"}}`,
			},
			launcher: "curseforge", instName: "CF Pack", minecraft: "1.19.2", loader: model.LoaderForge, loaderVer: "47.1.3",
		},
		{
			name: "modrinth",
			files: map[string]string{
				"profile.json": `{"name":"MR Pack","game_version":"1.20.1","loader":"quilt","version_id":"0.23.1"}`,
			},
			launcher: "modrinth", instName: "MR Pack", minecraft: "1.20.1", loader: model.LoaderQuilt, loaderVer: "0.23.1",
		},
		{
			name: "atlauncher",
			files: map[string]string{
				"instance.json": `{"name":"AT Pack","minecraftVersion":"1.18.2","loader":"neoforge","loaderVersion":"20.2.86"}`,
			},
			launcher: "atlauncher", instName: "AT Pack", minecraft: "1.18.2", loader: model.LoaderNeoForge, loaderVer: "20.2.86",
		},
		{
			name: "gdlauncher",
			files: map[string]string{
				"config.json": `{"name":"GD Pack","gameVersion":"1.20.1","loader":"fabric","loaderVersion":"0.16.9"}`,
			},
			launcher: "gdlauncher", instName: "GD Pack", minecraft: "1.20.1", loader: model.LoaderFabric, loaderVer: "0.16.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
				t.Fatal(err)
			}
			for name, contents := range tt.files {
				testutil.WriteTextFile(t, filepath.Join(root, name), contents)
			}
			det, err := Detect(root)
			if err != nil {
				t.Fatal(err)
			}
			if det.Info.Launcher != tt.launcher {
				t.Fatalf("launcher: got %q want %q", det.Info.Launcher, tt.launcher)
			}
			if det.Info.Name != tt.instName {
				t.Fatalf("name: got %q want %q", det.Info.Name, tt.instName)
			}
			if det.Info.Minecraft != tt.minecraft {
				t.Fatalf("minecraft: got %q want %q", det.Info.Minecraft, tt.minecraft)
			}
			if det.Info.Loader != tt.loader {
				t.Fatalf("loader: got %q want %q", det.Info.Loader, tt.loader)
			}
			if det.Info.LoaderVersion != tt.loaderVer {
				t.Fatalf("loader version: got %q want %q", det.Info.LoaderVersion, tt.loaderVer)
			}
		})
	}
}

func TestDetectDotMinecraftMods(t *testing.T) {
	root := t.TempDir()
	mods := filepath.Join(root, ".minecraft", "mods")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if det.ModsDir != mods {
		t.Fatalf("expected .minecraft/mods, got %s", det.ModsDir)
	}
}
