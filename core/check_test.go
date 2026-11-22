package core

import (
	"path/filepath"
	"testing"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func writeModrinthProfile(t *testing.T, root, loader, minecraft string) {
	t.Helper()
	testutil.WriteTextFile(t, filepath.Join(root, "profile.json"),
		`{"name":"Check Instance","loader":"`+loader+`","game_version":"`+minecraft+`"}`)
}

func checkIssueTypes(issues []model.Issue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.Type)
	}
	return out
}

func requireOnlyIssue(t *testing.T, report Report, want string) model.Issue {
	t.Helper()
	issues := Check(report)
	if len(issues) != 1 {
		t.Fatalf("expected exactly one %s issue, got %+v", want, issues)
	}
	if issues[0].Type != want {
		t.Fatalf("expected issue type %q, got %+v", want, issues)
	}
	return issues[0]
}

func TestCheckDuplicateID(t *testing.T) {
	root := newInstance(t)
	writeModrinthProfile(t, root, "fabric", "1.20.1")
	addJar(t, root, "alpha.jar", map[string]string{
		"fabric.mod.json": `{"id":"same-mod","name":"Same A","version":"1.0.0"}`,
	})
	addJar(t, root, "beta.jar", map[string]string{
		"fabric.mod.json": `{"id":"same-mod","name":"Same B","version":"2.0.0"}`,
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	issue := requireOnlyIssue(t, report, "duplicate-id")
	if issue.Mod != "same-mod" {
		t.Fatalf("unexpected issue mod: %+v", issue)
	}
}

func TestCheckLoaderMismatch(t *testing.T) {
	root := newInstance(t)
	writeModrinthProfile(t, root, "fabric", "1.20.1")
	addJar(t, root, "fabric.jar", map[string]string{
		"fabric.mod.json": `{"id":"fabric-mod","name":"Fabric Mod","version":"1.0.0"}`,
	})
	addJar(t, root, "forge.jar", map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"forge-mod\"\nversion = \"1.0.0\"\ndisplayName = \"Forge Mod\"\n",
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	issue := requireOnlyIssue(t, report, "loader-mismatch")
	if issue.Mod != "forge-mod" {
		t.Fatalf("unexpected issue mod: %+v", issue)
	}
}

func TestCheckMissingDependency(t *testing.T) {
	root := newInstance(t)
	writeModrinthProfile(t, root, "fabric", "1.20.1")
	addJar(t, root, "dependent.jar", map[string]string{
		"fabric.mod.json": `{"id":"dependent","name":"Dependent","version":"1.0.0","depends":{"missing-lib":">=1.0.0"}}`,
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	issue := requireOnlyIssue(t, report, "missing-dependency")
	if issue.Mod != "dependent" || issue.Message != `missing required dependency "missing-lib"` {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestCheckMinecraftVersion(t *testing.T) {
	root := newInstance(t)
	writeModrinthProfile(t, root, "fabric", "1.20.1")
	addJar(t, root, "newer.jar", map[string]string{
		"fabric.mod.json": `{"id":"newer-mod","name":"Newer Mod","version":"1.0.0","depends":{"minecraft":">=1.21.0"}}`,
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	issue := requireOnlyIssue(t, report, "minecraft-version")
	if issue.Mod != "newer-mod" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestCheckCleanInstance(t *testing.T) {
	root := newInstance(t)
	writeModrinthProfile(t, root, "fabric", "1.20.1")
	addJar(t, root, "clean.jar", map[string]string{
		"fabric.mod.json": `{"id":"clean-mod","name":"Clean Mod","version":"1.0.0","depends":{"fabricloader":">=0.15.0"}}`,
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if issues := Check(report); len(issues) != 0 {
		t.Fatalf("expected no issues, got %+v", issues)
	}
}
