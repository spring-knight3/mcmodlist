# PHASES.md

Ordered build plan for **mcmodlist**. Work one phase at a time.

> **STOP RULE:** After each phase, run the checks in `AGENTS.md` section 8,
> tick the boxes below, output the phase report (`AGENTS.md` section 9), and
> **STOP**. Do not begin the next phase until the user explicitly says to
> continue (for example "Continue to Phase 2").

Status key: `[ ]` not done, `[x]` done.

| Phase | Name | Status |
|-------|------|--------|
| 1 | Core: instance detection and jar scanning | [x] |
| 2 | CLI and output formats | [x] |
| 3 | Filtering, sorting, and friendly behaviour | [x] |
| 4 | Diagnostics, comparison, config, completions | [x] |
| 5 | GUI (Wails; superseded by 5R) | [x] |
| 5R | GUI pivot: Qt 6 frontend + Go engine | [ ] |
| 6 | Packaging: Arch Linux PKGBUILD | [ ] |

---

## Phase 1: Core (instance detection and jar scanning)

**Goal:** a tested `core` library that, given a directory, returns structured
data about the instance and every mod in it. No CLI, no formatting.

### Tasks
- [x] Initialise the Go module, directory layout from `AGENTS.md`, `README.md`,
      `DECISIONS.md`, and CI-ready `go test ./... -race`.
- [x] `model`: `Instance`, `Mod`, `Counts`, `Issue`, `Warning` structs.
- [x] `instance`: detect a valid instance (`mods/` or `.minecraft/mods/`
      and/or launcher metadata). Return typed errors for invalid paths and for
      "this looks like a mods folder; did you mean the parent?".
- [x] Launcher parsers: Prism/MultiMC (`instance.cfg`, `mmc-pack.json`),
      CurseForge (`minecraftinstance.json`), Modrinth App (`profile.json`),
      ATLauncher (`instance.json`), GDLauncher (`config.json`). Fallback to the
      directory name and jar-inferred loader.
- [x] `jar`: streaming reader that opens only needed entries.
- [x] Loader metadata parsers: Fabric (`fabric.mod.json`), Quilt
      (`quilt.mod.json`), Forge (`META-INF/mods.toml`, legacy `mcmod.info`),
      NeoForge (`META-INF/neoforge.mods.toml`).
- [x] Normalise authors (string, array, object forms) to a list.
- [x] Multiple mods per jar; multi-loader jars counted once.
- [x] Resolve version placeholders from `META-INF/MANIFEST.MF`, else `unknown`.
- [x] Ignore Jar-in-Jar by default (flag the option, expose it in the API);
      detect `*.jar.disabled` and mark them disabled.
- [x] Corrupt or metadata-less jars become `unrecognised` entries plus a
      warning; the scan never aborts.
- [x] Compatibility-layer detection (Sinytra Connector on Forge/NeoForge) and
      loader classification with correct counts (mods, not jars).
- [x] Bounded concurrent scanning with deterministic result ordering.
- [x] Programmatic fixture-jar builders and tests for every item above.

### Acceptance criteria
- A single call such as `core.Scan(path, opts)` returns the instance info, the
  mod list, counts per loader/layer, and warnings.
- All required fixtures from `AGENTS.md` section 6 that apply to this phase are
  covered by passing tests.
- `core` contains no printing, flag parsing, or display formatting.
- Re-running the scan on the same input yields byte-identical ordering.
- The instance on disk is never modified (a test asserts unchanged mtimes and
  contents).

### STOP
Output the phase report and **wait for "Continue to Phase 2"**.

---

## Phase 2: CLI and output formats

**Goal:** a working `mcmodlist` binary with the essential flags and all output
formats.

### Tasks
- [x] `cmd/mcmodlist` + `cli` wiring; positional `[instance]` and
      `-i/--instance`; default `.`.
- [x] Flags: `-f/--format`, `-o/--output`, `--force`, `--no-summary`,
      `--summary-only`, `--no-header`, `--color auto|always|never`
      (`--no-color` alias, honour `NO_COLOR`), `--verbose`, `-q/--quiet`,
      `-h/--help`, `-V/--version`.
- [x] Renderers: `pretty` (aligned, terminal-width aware), `markdown` (table),
      `csv` (RFC 4180, quoted), `json` (schema below), `html` (single
      self-contained file), `text` (plain).
- [x] Always show the instance name, Minecraft version, loader, loader-count
      line (`Fabric/Connector(n) Quilt(n) Forge(n) NeoForge(n)`, only non-zero
      loaders), and total.
- [x] Disabled, nested, and unrecognised entries visibly tagged.
- [x] Footer note for skipped disabled mods and unrecognised jars.
- [x] Error handling: results to stdout, errors/warnings to stderr; exit codes
      `0`, `1`, `2`.
- [x] `--help` with 4-5 copy-pasteable examples.
- [x] Golden-file tests for every format; CLI tests for flags and exit codes.

### JSON schema (frozen at the end of this phase)
```json
{
  "tool": {"name": "mcmodlist", "version": ""},
  "instance": {"name": "", "path": "", "launcher": "", "minecraft": "",
               "loader": "", "loaderVersion": ""},
  "counts": {"fabric": 0, "quilt": 0, "forge": 0, "neoforge": 0,
             "connector": 0, "total": 0, "disabled": 0, "unrecognised": 0},
  "mods": [{"name": "", "id": "", "version": "", "authors": [], "jar": "",
            "loader": "", "side": "", "status": "enabled|disabled|nested"}],
  "issues": [{"type": "", "mod": "", "message": ""}],
  "warnings": []
}
```

### Acceptance criteria
- `mcmodlist <fixture-instance>` prints correct output in all six formats.
- An invalid path exits `1` with a helpful message; a bad flag exits `2`.
- Output is identical across repeated runs and has no ANSI codes when piped.
- Pointing at a `mods/` folder suggests the parent instance directory.
- Golden tests pass on all formats.

### STOP
Output the phase report and **wait for "Continue to Phase 3"**.

---

## Phase 3: Filtering, sorting, and friendly behaviour

**Goal:** make the tool pleasant for both non-technical users and scripters.

### Tasks
- [x] Input flags: `--launcher`, `--list-instances` (search known launcher
      directories per OS), `--include-disabled`, `--include-nested`,
      `--recursive`, `--jobs`.
- [x] Output flags: `--columns`, `--sort` (prefix `-` reverses), `--group-by
      loader|side`, `--wide`, `--truncate <n>`, `--copy` (clipboard on
      Windows, macOS, Linux; clear message if unavailable).
- [x] Filter flags: `--search`, `--loader`, `--side`, `--author`,
      `--unrecognised`.
- [x] **Interactive picker** when no instance is given in a non-instance
      directory on a TTY; never prompts when piped.
- [x] Typo suggestions for unknown flags and formats ("did you mean ...").
- [x] Progress bar on stderr for large packs, TTY only.
- [x] UTF-8 output with a safe fallback.
- [x] Tests for each flag, the picker (with simulated input), and typo help.

### Acceptance criteria
- Every flag above works, is documented in `--help`, and has a test.
- `--format markdown --copy` places a valid markdown table on the clipboard
  (tested through an injectable clipboard interface).
- Piped output is clean: no prompts, no progress, no colour.
- Counts in the loader line reflect filters only when documented, and the
  behaviour is covered by a test and noted in `README.md`.

### STOP
Output the phase report and **wait for "Continue to Phase 4"**.

---

## Phase 4: Diagnostics, comparison, config, completions

**Goal:** the power-user features.

### Tasks
- [x] `--check`: detect duplicate mod ids across different jars, loader
      mismatches, missing declared dependencies, wrong Minecraft version.
      Populate `issues`; exit `3` when any are found.
- [x] `--diff <other-instance>`: mods added, removed, and version-changed, in
      all formats.
- [x] Optional config file `~/.config/mcmodlist/config.toml` (platform-correct
      path) for default format, columns, and colour. Flags override it.
- [x] `mcmodlist completion <bash|zsh|fish|powershell>`.
- [x] `README.md` finalised: install, usage, every flag, examples, JSON schema,
      exit codes, and the "paste into a support post" recipe.
- [x] Tests for `--check` (one fixture per issue type), `--diff`, config
      precedence, and completion script generation.

### Acceptance criteria
- Each `--check` issue type is triggered by a dedicated fixture and nothing
  else; a clean instance exits `0`.
- `--diff` is symmetric in tests (swap the arguments, added/removed swap).
- Config values are overridden by flags; a malformed config yields a clear
  warning, not a crash.
- The JSON schema from Phase 2 is unchanged except for additive fields
  (flagged in `DECISIONS.md`).

### STOP
Output the phase report and **wait for "Continue to Phase 5"**.

---

## Phase 5: GUI

> **Superseded by Phase 5R:** the Wails toolkit decision is replaced by a
> native C++/Qt 6 GUI talking to a Go engine over JSON-Lines. This section is
> kept as history; the Wails code remains until Phase 5R R6 removes it.

**Goal:** an optional graphical front end that reuses `core` and duplicates no
logic.

### Tasks
- [x] Ask the user to confirm the toolkit (Go options: Wails or Fyne) before
      writing code, and record it in `DECISIONS.md`.
- [x] Separate `gui/` package depending on `core` only; `--gui` in the CLI
      launches it if installed, otherwise prints an install hint.
- [x] Start screen: instances auto-listed from installed launchers, plus
      drag-and-drop and "Browse...".
- [x] Main view: sortable, filterable table; live search; loader filter
      chips; header bar with instance name and the loader-count line.
- [x] Details pane: description, authors, license, links, jar path, warnings.
- [x] Toggles: show disabled, show nested, group by loader.
- [x] Buttons: Copy as Markdown, Export (all formats), Check for problems,
      Compare with another instance.
- [x] Background scanning with a progress indicator; the UI never freezes.
- [x] Dark/light mode follows the system. Read-only, like the CLI.
- [x] Tests for view-model logic; manual test checklist in `README.md`.

### Acceptance criteria
- GUI output for a fixture instance matches the CLI's counts and mod list
  exactly (shared `core`, verified by a test).
- Scanning a 300+ mod fixture does not block the UI thread.
- No logic from `cli` is copied into `gui`.
- Building the CLI alone does not require the GUI toolkit or its dependencies.

### STOP
Output the phase report and **wait for "Continue to Phase 6"**.

---

## Phase 5R: GUI pivot — Qt 6 frontend + Go engine

**Goal:** replace the Wails GUI with a native C++/Qt 6 Widgets application.
Go remains the engine; the Qt app is a dumb view over a versioned JSON-Lines
protocol on the engine's stdin/stdout. Phase 6 packaging then covers both
executables shipped together.

### Hard rules
- `core` and the CLI do not change behaviour; the CLI still builds with no GUI
  dependency.
- `engine/` imports `core` only. `qtgui` knows only the protocol, never Go
  packages. Shared view-model logic moves from `gui/` into `engine/`; until R6,
  `gui/` may import `engine/`, but `engine/` never imports GUI code.
- The Qt app contains no parsing, filtering, sorting, grouping, counting,
  loader ordering, or export formatting. Enforce with the greps in the pivot
  brief (`QSortFilterProxyModel`/`std::sort` over rows in `qtgui/src/model`,
  and the no-hardcoded-color grep).
- No hardcoded colors in `qtgui/`; use `QPalette` roles plus one
  `PaletteUtil` blending helper, repaint custom widgets on palette changes,
  and never use colour as the only carrier of meaning.
- Never block the GUI thread: no synchronous engine calls or `waitFor*` on the
  GUI thread; parse oversized engine lines in a worker and return by signal.
- Preserve existing Go/Wails behaviour: columns, labels, defaults, toggles,
  empty states, Check semantics, Compare semantics, and export formats. Record
  assumptions in `DECISIONS.md`.
- Qt 6 only, C++20, CMake, Qt Test; Qt 6.5 minimum. Widgets and Test are
  enough; add Concurrent if needed. No other third-party libraries.

### R0: Decision and environment
- [x] Update `DECISIONS.md`: record the Qt pivot, the reason, and the rejected
      options (miqt-style bindings, rewriting `core`, localhost HTTP, Wails).
- [x] Update `PHASES.md`: mark Phase 5 toolkit superseded by 5R without
      deleting history; add this section.
- [x] Check Qt 6 development files and CMake availability; report honestly.

**Exit:** decisions and phase plan updated; Qt availability reported.

### R1: Engine and protocol (Go, no Qt needed)
- [x] Write `docs/PROTOCOL.md` first: transport, envelope, methods, events,
      error codes, scan store, cancellation, and client rules.
- [x] Implement `engine/` (protocol types, handlers, scan store) and
      `cmd/mcmodlist-engine/` (stdio loop). Move shared view-model logic from
      `gui/` into `engine/` on top of `core`.
- [x] Add golden JSON under `testdata/protocol/` for every method, errors, and
      progress events; add `scripts/engine-smoke.sh`.
- [x] Tests with `-race`: every method against fixtures, 300-mod fixture,
      cancel during scan, overlapping `view` during `scan`, malformed JSON,
      unknown method, protocol mismatch, oversized line, closed stdin, stable
      row ids, parity with `core.FilterMods`/`CountMods`/`OrderedLoaderLabels`,
      and export parity with the CLI.

**Exit:** Go checks clean; `engine/` imports `core` only; goldens cover every
method; CLI dependency check stays empty.

### R2: Qt skeleton and engine client
- [x] Create `qtgui/` with CMake, branding constants, `EngineLocator`,
      `EngineClient` (QProcess JSON-Lines, ids, signals, timeouts, restart),
      `Protocol` parsing, `PaletteUtil`, empty `MainWindow`, and `StartScreen`
      (instance list, Browse, manual path, drag-and-drop, `inspectPath`).
- [x] Qt Test offscreen: client against real engine and stub engine, golden
      parsing, split/partial lines, crash mid-request, version mismatch,
      heartbeat during slow engine call.

**Exit:** window opens offscreen and lists instances from the real engine;
golden tests pass; no GUI-thread blocking; color grep empty.

### R3: Main view
- [x] `ModTableModel` shows engine rows exactly as returned, including group
      header rows, non-selectable rows, spanning, and palette-drawn styling;
      header clicks send `view`; the model never reorders.
- [x] Search debounce (~120 ms), side filter, loader chips ordered by the
      engine and stable while filtered, disabled/nested/unrecognised toggles,
      loader/side grouping, header instance/count line, progress bar with
      Cancel.
- [x] Tests: exact rows/order/groups/counts vs engine JSON on the 300-mod
      fixture, latest-wins under rapid changes, palette swap recolours custom
      elements without restart.

**Exit:** Phase 5 main-view criteria re-verified on Qt; model grep clean;
300-mod state builds without freezing.

### R4: Details, Check, Compare, Copy, Export
- [x] `DetailsPane` for description/authors/license/links/jar path/warnings;
      links via `QDesktopServices`; jar path copy.
- [x] `Check` and `Compare` via `ReportDialog`; Compare keeps the original scan
      as base; Copy as Markdown to clipboard; Export with native save dialog and
      `QSaveFile`.
- [x] Tests: each action on fixtures; exported files byte-identical to CLI
      output for the same format/filters; readable engine errors.

**Exit:** Phase 5 details/toggle/button criteria re-verified on Qt.

### R5: Theming, accessibility, polish
- [x] Offscreen palette/colour-scheme swap test for every custom-painted
      element; run the color grep.
- [x] Keyboard navigation, accessible names/descriptions, HiDPI-correct
      drawing, `tr()` on user-visible strings, `QSettings` persistence
      (geometry, splitters, last path, engine override), clear error UX for
      engine missing/crash/protocol mismatch/invalid path/scan cancelled.

**Exit:** palette-swap test passes; color grep empty; controls keyboard
reachable.

### R6: Remove Wails and finish
- [x] Delete `gui/`, old `cmd/mcmodlist-gui`, and frontend; `go mod tidy`; no
      `wails` in `go.mod`/`go.sum`. Keep `mcmodlist --gui` launching the Qt
      `mcmodlist-gui` and keep the not-installed hint test.
- [x] Update `README.md` (build via `scripts/build-all.sh`, run,
      `MCMODLIST_ENGINE`, theming, Qt manual checklist) and `DECISIONS.md`.
- [x] Add CI steps: Go tests, engine build, Qt configure/build, offscreen
      `ctest`, color grep, model grep.
- [ ] Continue to original Phase 6 packaging for the two executables.

**Exit:** no Wails code/dependency remains; Go checks clean; CLI builds alone;
Qt tests pass offscreen.

### STOP
After each R sub-phase, run the checks from the pivot brief, tick completed
items, output a short phase report, and **stop** until told to continue.

---

## Phase 6: Packaging (Arch Linux PKGBUILD)

**Goal:** installable Arch Linux packages for the CLI (and the optional GUI),
following the Arch Go package guidelines, ready to submit to the AUR.

### Tasks
- [x] Ask the user to confirm before writing files: the **license** (do not
      guess), the real **repository URL**, the **maintainer** name/email for
      the PKGBUILD header, and whether to also provide a `-git` package.
      Record the answers in `DECISIONS.md`.
- [x] Make the version injectable at build time: `-V/--version` reads a
      `main.version` variable set with `-ldflags "-X main.version=$pkgver"`
      (fall back to `dev` when unset). Add a test.
- [x] `packaging/arch/PKGBUILD` for `mcmodlist` (release tarball from a tag):
  - [x] Standard header: `pkgname`, `pkgver`, `pkgrel=1`, `pkgdesc`,
        `arch=('x86_64' 'aarch64')`, `url`, `license`, `depends=('glibc')`
        (or none if built with `CGO_ENABLED=0`; record which in
        `DECISIONS.md`), `makedepends=('go')`, `source`, `sha256sums`.
  - [x] `prepare()`: `go mod download -x` with `GOFLAGS=-modcacherw`.
  - [x] `build()`: set `CGO_ENABLED`, `CGO_CPPFLAGS`, `CGO_CFLAGS`,
        `CGO_CXXFLAGS`, `CGO_LDFLAGS`, and
        `GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"`; build
        `./cmd/mcmodlist` with
        `-ldflags "-linkmode external -extldflags \"${LDFLAGS}\" -X main.version=${pkgver}"`.
  - [x] `check()`: `go test ./...` for `core` and `cli` (not the GUI).
  - [x] `package()`: install the binary to `/usr/bin/mcmodlist` (mode 755),
        the license to `/usr/share/licenses/$pkgname/`, `README.md` to
        `/usr/share/doc/$pkgname/`, and generated shell completions to
        `/usr/share/bash-completion/completions/mcmodlist`,
        `/usr/share/zsh/site-functions/_mcmodlist`, and
        `/usr/share/fish/vendor_completions.d/mcmodlist.fish` (using the
        Phase 4 `completion` subcommand).
- [x] GUI as a **split package** (`pkgname=('mcmodlist' 'mcmodlist-gui')`) or a
      separate PKGBUILD, whichever suits the toolkit chosen in Phase 5;
      `mcmodlist` must not depend on GUI libraries, and may list
      `mcmodlist-gui` in `optdepends`. The `--gui` install hint in the CLI
      names this package. Record the choice in `DECISIONS.md`.
- [x] If requested: `packaging/arch/PKGBUILD-git` for `mcmodlist-git` with
      `provides`/`conflicts`, `pkgver()` derived from `git describe`, and
      `source=("git+$url.git")` with `sha256sums=('SKIP')`.
- [x] Generate and commit `.SRCINFO` (`makepkg --printsrcinfo > .SRCINFO`) for
      each PKGBUILD.
- [x] `packaging/arch/README.md`: how to build locally (`makepkg -si`), how to
      test in a clean chroot (`extra-x86_64-build` / `makechrootpkg`), how to
      update `pkgver` and checksums for a release (`updpkgsums`), and how to
      push to the AUR.
- [ ] Lint with `namcap` on both the PKGBUILD and the built package; fix or
      justify every warning in `DECISIONS.md`.

### Acceptance criteria
- `makepkg` builds the package from a clean checkout with no network access
  beyond fetching the source and Go modules, and the resulting package
  installs and runs `mcmodlist --version` showing the packaged `pkgver`.
- The binary is a PIE, built with `-trimpath`, and the package contains the
  license, docs, and all three completion files.
- `namcap` reports no errors on the PKGBUILD or the package; any warnings are
  explained.
- `mcmodlist` installs and works without the GUI package or GUI libraries
  present.
- `.SRCINFO` matches the PKGBUILD.
- **Verification honesty:** if `makepkg`, `namcap`, or a clean chroot is not
  available in the agent's environment, the agent must say exactly which
  checks it could not run, provide the commands for the user to run, and must
  not report those criteria as passed.

### STOP
Output the final phase report, then **wait for further instructions**. The
project is not considered released until the user says so.
