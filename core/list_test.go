package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListInstancesUnder(t *testing.T) {
	home := t.TempDir()
	pack := filepath.Join(home, ".local", "share", "PrismLauncher", "instances", "pack")
	if err := os.MkdirAll(filepath.Join(pack, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(home, ".local", "share", "PrismLauncher", "instances", "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	instances := ListInstancesUnder(home)
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %+v", instances)
	}
	if instances[0].Name != "pack" || instances[0].Path != pack {
		t.Fatalf("unexpected instance: %+v", instances[0])
	}
}
