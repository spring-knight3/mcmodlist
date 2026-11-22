# DECISIONS.md

Non-obvious decisions and dependency justifications.

## Dependencies

- `github.com/pelletier/go-toml/v2` v2.2.4 — required to parse Forge/NeoForge
  `META-INF/mods.toml` and `META-INF/neoforge.mods.toml`; approved by the user.
  Chosen over writing a partial TOML parser because TOML edge cases are easy to
  get wrong and the library is small and maintained.

## Phase 1 decisions

- `core.Scan(path, opts)` is the only public entry point. It returns structured
  data and never prints or reads flags.
- Launcher metadata is best-effort. A corrupt metadata file does not fail
  detection; the directory name and jar inspection remain as fallbacks.
- Prism and MultiMC share the MMC pack format. When `mmc-pack.json` is present
  the launcher is reported as `prism`; `instance.cfg` alone reports `multimc`.
- Multi-loader jars are grouped by jar and loader, then counted once under the
  instance's active loader when present. Otherwise the loader order Fabric,
  Quilt, Forge, NeoForge is used deterministically.
- Multiple `[[mods]]` entries or multiple mods in `mcmod.info` remain separate
  entries; only loader variants of the same jar are collapsed by selection.
- Sinytra Connector is detected by mod id `connector` or a name containing
  "sinytra connector". On Fabric/Quilt instances, enabled Connector plus
  Forge/NeoForge jars reclassifies those mods as `<instance loader>/connector`.
- Disabled jars (`*.jar.disabled`) are parsed and returned with
  `StatusDisabled`; they are counted separately from loader totals.
- Nested jars are ignored unless `Options.IncludeNested` is set. Nested results
  use the nested entry path as `Jar` and `StatusNested`.
- Unrecognised jars get `Recognized=false`, empty loader/id, version `unknown`,
  and a warning; the scan continues.

## Phase 2 decisions

- Flags may appear before or after the positional instance; `cli.splitArgs`
  separates flag tokens from positionals so `mcmodlist <instance> --format json`
  works with the standard `flag` package.
- `--force` only controls overwriting `--output`; without it an existing output
  path exits `2`.
- Colour `auto` treats a character-device stdout as a terminal and honours
  `NO_COLOR`; `always`/`never` override. Pretty output currently uses colour
  only for the summary block when enabled.
- JSON keeps the frozen schema: compatibility layers are encoded in `mods[].loader`
  as `loader/layer` (for example `fabric/connector`), and `counts.connector`
  sums all `*/connector` labels. Unrecognised mods keep the schema `status`
  enum and are identified by empty loader plus `counts.unrecognised`/warnings.
- Pretty defaults to width 100 when no terminal width is injected; tests pass a
  wider `Options.Width` for stable goldens.

## Phase 3 decisions

- Disabled mods are hidden by default and shown with `--include-disabled`;
  nested mods require `--include-nested`, and `--recursive` scans mods
  subdirectories using slash-separated relative jar names.
- Filtering/sorting live in `core` (`FilterReport`, `SortMods`, `CountMods`) so
  the CLI and future GUI share the same logic. Loader counts, totals, and JSON
  `counts.*` are recomputed for the filtered view; this is documented in README.
- `--group-by` is a rendering concern: table formats group rows under headings,
  CSV adds a leading `Group` column, and JSON keeps the frozen schema.
- `--launcher` is only a filter for `--list-instances`; launcher detection for a
  scanned instance remains automatic. `--list-instances` searches common
  per-OS launcher directories and is best-effort.
- `--copy` uses an injectable `cli.Clipboard`; the default tries `clip`,
  `pbcopy`, then `wl-copy`/`xclip`/`xsel` and reports a clear error when no
  clipboard tool exists.
- Progress is a stderr `\r` update only when stderr is a TTY and the pack has at
  least 50 jars; piped output stays clean.
- Rendered output is passed through `strings.ToValidUTF8` with `�` fallback
  before writing or copying.

## Phase 4 decisions

- `core.Check(report)` returns deterministic `model.Issue` values for four
  diagnostic types: `duplicate-id`, `loader-mismatch`, `missing-dependency`,
  and `minecraft-version`. Duplicate ids are compared case-insensitively and
  only count when they appear in different jars. Loader mismatches are only
  reported when the instance loader is known and the enabled mod's effective
  loader differs; Connector-reclassified Forge/NeoForge mods are not mismatches.
  Dependency checks use declared required dependencies from Fabric/Quilt
  `depends` and Forge/NeoForge `dependencies` tables, excluding minecraft, java,
  loader, forge/fml, and neoforge entries. Minecraft constraints are evaluated
  with a small best-effort parser (`||`, comma/space AND terms, `>=`, `<=`,
  `>`, `<`, `=`, `==`, exact versions, and `x`/`*` prefixes), not a full semver
  library.
- `core.Diff(old, new)` compares recognised, non-nested mods by case-insensitive
  mod id, falling back to name+jar when id is empty. Unchanged mods are
  omitted; removed entries keep the old mod, added/version-changed entries keep
  the new mod. Version changes are case-insensitive and record `OldVersion`.
  The diff report reuses the first instance header and recomputes counts;
  the CLI then applies the same filters/sort as normal output.
- JSON schema changes are additive only: `mods[]` may include `change` and
  `oldVersion` for diff entries, and `issues` is populated when `--check` is
  used. No existing fields or enums changed.
- Config lives at `os.UserConfigDir()/mcmodlist/config.toml` and supports
  `format`, `columns`, and `color`. Config values are used as flag defaults, so
  explicitly parsed flags override them; malformed TOML or invalid values print
  a stderr warning and are ignored.
- `mcmodlist completion <shell>` prints static completion scripts for bash,
  zsh, fish, and powershell. `completion` is handled as a command before
  instance resolution so it never scans a directory named `completion`.

## Phase 5 decisions

- GUI toolkit: Wails v2 (`github.com/wailsapp/wails/v2` v2.17.1), chosen by the
  user. **Superseded by Phase 5R:** the GUI is now a native C++/Qt 6 app talking
  to a Go engine over JSON-Lines. The GUI lives in `gui/` plus
  `cmd/mcmodlist-gui` until Phase 5R R6 removes it; `cmd/mcmodlist` does not
  import it, so building the CLI alone does not compile Wails.
- GUI business logic stays in `gui` as pure view-model functions over `core`;
  Wails bindings are a thin shell. `mcmodlist --gui` runs the separate
  `mcmodlist-gui` binary when it is on `PATH` and otherwise prints an install
  hint.
- `model.Mod` gained optional `Description`, `License`, and `Links` fields so
  the GUI details pane can show metadata. The CLI JSON renderer does not emit
  them, so the frozen JSON schema is unchanged.
- GUI Export/Copy use simple GUI-local serializers for markdown/json/csv/text/
  html rather than importing `cli/render`, keeping `gui` independent of `cli`.
- Loader-count ordering moved to `core.OrderedLoaderLabels` so the CLI renderer
  and GUI share the Fabric/Quilt/Forge/NeoForge order instead of duplicating it.
- GUI state exposes `loaderOptions` from the unfiltered report so loader chips
  remain available while a loader filter is active; `loaders` still reflects
  the filtered view used by the header count line.
- Wails-bound `App` methods serialize access to the current report/state with
  a mutex; compare keeps the original scan as the diff/check base while the
  visible state becomes the diff report.

## Phase 5R decisions (supersedes the Phase 5 Wails toolkit)

- **Decision:** Go stays the engine. A new C++/Qt 6 Widgets app is the GUI.
  They talk over a versioned JSON-Lines protocol on the engine's stdin/stdout.
  Reason: the GUI must follow the global Qt theme (palette, style, accent color,
  live theme switches under KDE, `qt6ct`, GNOME Qt platform themes, Windows, and
  macOS); a Wails webview only sees `prefers-color-scheme` and cannot do this.
- Rejected: Go-to-Qt bindings such as miqt (cgo, slow builds, moving binding
  layer); rewriting `core` in C++ (re-proves jar/zip/TOML/JSON parsing for no
  user-visible gain); localhost HTTP API (ports, firewall prompts, process
  lifetime problems; a stdio child process has none); keeping Wails (does not
  meet the theming requirement).
- Environment check for R0: CMake 4.5.0, `qmake6`/Qt 6.12.0, and CMake configs
  for Qt6 Widgets/Test/Concurrent are installed, so Qt builds can be verified
  here rather than assumed.
- R1 engine notes: `engine/` owns the protocol, view-model, scan store
  (keeps 4 scans), cancellation, and rendering via `engine/render` (the CLI
  renderer is now a thin wrapper over it, so engine exports match the CLI
  byte-for-byte). `cancel` targets the request `id` of the scan/compare job.
  `compare` stores the diff as the scan's current view but keeps the
  original scan as the check/compare base, matching Phase 5 semantics.
  `rowId` values contain `\x1f` separators; protocol params must be built
  with a JSON encoder, not `fmt.Sprintf("%q")` (`\x` escapes are invalid JSON).
- R2 Qt notes: `qtgui/` uses only Qt6 Widgets/Test/Concurrent plus `QJsonDocument`
  for protocol parsing (no third-party JSON lib). `EngineClient` parses lines
  synchronously up to 256 KiB and hands larger lines to `QtConcurrent`; all
  pending-request/timer state stays on the GUI thread (oversized-line cleanup
  is queued back via `QMetaObject::invokeMethod`). The hello protocol check
  uses an atomic request id so the worker path never touches GUI state.
  Test stub engines are shell scripts under `qtgui/tests/` (raw string
  literals in the test .cpp broke `moc` parsing, so scripts live as files).
  `scripts/build-all.sh` builds both executables; ctest needs
  `MCMODLIST_ENGINE` pointing at `build/mcmodlist-engine` for the real-engine
  test (set automatically via CTest ENVIRONMENT).
- R3 Qt notes: `ModTableModel` stores engine rows verbatim and inserts group
  header entries at engine-provided boundaries; it never sorts or filters.
  Header clicks cycle asc → desc → default and send the sort spec to the
  engine (`field` / `-field`). `ViewController` implements latest-wins by
  dropping responses whose request id is not the newest pending view call.
  Group headers are painted by `ModTableDelegate` across the viewport using
  `AlternateBase`/`WindowText` palette roles. Loader chips are native
  checkable `QPushButton`s fed by `loaderOptions` (stable under filtering).
  Search is debounced 120 ms. Fixture jars in Qt tests are built by a
  minimal stored-ZIP writer in the test file.
- R4 Qt notes: `ModRow` gained `oldVersion` so diff rows can show
  old → new versions in the Compare report. Compare keeps the base scan as
  check/compare base (engine semantics); the table refreshes to the diff
  view and `ReportDialog` lists rows grouped by status. Export writes via
  `QSaveFile`; the GUI never formats output itself. Export byte-parity with
  the CLI is covered by the shared `engine/render` renderer (Go test in R1)
  plus Qt tests asserting content equality between `export` and `markdown`.
- R5 Qt notes: `EngineClient::stop()` now blocks QProcess signals before
  kill/waitForFinished so a deliberate stop (restart or destruction) cannot
  emit `engineExited` into a half-destroyed MainWindow (found via offscreen
  test segfault). `MCMODLIST_ENGINE` env override is persisted to QSettings
  `enginePath` on engine start so it survives restarts. Error banner gained a
  "Details…" button showing the last 200 captured engine stderr lines.
  Settings persisted via QSettings: geometry, splitter state, lastPath,
  enginePath. DetailsPane re-applies its BrightText warning palette on
  PaletteChange/ApplicationPaletteChange.
- R6: Wails GUI removed (`gui/`, `cmd/mcmodlist-gui`, frontend deleted;
  `go mod tidy` dropped all wails deps). The Qt `mcmodlist-gui` binary keeps
  the same name so `mcmodlist --gui` and `TestGUIHintWhenNotInstalled` work
  unchanged; the not-installed hint now points at `scripts/build-all.sh`.
  CI added at `.github/workflows/ci.yml` (Go checks + Qt configure/build/
  offscreen ctest + both policy greps).
- Phase 6 packaging answers (user-confirmed): license GPL-3.0-only (LICENSE
  added from SPDX text); no public repository yet, so PKGBUILD url/source are
  placeholders marked DRAFT and sha256sums must be regenerated with
  `updpkgsums`; maintainer spring <spring@localhost>; layout is a split
  package (`mcmodlist` CLI + `mcmodlist-gui` shipping `mcmodlist-gui` and
  `mcmodlist-engine` together so EngineLocator finds the engine beside the
  GUI) plus a single-binary `mcmodlist-git`.
- Version is now injectable: `cli.Version` and `engine.EngineVersion` are
  vars (were consts) set via `-ldflags "-X mcmodlist/cli.Version=$pkgver"` and
  `-X mcmodlist/engine.EngineVersion=$pkgver"`; verified `--version` prints
  the injected value and the binary is a PIE.
- CGO_ENABLED=1 in the PKGBUILDs so `-linkmode external` honours LDFLAGS;
  runtime dep is `glibc`. `mcmodlist-gui` adds `qt6-base` (Arch's Qt 6
  Widgets package) inside `package_mcmodlist-gui()` because split-package
  deps must be declared in the package function to reach .SRCINFO/.PKGINFO.
- `namcap` is not installed in this environment; PKGBUILD lint and package
  lint could not be run. Commands are in packaging/arch/README.md.
