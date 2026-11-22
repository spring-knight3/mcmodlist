package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"mcmodlist/core"
	"mcmodlist/core/instance"
	"mcmodlist/core/model"
	"mcmodlist/engine/render"
)

// ScanFunc scans one instance path and reports progress.
type ScanFunc func(ctx context.Context, path string, progress func(done, total int)) (core.Report, error)

// Options configures an Engine.
type Options struct {
	MaxScans int
	ScanFunc ScanFunc
}

type scanEntry struct {
	base    core.Report
	current core.Report
}

// Engine serves protocol requests.
type Engine struct {
	opts   Options
	mu     sync.Mutex
	scans  map[string]*scanEntry
	order  []string
	nextID uint64
	jobs   map[uint64]context.CancelFunc
	emitMu sync.Mutex
	wg     sync.WaitGroup
}

// New returns an Engine.
func New(opts Options) *Engine {
	if opts.MaxScans <= 0 {
		opts.MaxScans = 4
	}
	if opts.ScanFunc == nil {
		opts.ScanFunc = defaultScan
	}
	return &Engine{opts: opts, scans: make(map[string]*scanEntry), jobs: make(map[uint64]context.CancelFunc)}
}

// Submit handles one request. Responses and events are emitted through emit,
// which Engine serializes.
func (e *Engine) Submit(req Request, emit func(Message)) {
	if req.V != ProtocolVersion {
		e.emit(emit, errorMessage(req.ID, CodeUnsupportedVersion, fmt.Sprintf("unsupported protocol version %d", req.V), nil))
		return
	}
	switch req.Method {
	case "hello":
		e.emit(emit, response(req.ID, map[string]any{"protocol": ProtocolVersion, "engineVersion": EngineVersion, "methods": Methods}))
	case "listInstances":
		e.handleListInstances(req, emit)
	case "inspectPath":
		e.handleInspectPath(req, emit)
	case "scan":
		e.handleScan(req, emit)
	case "cancel":
		e.handleCancel(req, emit)
	case "view":
		e.handleView(req, emit)
	case "details":
		e.handleDetails(req, emit)
	case "check":
		e.handleCheck(req, emit)
	case "compare":
		e.handleCompare(req, emit)
	case "export":
		e.handleExport(req, emit)
	case "markdown":
		e.handleMarkdown(req, emit)
	case "shutdown":
		e.emit(emit, response(req.ID, map[string]any{"bye": true}))
	default:
		e.emit(emit, errorMessage(req.ID, CodeInvalidRequest, fmt.Sprintf("unknown method %q", req.Method), nil))
	}
}

// Close cancels active jobs and waits for them.
func (e *Engine) Close() {
	e.mu.Lock()
	for _, cancel := range e.jobs {
		cancel()
	}
	e.mu.Unlock()
	e.wg.Wait()
}

func (e *Engine) emit(emit func(Message), msg Message) {
	e.emitMu.Lock()
	defer e.emitMu.Unlock()
	emit(msg)
}

func (e *Engine) handleListInstances(req Request, emit func(Message)) {
	var p struct{ Launcher string }
	if err := decodeParams(req, &p); err != nil {
		e.emit(emit, invalid(req))
		return
	}
	instances, err := ListInstances()
	if err != nil {
		e.emit(emit, errorMessage(req.ID, CodeInternal, err.Error(), nil))
		return
	}
	out := make([]InstanceOption, 0, len(instances))
	for _, inst := range instances {
		if p.Launcher != "" && !strings.EqualFold(inst.Launcher, p.Launcher) {
			continue
		}
		out = append(out, inst)
	}
	e.emit(emit, response(req.ID, map[string]any{"instances": out}))
}

func (e *Engine) handleInspectPath(req Request, emit func(Message)) {
	var p struct{ Path string }
	if err := decodeParams(req, &p); err != nil {
		e.emit(emit, invalid(req))
		return
	}
	det, err := instance.Detect(p.Path)
	if err != nil {
		code := CodeNotAnInstance
		detail := map[string]any{"kind": "not_instance"}
		var ierr *instance.Error
		if errors.As(err, &ierr) {
			switch ierr.Kind {
			case instance.KindNotDirectory:
				code = CodePathNotFound
			case instance.KindUnreadable:
				code = CodeUnreadable
			case instance.KindModsFolder:
				detail = map[string]any{"kind": "mods_folder", "suggestion": ierr.Suggestion}
			}
		}
		e.emit(emit, errorMessage(req.ID, code, err.Error(), detail))
		return
	}
	e.emit(emit, response(req.ID, map[string]any{"valid": true, "path": det.Info.Path, "instance": instanceOption(det.Info)}))
}

func (e *Engine) handleScan(req Request, emit func(Message)) {
	var p struct{ Path string }
	if err := decodeParams(req, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		e.emit(emit, invalid(req))
		return
	}
	e.startJob(req, emit, p.Path, func(ctx context.Context, progress func(int, int)) (core.Report, error) {
		return e.opts.ScanFunc(ctx, p.Path, progress)
	}, func(report core.Report) any {
		id := e.store(report)
		return map[string]any{"scanId": id, "summary": summaryFor(report)}
	})
}

func (e *Engine) handleCompare(req Request, emit func(Message)) {
	var p struct {
		ScanID string `json:"scanId"`
		Path   string `json:"path"`
	}
	if err := decodeParams(req, &p); err != nil || p.ScanID == "" || strings.TrimSpace(p.Path) == "" {
		e.emit(emit, invalid(req))
		return
	}
	e.mu.Lock()
	entry, ok := e.scans[p.ScanID]
	var base core.Report
	if ok {
		base = entry.base
	}
	e.mu.Unlock()
	if !ok {
		e.emit(emit, errorMessage(req.ID, CodeUnknownScan, "unknown scanId", nil))
		return
	}
	e.startJob(req, emit, p.Path, func(ctx context.Context, progress func(int, int)) (core.Report, error) {
		return e.opts.ScanFunc(ctx, p.Path, progress)
	}, func(other core.Report) any {
		diff := DiffReports(base, other)
		e.mu.Lock()
		if ent := e.scans[p.ScanID]; ent != nil {
			ent.current = diff
		}
		e.mu.Unlock()
		return map[string]any{"scanId": p.ScanID, "summary": summaryFor(diff)}
	})
}

func (e *Engine) handleCancel(req Request, emit func(Message)) {
	var p struct{ ID uint64 }
	if err := decodeParams(req, &p); err != nil {
		e.emit(emit, invalid(req))
		return
	}
	e.mu.Lock()
	cancel := e.jobs[p.ID]
	e.mu.Unlock()
	cancelled := false
	if cancel != nil {
		cancel()
		cancelled = true
	}
	e.emit(emit, response(req.ID, map[string]any{"cancelled": cancelled}))
}

func (e *Engine) handleView(req Request, emit func(Message)) {
	var p struct {
		ScanID string      `json:"scanId"`
		View   ViewOptions `json:"view"`
	}
	if err := decodeParams(req, &p); err != nil || p.ScanID == "" {
		e.emit(emit, invalid(req))
		return
	}
	entry, ok := e.getScan(p.ScanID)
	if !ok {
		e.emit(emit, errorMessage(req.ID, CodeUnknownScan, "unknown scanId", nil))
		return
	}
	view, err := BuildViewReport(entry.current, p.View)
	if err != nil {
		e.emit(emit, errorMessage(req.ID, CodeInvalidRequest, err.Error(), nil))
		return
	}
	state := BuildState(entry.current, p.View)
	state = groupState(state, view.Mods, p.View.GroupBy)
	e.emit(emit, response(req.ID, map[string]any{"state": state}))
}

func (e *Engine) handleDetails(req Request, emit func(Message)) {
	var p struct{ ScanID, RowID string }
	if err := decodeParams(req, &p); err != nil || p.ScanID == "" || p.RowID == "" {
		e.emit(emit, invalid(req))
		return
	}
	entry, ok := e.getScan(p.ScanID)
	if !ok {
		e.emit(emit, errorMessage(req.ID, CodeUnknownScan, "unknown scanId", nil))
		return
	}
	for _, mod := range entry.current.Mods {
		if rowID(mod) == p.RowID {
			row := rowFromMod(mod)
			e.emit(emit, response(req.ID, Details{Mod: row, Warnings: warningsForJar(entry.current.Warnings, row.Jar)}))
			return
		}
	}
	e.emit(emit, errorMessage(req.ID, CodeInvalidRequest, "unknown rowId", nil))
}

func (e *Engine) handleCheck(req Request, emit func(Message)) {
	var p struct {
		ScanID string `json:"scanId"`
	}
	if err := decodeParams(req, &p); err != nil || p.ScanID == "" {
		e.emit(emit, invalid(req))
		return
	}
	entry, ok := e.getScan(p.ScanID)
	if !ok {
		e.emit(emit, errorMessage(req.ID, CodeUnknownScan, "unknown scanId", nil))
		return
	}
	e.emit(emit, response(req.ID, map[string]any{"issues": CheckReport(entry.base)}))
}

func (e *Engine) handleExport(req Request, emit func(Message)) {
	e.handleExportFormat(req, emit, false)
}
func (e *Engine) handleMarkdown(req Request, emit func(Message)) {
	e.handleExportFormat(req, emit, true)
}

func (e *Engine) handleExportFormat(req Request, emit func(Message), markdownOnly bool) {
	var p struct {
		ScanID string      `json:"scanId"`
		Format string      `json:"format"`
		View   ViewOptions `json:"view"`
	}
	if err := decodeParams(req, &p); err != nil || p.ScanID == "" {
		e.emit(emit, invalid(req))
		return
	}
	format := p.Format
	if markdownOnly {
		format = "markdown"
	}
	entry, ok := e.getScan(p.ScanID)
	if !ok {
		e.emit(emit, errorMessage(req.ID, CodeUnknownScan, "unknown scanId", nil))
		return
	}
	view, err := BuildViewReport(entry.current, p.View)
	if err != nil {
		e.emit(emit, errorMessage(req.ID, CodeInvalidRequest, err.Error(), nil))
		return
	}
	skipped := entry.current.Counts.Disabled - view.Counts.Disabled
	if skipped < 0 {
		skipped = 0
	}
	out, err := render.Render(format, view, render.Options{Version: EngineVersion, GroupBy: p.View.GroupBy, SkippedDisabled: skipped})
	if err != nil {
		e.emit(emit, errorMessage(req.ID, CodeInvalidRequest, err.Error(), nil))
		return
	}
	if markdownOnly {
		e.emit(emit, response(req.ID, map[string]any{"content": out}))
		return
	}
	e.emit(emit, response(req.ID, map[string]any{"format": format, "content": out}))
}

func (e *Engine) startJob(req Request, emit func(Message), path string, scan func(context.Context, func(int, int)) (core.Report, error), finish func(core.Report) any) {
	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.jobs[req.ID] = cancel
	e.mu.Unlock()
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() { e.mu.Lock(); delete(e.jobs, req.ID); e.mu.Unlock() }()
		progress := func(done, total int) {
			e.emit(emit, event(req.ID, "scan-progress", map[string]any{"done": done, "total": total}))
		}
		report, err := scan(ctx, progress)
		if err != nil {
			code := CodeInternal
			if errors.Is(err, context.Canceled) {
				code = CodeScanCancelled
			} else if body := scanError(err); body != nil {
				e.emit(emit, errorMessage(req.ID, body.Code, body.Message, nil))
				return
			}
			e.emit(emit, errorMessage(req.ID, code, err.Error(), nil))
			return
		}
		e.emit(emit, response(req.ID, finish(report)))
	}()
}

func defaultScan(ctx context.Context, path string, progress func(int, int)) (core.Report, error) {
	type result struct {
		report core.Report
		err    error
	}
	done := make(chan result, 1)
	go func() {
		rep, err := core.Scan(path, core.Options{IncludeNested: true, Progress: func(d, t int) {
			if ctx.Err() == nil {
				progress(d, t)
			}
		}})
		done <- result{rep, err}
	}()
	select {
	case r := <-done:
		return r.report, r.err
	case <-ctx.Done():
		return core.Report{}, ctx.Err()
	}
}

func scanError(err error) *ErrorBody {
	var ierr *instance.Error
	if errors.As(err, &ierr) {
		switch ierr.Kind {
		case instance.KindNotDirectory:
			return &ErrorBody{Code: CodePathNotFound, Message: err.Error()}
		case instance.KindUnreadable:
			return &ErrorBody{Code: CodeUnreadable, Message: err.Error()}
		default:
			return &ErrorBody{Code: CodeNotAnInstance, Message: err.Error()}
		}
	}
	return nil
}

func (e *Engine) store(report core.Report) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nextID++
	id := fmt.Sprintf("scan-%d", e.nextID)
	e.scans[id] = &scanEntry{base: report, current: report}
	e.order = append(e.order, id)
	for len(e.order) > e.opts.MaxScans {
		old := e.order[0]
		e.order = e.order[1:]
		delete(e.scans, old)
	}
	return id
}

func (e *Engine) getScan(id string) (scanEntry, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, ok := e.scans[id]
	if !ok {
		return scanEntry{}, false
	}
	return *entry, true
}

func summaryFor(report core.Report) map[string]any {
	state := BuildState(report, ViewOptions{})
	return map[string]any{"instance": state.Instance, "loaders": state.Loaders, "loaderOptions": state.LoaderOptions, "total": state.Total, "disabled": state.Disabled, "warnings": state.Warnings, "issues": state.Issues}
}

func groupState(state State, mods []model.Mod, groupBy string) State {
	groupBy = strings.ToLower(strings.TrimSpace(groupBy))
	if (groupBy != "loader" && groupBy != "side") || len(state.Rows) != len(mods) {
		return state
	}
	keyOf := func(m model.Mod) string {
		if groupBy == "loader" {
			return loaderLabel(m)
		}
		return m.Side
	}
	titleOf := func(key string) string {
		if groupBy == "loader" {
			return "Loader: " + render.DisplayLoaderLabel(key)
		}
		if key == "" {
			return "Side: -"
		}
		return "Side: " + key
	}
	byKey := map[string][]int{}
	var keys []string
	for i, m := range mods {
		k := keyOf(m)
		if _, ok := byKey[k]; !ok {
			keys = append(keys, k)
		}
		byKey[k] = append(byKey[k], i)
	}
	sort.Strings(keys)
	var order []int
	var groups []Group
	for _, k := range keys {
		idxs := byKey[k]
		start := len(order)
		order = append(order, idxs...)
		groups = append(groups, Group{Key: k, Title: titleOf(k), Start: start, Count: len(idxs)})
	}
	rows := make([]ModRow, 0, len(order))
	pos := 0
	for _, k := range keys {
		for range byKey[k] {
			row := state.Rows[order[pos]]
			row.Group = k
			row.GroupTitle = titleOf(k)
			rows = append(rows, row)
			pos++
		}
	}
	state.Rows = rows
	state.Groups = groups
	return state
}

func invalid(req Request) Message {
	return errorMessage(req.ID, CodeInvalidRequest, "invalid request", nil)
}
