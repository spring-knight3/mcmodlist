package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mcmodlist/core"
	"mcmodlist/core/model"
	"mcmodlist/core/testutil"
	"mcmodlist/engine"
	enginerender "mcmodlist/engine/render"
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

func fabricJar(id, name, version string) map[string]string {
	return map[string]string{
		"fabric.mod.json": fmt.Sprintf(`{"id":%q,"name":%q,"version":%q,"authors":["Alice"],"description":"desc %s","license":"MIT"}`, id, name, version, id),
	}
}

func fixtureInstance(t *testing.T) string {
	t.Helper()
	root := newInstance(t)
	addJar(t, root, "alpha.jar", fabricJar("alpha", "Alpha Mod", "1.0.0"))
	addJar(t, root, "beta.jar", fabricJar("beta", "Beta Mod", "2.0.0"))
	addJar(t, root, "quilt.jar", map[string]string{
		"quilt.mod.json": `{"id":"quilt-mod","name":"Quilt Mod","version":"1.0.0"}`,
	})
	addJar(t, root, "disabled.jar.disabled", fabricJar("disabled", "Disabled Mod", "1.0.0"))
	addJar(t, root, "empty.jar", map[string]string{"README.txt": "no metadata"})
	return root
}

func request(t *testing.T, e *engine.Engine, req engine.Request) engine.Message {
	t.Helper()
	ch := make(chan engine.Message, 8)
	e.Submit(req, func(m engine.Message) { ch <- m })
	select {
	case m := <-ch:
		return m
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for response")
		return engine.Message{}
	}
}

func submitAsync(t *testing.T, e *engine.Engine, req engine.Request) <-chan engine.Message {
	t.Helper()
	ch := make(chan engine.Message, 4096)
	e.Submit(req, func(m engine.Message) { ch <- m })
	return ch
}

func waitResponse(t *testing.T, ch <-chan engine.Message, id uint64) engine.Message {
	t.Helper()
	deadline := time.After(30 * time.Second)
	for {
		select {
		case m := <-ch:
			if m.ID == id && m.Event == "" {
				return m
			}
		case <-deadline:
			t.Fatalf("timed out waiting for response id %d", id)
			return engine.Message{}
		}
	}
}

func mustResult(t *testing.T, m engine.Message) json.RawMessage {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("unexpected error %+v", m.Error)
	}
	if m.Event != "" {
		t.Fatalf("unexpected event %s", m.Event)
	}
	return m.Result
}

func mustError(t *testing.T, m engine.Message, code string) engine.ErrorBody {
	t.Helper()
	if m.Error == nil {
		t.Fatalf("expected error, got %+v", m)
	}
	if m.Error.Code != code {
		t.Fatalf("error code %q, want %q", m.Error.Code, code)
	}
	return *m.Error
}

func scanInstance(t *testing.T, e *engine.Engine, root string) string {
	t.Helper()
	params, _ := json.Marshal(map[string]string{"path": root})
	ch := submitAsync(t, e, engine.Request{V: engine.ProtocolVersion, ID: 1, Method: "scan", Params: params})
	m := waitResponse(t, ch, 1)
	var res struct {
		ScanID  string `json:"scanId"`
		Summary struct {
			Total int `json:"total"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(mustResult(t, m), &res); err != nil {
		t.Fatal(err)
	}
	if res.ScanID == "" {
		t.Fatal("empty scanId")
	}
	return res.ScanID
}

func TestHelloUnknownAndVersion(t *testing.T) {
	e := engine.New(engine.Options{})
	defer e.Close()

	m := request(t, e, engine.Request{V: 1, ID: 1, Method: "hello"})
	var res struct {
		Protocol int      `json:"protocol"`
		Methods  []string `json:"methods"`
	}
	if err := json.Unmarshal(mustResult(t, m), &res); err != nil {
		t.Fatal(err)
	}
	if res.Protocol != engine.ProtocolVersion || len(res.Methods) != len(engine.Methods) {
		t.Fatalf("bad hello result: %+v", res)
	}

	mustError(t, request(t, e, engine.Request{V: 1, ID: 2, Method: "bogus"}), engine.CodeInvalidRequest)
	mustError(t, request(t, e, engine.Request{V: 2, ID: 3, Method: "hello"}), engine.CodeUnsupportedVersion)
}

func TestScanViewDetailsCheckExport(t *testing.T) {
	root := fixtureInstance(t)
	report, err := core.Scan(root, core.Options{IncludeNested: true})
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(engine.Options{})
	defer e.Close()
	scanID := scanInstance(t, e, root)

	viewReq := engine.Request{V: 1, ID: 2, Method: "view", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{}}`, scanID))}
	m := request(t, e, viewReq)
	var vr struct {
		State engine.State `json:"state"`
	}
	if err := json.Unmarshal(mustResult(t, m), &vr); err != nil {
		t.Fatal(err)
	}
	state := vr.State

	wantMods := core.FilterMods(report.Mods, core.FilterOptions{})
	if len(state.Rows) != len(wantMods) {
		t.Fatalf("rows %d, want %d", len(state.Rows), len(wantMods))
	}
	for i, row := range state.Rows {
		mod := wantMods[i]
		if row.RowID == "" || row.Name != mod.Name || row.ID != mod.ID || row.Jar != mod.Jar {
			t.Fatalf("row %d mismatch: %+v vs %+v", i, row, mod)
		}
	}

	counts := core.CountMods(wantMods)
	wantLabels := core.OrderedLoaderLabels(counts)
	if len(state.Loaders) != len(wantLabels) {
		t.Fatalf("loaders %v, want %v", state.Loaders, wantLabels)
	}
	for i, label := range wantLabels {
		if state.Loaders[i].Label != label || state.Loaders[i].Count != counts.Loaders[label] {
			t.Fatalf("loader %d mismatch: %+v want %s=%d", i, state.Loaders[i], label, counts.Loaders[label])
		}
	}
	if len(state.LoaderOptions) < 2 {
		t.Fatalf("loaderOptions should keep fabric+quilt while filtered later: %v", state.LoaderOptions)
	}

	filtered := engine.Request{V: 1, ID: 3, Method: "view", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{"loader":"fabric","search":"alpha"}}`, scanID))}
	fm := request(t, e, filtered)
	var fv struct {
		State engine.State `json:"state"`
	}
	if err := json.Unmarshal(mustResult(t, fm), &fv); err != nil {
		t.Fatal(err)
	}
	if len(fv.State.LoaderOptions) != len(state.LoaderOptions) {
		t.Fatalf("loaderOptions changed under filter: %v vs %v", fv.State.LoaderOptions, state.LoaderOptions)
	}
	if len(fv.State.Rows) != 1 || fv.State.Rows[0].ID != "alpha" {
		t.Fatalf("filtered rows: %+v", fv.State.Rows)
	}

	rowID := state.Rows[0].RowID
	detParams, _ := json.Marshal(map[string]string{"scanId": scanID, "rowId": rowID})
	det := engine.Request{V: 1, ID: 4, Method: "details", Params: detParams}
	dm := request(t, e, det)
	var detRes engine.Details
	if err := json.Unmarshal(mustResult(t, dm), &detRes); err != nil {
		t.Fatal(err)
	}
	if detRes.Mod.RowID != rowID || detRes.Mod.Name == "" {
		t.Fatalf("bad details: %+v", detRes)
	}

	chk := engine.Request{V: 1, ID: 5, Method: "check", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q}`, scanID))}
	cm := request(t, e, chk)
	var cr struct {
		Issues []model.Issue `json:"issues"`
	}
	if err := json.Unmarshal(mustResult(t, cm), &cr); err != nil {
		t.Fatal(err)
	}
	if len(cr.Issues) != len(core.Check(report)) {
		t.Fatalf("issues %d, want %d", len(cr.Issues), len(core.Check(report)))
	}

	exp := engine.Request{V: 1, ID: 6, Method: "export", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q,"format":"json","view":{}}`, scanID))}
	em := request(t, e, exp)
	var er struct {
		Format  string `json:"format"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(mustResult(t, em), &er); err != nil {
		t.Fatal(err)
	}
	view, err := engine.BuildViewReport(report, engine.ViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	skipped := report.Counts.Disabled - view.Counts.Disabled
	if skipped < 0 {
		skipped = 0
	}
	want, err := enginerender.Render("json", view, enginerender.Options{Version: engine.EngineVersion, SkippedDisabled: skipped})
	if err != nil {
		t.Fatal(err)
	}
	if er.Content != want {
		t.Fatal("export json does not match engine renderer output")
	}

	md := engine.Request{V: 1, ID: 7, Method: "markdown", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{}}`, scanID))}
	mm := request(t, e, md)
	var mr struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(mustResult(t, mm), &mr); err != nil {
		t.Fatal(err)
	}
	wantMD, err := enginerender.Render("markdown", view, enginerender.Options{Version: engine.EngineVersion, SkippedDisabled: skipped})
	if err != nil {
		t.Fatal(err)
	}
	if mr.Content != wantMD {
		t.Fatal("markdown export does not match engine renderer output")
	}
}

func TestCancelDuringScan(t *testing.T) {
	e := engine.New(engine.Options{ScanFunc: func(ctx context.Context, path string, progress func(int, int)) (core.Report, error) {
		<-ctx.Done()
		return core.Report{}, ctx.Err()
	}})
	defer e.Close()

	params, _ := json.Marshal(map[string]string{"path": "/tmp/whatever"})
	ch := submitAsync(t, e, engine.Request{V: 1, ID: 10, Method: "scan", Params: params})
	// Give the job a moment to register, then cancel it.
	time.Sleep(50 * time.Millisecond)
	cancelParams, _ := json.Marshal(map[string]uint64{"id": 10})
	cm := request(t, e, engine.Request{V: 1, ID: 11, Method: "cancel", Params: cancelParams})
	var cres struct {
		Cancelled bool `json:"cancelled"`
	}
	if err := json.Unmarshal(mustResult(t, cm), &cres); err != nil {
		t.Fatal(err)
	}
	if !cres.Cancelled {
		t.Fatal("cancel reported not cancelled")
	}
	m := waitResponse(t, ch, 10)
	mustError(t, m, engine.CodeScanCancelled)
}

func TestViewDuringBlockedScan(t *testing.T) {
	root := fixtureInstance(t)
	release := make(chan struct{})
	e := engine.New(engine.Options{ScanFunc: func(ctx context.Context, path string, progress func(int, int)) (core.Report, error) {
		select {
		case <-release:
			return core.Scan(path, core.Options{IncludeNested: true})
		case <-ctx.Done():
			return core.Report{}, ctx.Err()
		}
	}})
	defer e.Close()

	params, _ := json.Marshal(map[string]string{"path": root})
	ch := submitAsync(t, e, engine.Request{V: 1, ID: 20, Method: "scan", Params: params})

	// The engine stays responsive while the scan is blocked.
	hm := request(t, e, engine.Request{V: 1, ID: 21, Method: "hello"})
	mustResult(t, hm)
	mustError(t, request(t, e, engine.Request{V: 1, ID: 22, Method: "view", Params: json.RawMessage(`{"scanId":"scan-1","view":{}}`)}), engine.CodeUnknownScan)

	close(release)
	m := waitResponse(t, ch, 20)
	var res struct {
		ScanID string `json:"scanId"`
	}
	if err := json.Unmarshal(mustResult(t, m), &res); err != nil {
		t.Fatal(err)
	}
	vm := request(t, e, engine.Request{V: 1, ID: 23, Method: "view", Params: json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{}}`, res.ScanID))})
	var vr struct {
		State engine.State `json:"state"`
	}
	if err := json.Unmarshal(mustResult(t, vm), &vr); err != nil {
		t.Fatal(err)
	}
	if vr.State.Total == 0 {
		t.Fatal("expected mods after scan completed")
	}
}

func TestGroupedView(t *testing.T) {
	root := fixtureInstance(t)
	e := engine.New(engine.Options{})
	defer e.Close()
	scanID := scanInstance(t, e, root)

	params := json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{"groupBy":"loader","showDisabled":true}}`, scanID))
	m := request(t, e, engine.Request{V: 1, ID: 30, Method: "view", Params: params})
	var vr struct {
		State engine.State `json:"state"`
	}
	if err := json.Unmarshal(mustResult(t, m), &vr); err != nil {
		t.Fatal(err)
	}
	if len(vr.State.Groups) == 0 {
		t.Fatal("expected groups")
	}
	seen := map[string]bool{}
	for _, g := range vr.State.Groups {
		seen[g.Key] = true
		if g.Count <= 0 || g.Title == "" {
			t.Fatalf("bad group %+v", g)
		}
	}
	if !seen["fabric"] || !seen["quilt"] {
		t.Fatalf("missing loader groups: %v", vr.State.Groups)
	}
}

func TestLargeFixture(t *testing.T) {
	root := newInstance(t)
	var b strings.Builder
	b.WriteString("modLoader = \"javafml\"\n")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&b, "[[mods]]\nmodId = \"bulk-%03d\"\nversion = \"1.0.%d\"\ndisplayName = \"Bulk %03d\"\n", i, i, i)
	}
	addJar(t, root, "bulk.jar", map[string]string{"META-INF/mods.toml": b.String()})

	e := engine.New(engine.Options{})
	defer e.Close()
	scanID := scanInstance(t, e, root)
	params := json.RawMessage(fmt.Sprintf(`{"scanId":%q,"view":{"sort":"name"}}`, scanID))
	m := request(t, e, engine.Request{V: 1, ID: 40, Method: "view", Params: params})
	var vr struct {
		State engine.State `json:"state"`
	}
	if err := json.Unmarshal(mustResult(t, m), &vr); err != nil {
		t.Fatal(err)
	}
	if vr.State.Total != 300 || len(vr.State.Rows) != 300 {
		t.Fatalf("large fixture: total %d rows %d", vr.State.Total, len(vr.State.Rows))
	}
	ids := map[string]bool{}
	for _, row := range vr.State.Rows {
		if row.RowID == "" || ids[row.RowID] {
			t.Fatalf("duplicate/empty row id: %q", row.RowID)
		}
		ids[row.RowID] = true
	}
}

func TestProtocolGoldens(t *testing.T) {
	entries, err := os.ReadDir("../testdata/protocol")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		data, err := os.ReadFile(filepath.Join("../testdata/protocol", name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "malformed.request.json" {
			var req engine.Request
			if err := json.Unmarshal(data, &req); err == nil {
				t.Fatal("malformed golden unexpectedly parsed")
			}
			continue
		}
		switch {
		case strings.HasSuffix(name, ".request.json"):
			var req engine.Request
			if err := json.Unmarshal(data, &req); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			wantV := engine.ProtocolVersion
			if name == "unsupported_version.request.json" {
				wantV = engine.ProtocolVersion + 1
			}
			if req.V != wantV {
				t.Fatalf("%s: version %d, want %d", name, req.V, wantV)
			}
			seen[req.Method] = true
		case strings.HasSuffix(name, ".response.json"), strings.HasSuffix(name, ".event.json"):
			var msg engine.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if msg.V != engine.ProtocolVersion {
				t.Fatalf("%s: version %d", name, msg.V)
			}
		}
	}
	for _, method := range engine.Methods {
		if !seen[method] {
			t.Fatalf("no golden request for method %q", method)
		}
	}
}

func TestPathErrors(t *testing.T) {
	e := engine.New(engine.Options{})
	defer e.Close()
	missing := filepath.Join(t.TempDir(), "nope")
	params, _ := json.Marshal(map[string]string{"path": missing})
	m := request(t, e, engine.Request{V: 1, ID: 50, Method: "inspectPath", Params: params})
	mustError(t, m, engine.CodePathNotFound)

	emptyDir := t.TempDir()
	params, _ = json.Marshal(map[string]string{"path": emptyDir})
	m = request(t, e, engine.Request{V: 1, ID: 51, Method: "inspectPath", Params: params})
	mustError(t, m, engine.CodeNotAnInstance)
}
