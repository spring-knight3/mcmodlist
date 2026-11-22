# AGENTS.md

Binding instructions for any AI agent or contributor working on **mcmodlist**.
If this file conflicts with a request in chat, stop and ask.

## 1. Project summary

`mcmodlist` inspects a Minecraft **instance** (not merely its `mods/` folder)
and reports the mods it contains: in-game name, mod id, version, author(s),
`.jar` filename, and loader, plus a loader-count summary and the instance name.

Build plan: see `PHASES.md`. Starting brief: see `PROMPT.md`.

## 2. STOP PROTOCOL (most important rule)

Work is split into phases in `PHASES.md`. After finishing a phase you MUST:

1. Run all checks in section 8 and confirm they pass.
2. Tick the completed items in `PHASES.md`.
3. Output a **phase report** (format in section 9).
4. **STOP.** Do not start, plan in code, scaffold, or "get ahead" on the next
   phase. No files for future phases, no stubs, no TODO scaffolding beyond what
   the current phase needs.
5. Wait until the user explicitly says to continue (for example
   "Continue to Phase 2"). Silence, thanks, or a question is NOT permission.

If you are unsure whether a phase is finished, it is not: report what is left
and stop.

## 3. Architecture

```
mcmodlist/
  cmd/mcmodlist/      main package; wires cli to os.Args, nothing else
  core/               library: detection, scanning, classification, diff, check
    instance/         launcher metadata parsers + instance detection
    jar/              streaming jar reader + per-loader metadata parsers
    model/            shared structs (Instance, Mod, Counts, Issue)
  cli/                flag parsing, renderers, prompts, completions
    render/           pretty, markdown, csv, json, html, text
  engine/               JSON-Lines protocol + GUI view-model (imports core only)
  qtgui/                native Qt 6/C++20 GUI; talks to engine/, never to Go
  packaging/arch/       (final phase only) PKGBUILD(s), .SRCINFO, packaging notes
  testdata/           small hand-written metadata files; jars are built in tests
  PROMPT.md  AGENTS.md  PHASES.md  DECISIONS.md  README.md
```

Rules:

- `core` returns structured data. It never prints, never reads flags, never
  touches the terminal, and never formats strings for display.
- `cli` and `engine` depend on `core`. `qtgui` knows only the JSON-Lines
  protocol. `core` depends on neither.
- No logic is duplicated between front ends. If the CLI and GUI both need it,
  it belongs in `core`.
- Parsers are isolated per launcher and per loader behind small interfaces, so
  adding one never edits the others.

## 4. Behavioural requirements (the contract)

- **Read-only.** Never write, move, rename, or delete anything inside an
  instance. The only files the tool may write are `--output` targets and its
  own optional config file.
- **Never abort on a bad jar.** Corrupt, unreadable, or metadata-less jars are
  reported as `<filename> (unrecognised)` with a warning; the run continues.
- **Multiple mods per jar:** each is its own entry and carries the `.jar` name.
- **Multi-loader jars** are counted once, under the instance's active loader.
- **Counts are mods, not jars.**
- **Loader line** shape: `Fabric/Connector(<n>) Quilt(<n>) Forge(<n>)
  NeoForge(<n>)`; only loaders with at least one mod are shown, in the order
  Fabric, Quilt, Forge, NeoForge. Compatibility layers are shown as
  `<Loader>/<Layer>`, for example `Fabric/Connector`.
- **Instance name is always shown** (except in `--no-summary`, `csv`, and
  `--no-header` data-only modes; JSON always includes it).
- **Jar contents win** over launcher metadata for per-mod loader
  classification; metadata is used for the instance header only.
- **Version placeholders** such as `${file.jarVersion}` resolve from
  `META-INF/MANIFEST.MF`; otherwise show `unknown`.
- **Deterministic output.** Stable sort (default: name, case-insensitive, then
  id, then jar) so output can be diffed and golden-tested.
- **stdout is for results only.** Warnings, progress, and errors go to stderr.
- **Exit codes:** `0` ok, `1` invalid instance or unreadable path, `2` invalid
  flags, `3` `--check` found problems.
- **Prompts only on a TTY.** Never prompt or draw progress when output is
  piped.

## 5. Code conventions

- Go, latest stable. `gofmt` and `go vet` clean. Add `golangci-lint` config
  if the user approves.
- Standard library first. Each new dependency needs a one-line justification in
  `DECISIONS.md`. Prefer small, well-maintained libraries (TOML, a CLI flag
  library, a terminal library).
- Errors are values: wrap with context (`fmt.Errorf("reading %s: %w", ...)`).
  User-facing messages say what is wrong and how to fix it; no stack traces.
- Stream only the needed jar entries; never extract whole archives to disk.
- Scan jars concurrently with a bounded worker pool (`--jobs`).
- Cross-platform: paths via `filepath`, tested on Windows, macOS, and Linux
  conventions. No shelling out for things the standard library can do.
- Exported identifiers get doc comments. Keep functions short and named for
  what they do.
- No global mutable state. Pass dependencies explicitly.

## 6. Testing

- Every feature ships with tests in the same phase.
- Build fixture jars **programmatically in test helpers** (using `archive/zip`)
  instead of committing binary blobs. Small hand-written metadata files may
  live in `testdata/`.
- Required fixtures over the project's life: one jar per loader, a multi-mod
  jar, a multi-loader jar, a corrupt jar, a metadata-less jar, a Jar-in-Jar
  jar, a disabled jar, a Connector setup, and one instance per supported
  launcher.
- Renderers use golden-file tests; update goldens only with an explanation.
- Table-driven tests preferred. Tests must be hermetic: no network, no real
  user directories, no reliance on a real Minecraft install.

## 7. Boundaries

**Always:** follow this file, keep `core` pure, write tests, update docs when
behaviour changes, log non-obvious decisions in `DECISIONS.md`.

**Ask first:** adding a dependency, changing the output shape or JSON schema
after Phase 2, changing exit codes, adding a flag not in `PHASES.md`, anything
networked.

**Never:** modify an instance, make network requests (no Modrinth/CurseForge
API calls unless a future phase explicitly adds opt-in `--online`), commit
secrets or real user data, skip or delete a failing test to make a phase pass,
start the next phase without permission.

## 8. Checks before reporting a phase done

Run and require all of these to pass:

```
gofmt -l .          # no output
go vet ./...
go build ./...
go test ./... -race
```

Plus the phase's own acceptance criteria in `PHASES.md`, verified by running
the tool where applicable (not just by reading the code).

## 9. Phase report format

End every phase with exactly this, then stop:

```
## Phase N report
**Status:** complete | incomplete
**Done:** bullet list of what was built
**Acceptance criteria:** each criterion with pass/fail and how it was verified
**Checks:** results of the section 8 commands
**Decisions / deviations:** anything logged in DECISIONS.md
**Known gaps / risks:** honest list, or "none"
**Next:** one line naming the next phase.
Waiting for you to say "Continue to Phase N+1".
```

## 10. Communication

- Be concise and factual. Do not claim something works unless you ran it.
- If the spec is ambiguous or two rules conflict, ask; do not guess silently.
- Prefer small, reviewable changes over large rewrites.
- Keep `README.md` accurate for what exists *now*, not what is planned.
