package core

import (
	"os"
	"path/filepath"
	"testing"

	"mcmodlist/core/testutil"
)

func TestScanRecursiveAndProgress(t *testing.T) {
	root := newInstance(t)
	if err := os.MkdirAll(filepath.Join(root, "mods", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.BuildTextJar(t, filepath.Join(root, "mods", "sub", "nested.jar"), map[string]string{
		"fabric.mod.json": `{"id":"nested-mod","name":"Nested Mod","version":"1"}`,
	})

	report, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Total != 0 {
		t.Fatalf("non-recursive scan should ignore subdirectory, got %+v", report.Mods)
	}

	var calls [][2]int
	report, err = Scan(root, Options{Recursive: true, Jobs: 1, Progress: func(done, total int) {
		calls = append(calls, [2]int{done, total})
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mods) != 1 || report.Mods[0].ID != "nested-mod" || report.Mods[0].Jar != "sub/nested.jar" {
		t.Fatalf("recursive scan failed: %+v", report.Mods)
	}
	if len(calls) < 2 || calls[0] != [2]int{0, 1} || calls[len(calls)-1] != [2]int{1, 1} {
		t.Fatalf("unexpected progress calls: %+v", calls)
	}
}
