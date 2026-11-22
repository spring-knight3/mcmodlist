package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mcmodlist-test-config")
	if err != nil {
		os.Exit(m.Run())
	}
	_ = os.Setenv("XDG_CONFIG_HOME", dir)
	_ = os.Setenv("APPDATA", dir)
	_ = os.Setenv("HOME", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func newInstance(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func addFabricJar(t *testing.T, root, filename string) {
	t.Helper()
	testutil.BuildTextJar(t, filepath.Join(root, "mods", filename), map[string]string{
		"fabric.mod.json": `{"id":"fabric-mod","name":"Fabric Mod","version":"1.0.0","authors":["Alice"]}`,
	})
}

func run(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHelpAndVersion(t *testing.T) {
	code, out, _ := run("--help")
	if code != 0 || !strings.Contains(out, "Examples:") {
		t.Fatalf("help failed: code=%d out=%q", code, out)
	}
	code, out, _ = run("-V")
	if code != 0 || out != "mcmodlist dev\n" {
		t.Fatalf("version failed: code=%d out=%q", code, out)
	}
}

func TestVersionFlagReflectsVariable(t *testing.T) {
	old := Version
	Version = "9.9.9-test"
	defer func() { Version = old }()
	code, out, _ := run("--version")
	if code != 0 || out != "mcmodlist 9.9.9-test\n" {
		t.Fatalf("injected version failed: code=%d out=%q", code, out)
	}
}

func TestHelpInstanceCheck(t *testing.T) {
	root := newInstance(t)
	addFabricJar(t, root, "fabric.jar")

	code, out, _ := run(root, "--help")
	if code != 0 {
		t.Fatalf("help with instance failed: %d", code)
	}
	if !strings.HasPrefix(out, "This looks like a valid instance.\n") {
		t.Fatalf("expected valid-instance line first, got %q", out)
	}

	code, out, _ = run(root, "--help", "--color", "always")
	if code != 0 || !strings.Contains(out, "\x1b[32mThis looks like a valid instance.\x1b[0m") {
		t.Fatalf("expected green valid-instance line, got %d %q", code, out)
	}

	missing := filepath.Join(t.TempDir(), "missing")
	code, out, _ = run(missing, "--help", "--color", "always")
	if code != 0 || !strings.Contains(out, "\x1b[31mThis does not look like a valid instance.\x1b[0m") {
		t.Fatalf("expected red invalid-instance line, got %d %q", code, out)
	}

	code, out, _ = run("--help")
	if code != 0 || strings.Contains(out, "valid instance") {
		t.Fatalf("help without instance should not check, got %d %q", code, out)
	}
}

func TestInvalidInstanceAndModsFolder(t *testing.T) {
	code, _, stderr := run(filepath.Join(t.TempDir(), "missing"))
	if code != 1 || !strings.Contains(stderr, "not a directory") {
		t.Fatalf("expected exit 1 not-a-directory, got %d %q", code, stderr)
	}

	root := t.TempDir()
	mods := filepath.Join(root, "instance", "mods")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(mods, "a.jar"), []byte("x"))
	code, _, stderr = run(mods)
	if code != 1 || !strings.Contains(stderr, "mods folder") || !strings.Contains(stderr, "try: mcmodlist") {
		t.Fatalf("expected mods-folder suggestion, got %d %q", code, stderr)
	}
}

func TestBadFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--bogus"},
		{"--format", "yaml"},
		{"--color", "blue"},
		{"--no-summary", "--summary-only"},
		{"one", "two"},
	} {
		code, _, _ := run(args...)
		if code != 2 {
			t.Fatalf("args %v: expected exit 2, got %d", args, code)
		}
	}
}

func TestFormatsAndFlagsAfterInstance(t *testing.T) {
	root := newInstance(t)
	addFabricJar(t, root, "fabric.jar")
	for _, format := range []string{"pretty", "markdown", "csv", "json", "html", "text"} {
		code, out, stderr := run(root, "--format", format)
		if code != 0 {
			t.Fatalf("format %s failed: %d %s", format, code, stderr)
		}
		if out == "" {
			t.Fatalf("format %s produced empty output", format)
		}
		if strings.Contains(out, "\x1b[") {
			t.Fatalf("format %s contains ANSI codes when piped", format)
		}
	}

	code, out, _ := run(root, "--format", "json")
	if code != 0 {
		t.Fatalf("json failed: %d", code)
	}
	var doc struct {
		Instance struct {
			Name string `json:"name"`
		} `json:"instance"`
		Counts struct {
			Fabric int `json:"fabric"`
			Total  int `json:"total"`
		} `json:"counts"`
		Mods []struct {
			Name   string `json:"name"`
			Loader string `json:"loader"`
		} `json:"mods"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out)
	}
	if doc.Instance.Name != filepath.Base(root) || doc.Counts.Fabric != 1 || doc.Counts.Total != 1 {
		t.Fatalf("unexpected json doc: %+v", doc)
	}
	if len(doc.Mods) != 1 || doc.Mods[0].Loader != "fabric" {
		t.Fatalf("unexpected json mods: %+v", doc.Mods)
	}
}

func TestSummaryFlags(t *testing.T) {
	root := newInstance(t)
	addFabricJar(t, root, "fabric.jar")

	code, out, _ := run(root, "--summary-only")
	if code != 0 || !strings.Contains(out, "Instance:") || strings.Contains(out, "Fabric Mod") {
		t.Fatalf("summary-only failed: %d %q", code, out)
	}
	code, out, _ = run(root, "--no-summary")
	if code != 0 || strings.Contains(out, "Instance:") || !strings.Contains(out, "Fabric Mod") {
		t.Fatalf("no-summary failed: %d %q", code, out)
	}
	code, out, _ = run(root, "--no-header")
	if code != 0 || strings.Contains(out, "Instance:") || strings.Contains(out, "Name") || !strings.Contains(out, "Fabric Mod") {
		t.Fatalf("no-header failed: %d %q", code, out)
	}
}

func TestOutputFileForce(t *testing.T) {
	root := newInstance(t)
	addFabricJar(t, root, "fabric.jar")
	outPath := filepath.Join(t.TempDir(), "mods.json")

	code, stdout, stderr := run(root, "--format", "json", "--output", outPath)
	if code != 0 || stdout != "" {
		t.Fatalf("output write failed: %d stdout=%q stderr=%s", code, stdout, stderr)
	}
	data, err := os.ReadFile(outPath)
	if err != nil || !strings.Contains(string(data), "fabric-mod") {
		t.Fatalf("output file missing content: %v %q", err, data)
	}
	code, _, stderr = run(root, "--format", "json", "--output", outPath)
	if code != 2 || !strings.Contains(stderr, "--force") {
		t.Fatalf("expected force error, got %d %q", code, stderr)
	}
	code, _, _ = run(root, "--format", "json", "--output", outPath, "--force")
	if code != 0 {
		t.Fatalf("force overwrite failed: %d", code)
	}
}

func TestWarningsAndQuietVerbose(t *testing.T) {
	root := newInstance(t)
	testutil.WriteFile(t, filepath.Join(root, "mods", "corrupt.jar"), []byte("not a zip"))

	code, _, stderr := run(root, "--format", "json")
	if code != 0 {
		t.Fatalf("scan failed: %d %s", code, stderr)
	}
	if !strings.Contains(stderr, "warning") {
		t.Fatalf("expected warning on stderr, got %q", stderr)
	}
	code, _, stderr = run(root, "--format", "json", "--quiet")
	if code != 0 || stderr != "" {
		t.Fatalf("quiet failed: %d %q", code, stderr)
	}
	code, _, stderr = run(root, "--format", "json", "--verbose")
	if code != 0 || !strings.Contains(stderr, "scanned") {
		t.Fatalf("verbose failed: %d %q", code, stderr)
	}
}

type fakeClipboard struct {
	text string
	err  error
}

func (f *fakeClipboard) Copy(text string) error {
	f.text = text
	return f.err
}

func useClipboard(t *testing.T, cb Clipboard) {
	t.Helper()
	old := clipboard
	clipboard = cb
	t.Cleanup(func() { clipboard = old })
}

func useTTY(t *testing.T, tty bool) {
	t.Helper()
	oldOut := isTTY
	oldIn := isTTYInput
	isTTY = func(io.Writer) bool { return tty }
	isTTYInput = func(io.Reader) bool { return tty }
	t.Cleanup(func() { isTTY = oldOut; isTTYInput = oldIn })
}

func useListInstances(t *testing.T, instances []model.Instance, err error) {
	t.Helper()
	old := listInstances
	listInstances = func() ([]model.Instance, error) { return instances, err }
	t.Cleanup(func() { listInstances = old })
}

func runIO(args []string, stdin string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := RunIO(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func addFabricMod(t *testing.T, root, filename, name, id, version, author, side string) {
	t.Helper()
	metadata := `{"id":"` + id + `","name":"` + name + `","version":"` + version + `","authors":["` + author + `"]`
	if side != "" {
		metadata += `,"side":"` + side + `"`
	}
	metadata += `}`
	testutil.BuildTextJar(t, filepath.Join(root, "mods", filename), map[string]string{"fabric.mod.json": metadata})
}

func decodeMods(t *testing.T, out string) struct {
	Counts struct {
		Fabric       int `json:"fabric"`
		Total        int `json:"total"`
		Disabled     int `json:"disabled"`
		Unrecognised int `json:"unrecognised"`
	} `json:"counts"`
	Mods []struct {
		Name    string `json:"name"`
		ID      string `json:"id"`
		Version string `json:"version"`
		Jar     string `json:"jar"`
		Loader  string `json:"loader"`
		Side    string `json:"side"`
		Status  string `json:"status"`
	} `json:"mods"`
} {
	t.Helper()
	var doc struct {
		Counts struct {
			Fabric       int `json:"fabric"`
			Total        int `json:"total"`
			Disabled     int `json:"disabled"`
			Unrecognised int `json:"unrecognised"`
		} `json:"counts"`
		Mods []struct {
			Name    string `json:"name"`
			ID      string `json:"id"`
			Version string `json:"version"`
			Jar     string `json:"jar"`
			Loader  string `json:"loader"`
			Side    string `json:"side"`
			Status  string `json:"status"`
		} `json:"mods"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out)
	}
	return doc
}

func modIDs(doc struct {
	Counts struct {
		Fabric       int `json:"fabric"`
		Total        int `json:"total"`
		Disabled     int `json:"disabled"`
		Unrecognised int `json:"unrecognised"`
	} `json:"counts"`
	Mods []struct {
		Name    string `json:"name"`
		ID      string `json:"id"`
		Version string `json:"version"`
		Jar     string `json:"jar"`
		Loader  string `json:"loader"`
		Side    string `json:"side"`
		Status  string `json:"status"`
	} `json:"mods"`
}) []string {
	ids := make([]string, 0, len(doc.Mods))
	for _, mod := range doc.Mods {
		ids = append(ids, mod.ID)
	}
	return ids
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func TestIncludeDisabledAndUnrecognised(t *testing.T) {
	root := newInstance(t)
	addFabricMod(t, root, "alpha.jar", "Alpha Mod", "alpha", "1.0.0", "Alice", "client")
	addFabricMod(t, root, "disabled.jar.disabled", "Disabled Mod", "disabled-mod", "1.0.0", "Dave", "")
	testutil.WriteFile(t, filepath.Join(root, "mods", "corrupt.jar"), []byte("not a zip"))
	testutil.BuildTextJar(t, filepath.Join(root, "mods", "empty.jar"), map[string]string{"README.txt": "no metadata"})

	code, out, stderr := run(root, "--format", "json")
	if code != 0 {
		t.Fatalf("default scan failed: %d %s", code, stderr)
	}
	doc := decodeMods(t, out)
	ids := modIDs(doc)
	if containsID(ids, "disabled-mod") {
		t.Fatalf("disabled mod should be hidden by default: %+v", ids)
	}
	if doc.Counts.Total != 3 || doc.Counts.Unrecognised != 2 {
		t.Fatalf("unexpected default counts: %+v", doc.Counts)
	}

	code, out, stderr = run(root, "--format", "json", "--include-disabled")
	if code != 0 {
		t.Fatalf("include-disabled failed: %d %s", code, stderr)
	}
	doc = decodeMods(t, out)
	if !containsID(modIDs(doc), "disabled-mod") || doc.Counts.Disabled != 1 || doc.Counts.Total != 4 {
		t.Fatalf("include-disabled failed: %+v", doc)
	}

	code, out, stderr = run(root, "--format", "json", "--unrecognised")
	if code != 0 {
		t.Fatalf("unrecognised filter failed: %d %s", code, stderr)
	}
	doc = decodeMods(t, out)
	if len(doc.Mods) != 2 || doc.Counts.Unrecognised != 2 {
		t.Fatalf("unrecognised filter failed: %+v", doc)
	}
	for _, mod := range doc.Mods {
		if mod.Loader != "" {
			t.Fatalf("unrecognised filter returned recognised mod: %+v", mod)
		}
	}
}

func TestFiltersAndCountsReflectFilters(t *testing.T) {
	root := newInstance(t)
	addFabricMod(t, root, "alpha.jar", "Alpha Mod", "alpha", "1.0.0", "Alice", "client")
	testutil.BuildTextJar(t, filepath.Join(root, "mods", "beta.jar"), map[string]string{
		"quilt.mod.json": `{"id":"beta","name":"Beta Mod","version":"2.0.0","authors":["Bob"],"side":"server"}`,
	})
	testutil.BuildTextJar(t, filepath.Join(root, "mods", "gamma.jar"), map[string]string{
		"META-INF/mods.toml": "modLoader = \"javafml\"\n[[mods]]\nmodId = \"gamma\"\nversion = \"3.0.0\"\ndisplayName = \"Gamma Mod\"\nauthors = \"Carol\"\n",
	})

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"search", []string{"--search", "gamma"}, []string{"gamma"}},
		{"loader", []string{"--loader", "fabric"}, []string{"alpha"}},
		{"side", []string{"--side", "server"}, []string{"beta"}},
		{"author", []string{"--author", "bob"}, []string{"beta"}},
	}
	for _, tc := range cases {
		args := append([]string{root, "--format", "json"}, tc.args...)
		code, out, stderr := run(args...)
		if code != 0 {
			t.Fatalf("%s failed: %d %s", tc.name, code, stderr)
		}
		doc := decodeMods(t, out)
		ids := modIDs(doc)
		if len(ids) != len(tc.want) || !containsID(ids, tc.want[0]) {
			t.Fatalf("%s: got ids %v want %v", tc.name, ids, tc.want)
		}
		if doc.Counts.Total != len(tc.want) {
			t.Fatalf("%s: counts should reflect filters, got %+v", tc.name, doc.Counts)
		}
	}
}

func TestSortColumnsGroupWideTruncate(t *testing.T) {
	root := newInstance(t)
	addFabricMod(t, root, "alpha.jar", "Alpha Mod", "alpha", "1.0.0", "Alice", "client")
	addFabricMod(t, root, "beta.jar", "Beta Mod", "beta", "2.0.0", "Bob", "server")
	addFabricMod(t, root, "long.jar", "A Very Long Mod Name For Truncation Testing", "long", "1.0.0", "Long", "client")

	code, out, stderr := run(root, "--format", "text", "--sort", "-name", "--columns", "name,id")
	if code != 0 {
		t.Fatalf("sort/columns failed: %d %s", code, stderr)
	}
	if strings.Index(out, "Beta Mod") > strings.Index(out, "Alpha Mod") {
		t.Fatalf("--sort -name did not reverse order: %q", out)
	}
	if strings.Contains(out, "Version") {
		t.Fatalf("--columns name,id should omit Version: %q", out)
	}

	code, out, stderr = run(root, "--format", "text", "--columns", "name,id", "--group-by", "side")
	if code != 0 {
		t.Fatalf("group-by failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "Side: client") || !strings.Contains(out, "Side: server") {
		t.Fatalf("group-by side missing headings: %q", out)
	}

	code, out, stderr = run(root, "--format", "text", "--columns", "name", "--truncate", "5")
	if code != 0 {
		t.Fatalf("truncate failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "A Ve…") {
		t.Fatalf("truncate did not shorten long name: %q", out)
	}

	code, out, stderr = run(root, "--format", "pretty")
	if code != 0 {
		t.Fatalf("pretty failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "…") || strings.Contains(out, "A Very Long Mod Name For Truncation Testing") {
		t.Fatalf("expected default pretty truncation: %q", out)
	}
	code, out, stderr = run(root, "--format", "pretty", "--wide")
	if code != 0 {
		t.Fatalf("wide failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "A Very Long Mod Name For Truncation Testing") {
		t.Fatalf("--wide should not truncate: %q", out)
	}

	code, _, stderr = run(root, "--format", "json", "--jobs", "2")
	if code != 0 {
		t.Fatalf("jobs failed: %d %s", code, stderr)
	}
}

func TestRecursiveAndIncludeNested(t *testing.T) {
	root := newInstance(t)
	if err := os.MkdirAll(filepath.Join(root, "mods", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.BuildTextJar(t, filepath.Join(root, "mods", "sub", "nested.jar"), map[string]string{
		"fabric.mod.json": `{"id":"recursive-mod","name":"Recursive Mod","version":"1"}`,
	})
	inner := testutil.JarBytes(t, testutil.TextEntries(map[string]string{
		"fabric.mod.json": `{"id":"inner-mod","name":"Inner Mod","version":"1"}`,
	}))
	testutil.BuildJar(t, filepath.Join(root, "mods", "outer.jar"), map[string][]byte{
		"fabric.mod.json": []byte(`{"id":"outer-mod","name":"Outer Mod","version":"1"}`),
		"libs/inner.jar":  inner,
	})

	code, out, stderr := run(root, "--format", "json")
	if code != 0 {
		t.Fatalf("scan failed: %d %s", code, stderr)
	}
	if strings.Contains(out, "recursive-mod") || strings.Contains(out, "inner-mod") {
		t.Fatalf("default scan should hide nested/recursive mods: %s", out)
	}

	code, out, stderr = run(root, "--format", "json", "--recursive")
	if code != 0 {
		t.Fatalf("recursive failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "sub/nested.jar") {
		t.Fatalf("recursive output missing sub path: %s", out)
	}

	code, out, stderr = run(root, "--format", "json", "--include-nested")
	if code != 0 {
		t.Fatalf("include-nested failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "inner-mod") || !strings.Contains(out, `"status":"nested"`) && !strings.Contains(out, `"status": "nested"`) {
		t.Fatalf("include-nested output missing nested mod: %s", out)
	}
}

func TestCopyListPickerAndTypoHelp(t *testing.T) {
	root := newInstance(t)
	addFabricMod(t, root, "alpha.jar", "Alpha Mod", "alpha", "1.0.0", "Alice", "client")

	cb := &fakeClipboard{}
	useClipboard(t, cb)
	code, out, stderr := run(root, "--format", "markdown", "--copy")
	if code != 0 {
		t.Fatalf("copy failed: %d %s", code, stderr)
	}
	if out != "" {
		t.Fatalf("--copy should not write stdout, got %q", out)
	}
	if !strings.Contains(cb.text, "| Name |") || !strings.Contains(cb.text, "Alpha Mod") {
		t.Fatalf("clipboard missing markdown table: %q", cb.text)
	}
	cb.err = io.ErrClosedPipe
	code, _, stderr = run(root, "--format", "markdown", "--copy")
	if code != 1 || !strings.Contains(stderr, "clipboard unavailable") {
		t.Fatalf("expected clipboard error, got %d %q", code, stderr)
	}

	useListInstances(t, []model.Instance{
		{Name: "Pack A", Path: "/tmp/pack-a", Launcher: "prism"},
		{Name: "Pack B", Path: "/tmp/pack-b", Launcher: "curseforge"},
	}, nil)
	code, out, _ = run("--list-instances")
	if code != 0 || !strings.Contains(out, "/tmp/pack-a") || !strings.Contains(out, "/tmp/pack-b") {
		t.Fatalf("list-instances failed: %d %q", code, out)
	}
	code, out, _ = run("--list-instances", "--launcher", "prism")
	if code != 0 || !strings.Contains(out, "/tmp/pack-a") || strings.Contains(out, "/tmp/pack-b") {
		t.Fatalf("list-instances launcher filter failed: %d %q", code, out)
	}

	useTTY(t, true)
	useListInstances(t, []model.Instance{{Name: "Picked", Path: root, Launcher: "prism"}}, nil)
	code, out, stderr = runIO([]string{"--format", "json", "--color", "never"}, "1\n")
	if code != 0 {
		t.Fatalf("picker failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, `"id":"alpha"`) && !strings.Contains(out, `"id": "alpha"`) {
		t.Fatalf("picker did not scan selected instance: %s", out)
	}

	code, _, stderr = run("--formt", "json")
	if code != 2 || !strings.Contains(stderr, "did you mean") {
		t.Fatalf("expected flag suggestion, got %d %q", code, stderr)
	}
	code, _, stderr = run("--format", "jsn")
	if code != 2 || !strings.Contains(stderr, "did you mean") {
		t.Fatalf("expected format suggestion, got %d %q", code, stderr)
	}
}

func TestCheckFlag(t *testing.T) {
	clean := newInstance(t)
	addFabricMod(t, clean, "clean.jar", "Clean Mod", "clean-mod", "1.0.0", "Alice", "")
	code, out, stderr := run(clean, "--check", "--format", "json")
	if code != 0 {
		t.Fatalf("clean --check failed: %d %s", code, stderr)
	}
	if strings.Contains(stderr, "issue[") {
		t.Fatalf("clean --check printed issues: %q", stderr)
	}
	if !strings.Contains(out, `"issues": []`) && !strings.Contains(out, `"issues":[]`) {
		t.Fatalf("clean --check should emit empty issues, got %s", out)
	}

	dup := newInstance(t)
	addFabricMod(t, dup, "one.jar", "Dup One", "dup-mod", "1.0.0", "Alice", "")
	addFabricMod(t, dup, "two.jar", "Dup Two", "dup-mod", "2.0.0", "Bob", "")
	code, out, stderr = run(dup, "--check", "--format", "json")
	if code != 3 {
		t.Fatalf("expected --check exit 3, got %d stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "issue[duplicate-id]") {
		t.Fatalf("expected duplicate-id issue on stderr, got %q", stderr)
	}
	if !strings.Contains(out, `"type": "duplicate-id"`) && !strings.Contains(out, `"type":"duplicate-id"`) {
		t.Fatalf("expected duplicate-id in JSON issues, got %s", out)
	}
}

type diffModJSON struct {
	ID         string `json:"id"`
	Change     string `json:"change"`
	OldVersion string `json:"oldVersion"`
}

func decodeDiffMods(t *testing.T, out string) map[string]diffModJSON {
	t.Helper()
	var doc struct {
		Mods []diffModJSON `json:"mods"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out)
	}
	byID := make(map[string]diffModJSON, len(doc.Mods))
	for _, mod := range doc.Mods {
		byID[mod.ID] = mod
	}
	return byID
}

func TestDiffSymmetric(t *testing.T) {
	oldInst := newInstance(t)
	addFabricMod(t, oldInst, "shared.jar", "Shared Mod", "shared", "1.0.0", "Alice", "")
	addFabricMod(t, oldInst, "removed.jar", "Removed Mod", "removed", "1.0.0", "Bob", "")
	addFabricMod(t, oldInst, "changed.jar", "Changed Mod", "changed", "1.0.0", "Carol", "")

	newInst := newInstance(t)
	addFabricMod(t, newInst, "shared.jar", "Shared Mod", "shared", "1.0.0", "Alice", "")
	addFabricMod(t, newInst, "added.jar", "Added Mod", "added", "1.0.0", "Dave", "")
	addFabricMod(t, newInst, "changed.jar", "Changed Mod", "changed", "2.0.0", "Carol", "")

	code, out, stderr := run(oldInst, "--diff", newInst, "--format", "json")
	if code != 0 {
		t.Fatalf("forward diff failed: %d %s", code, stderr)
	}
	forward := decodeDiffMods(t, out)
	if forward["added"].Change != "added" || forward["removed"].Change != "removed" {
		t.Fatalf("forward diff added/removed wrong: %+v", forward)
	}
	if forward["changed"].Change != "version-changed" || forward["changed"].OldVersion != "1.0.0" {
		t.Fatalf("forward diff version change wrong: %+v", forward["changed"])
	}
	if _, ok := forward["shared"]; ok {
		t.Fatalf("unchanged mod should not appear in diff: %+v", forward)
	}

	code, out, stderr = run(newInst, "--diff", oldInst, "--format", "json")
	if code != 0 {
		t.Fatalf("reverse diff failed: %d %s", code, stderr)
	}
	reverse := decodeDiffMods(t, out)
	if reverse["added"].Change != "removed" || reverse["removed"].Change != "added" {
		t.Fatalf("reverse diff added/removed did not swap: %+v", reverse)
	}
	if reverse["changed"].Change != "version-changed" || reverse["changed"].OldVersion != "2.0.0" {
		t.Fatalf("reverse diff version change wrong: %+v", reverse["changed"])
	}
}

func setConfigHome(t *testing.T, dir string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", dir)
	case "darwin":
		t.Setenv("HOME", dir)
	default:
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "mcmodlist")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConfigPrecedenceAndMalformed(t *testing.T) {
	root := newInstance(t)
	addFabricMod(t, root, "alpha.jar", "Alpha Mod", "alpha", "1.0.0", "Alice", "")

	setConfigHome(t, t.TempDir())
	writeConfig(t, "format = \"pretty\"\ncolumns = \"name\"\ncolor = \"always\"\n")

	code, out, stderr := run(root)
	if code != 0 {
		t.Fatalf("config defaults failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "\x1b[1mInstance:") {
		t.Fatalf("config color=always should colour summary when piped: %q", out)
	}
	if strings.Contains(out, "ID") || strings.Contains(out, "alpha.jar") {
		t.Fatalf("config columns=name should omit other columns: %q", out)
	}

	code, out, stderr = run(root, "--format", "json")
	if code != 0 || !strings.Contains(out, `"id": "alpha"`) && !strings.Contains(out, `"id":"alpha"`) {
		t.Fatalf("--format should override config format: %d %s %s", code, out, stderr)
	}

	code, out, stderr = run(root, "--color", "never")
	if code != 0 || strings.Contains(out, "\x1b[") {
		t.Fatalf("--color never should override config color: %d %q %s", code, out, stderr)
	}

	writeConfig(t, "format = \n")
	code, out, stderr = run(root, "--format", "json")
	if code != 0 {
		t.Fatalf("malformed config should not fail: %d %s", code, stderr)
	}
	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "malformed config") {
		t.Fatalf("expected malformed config warning, got %q", stderr)
	}
	if !strings.Contains(out, `"id": "alpha"`) && !strings.Contains(out, `"id":"alpha"`) {
		t.Fatalf("malformed config should still render output: %s", out)
	}
}

func TestCompletionScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		code, out, stderr := run("completion", shell)
		if code != 0 {
			t.Fatalf("completion %s failed: %d %s", shell, code, stderr)
		}
		if !strings.Contains(out, "mcmodlist") {
			t.Fatalf("completion %s missing mcmodlist reference: %q", shell, out)
		}
	}
	for _, args := range [][]string{
		{"completion"},
		{"completion", "nushell"},
		{"completion", "bash", "extra"},
	} {
		code, _, stderr := run(args...)
		if code != 2 {
			t.Fatalf("args %v: expected exit 2, got %d stderr=%q", args, code, stderr)
		}
	}
}

func TestGUIHintWhenNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	code, _, stderr := run("--gui")
	if code != 1 || !strings.Contains(stderr, "GUI is not installed") {
		t.Fatalf("expected GUI install hint, got %d %q", code, stderr)
	}
}
