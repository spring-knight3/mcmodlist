# mcmodlist

`mcmodlist` inspects a Minecraft **instance** directory and reports the mods it
contains: in-game name, mod id, version, author(s), `.jar` filename, loader,
compatibility-layer classification, diagnostics, and instance-to-instance diffs.

## Install

Build from source with Go:

```sh
go build -o mcmodlist ./cmd/mcmodlist
# or install into $GOBIN/PATH
go install ./cmd/mcmodlist
```

The tool is read-only: it never modifies an instance. The only files it writes
are explicit `--output` targets and the optional user config file described
below.

## Usage

```sh
mcmodlist [instance] [flags]
mcmodlist -i <instance> [flags]
mcmodlist --list-instances [--launcher <name>]
mcmodlist completion <bash|zsh|fish|powershell>
```

The instance defaults to `.`. Flags may appear before or after the instance
path. `--help` prints usage; if an instance path is also supplied, it first
prints whether that path looks like a valid instance.

### Instance and input flags

- `-i, --instance <path>` instance directory (default `.`; on a TTY in a
  non-instance directory an interactive picker is shown)
- `--launcher <name>` filter `--list-instances` by launcher
- `--list-instances` list known launcher instances and exit
- `--include-disabled` include disabled mods (hidden by default)
- `--include-nested` include mods nested inside jars
- `--recursive` scan subdirectories of the mods folder
- `--jobs <n>` jar scan worker count

### Output flags

- `-f, --format <format>` `pretty`, `markdown`, `csv`, `json`, `html`, or `text`
  (default `pretty`)
- `-o, --output <path>` write output to a file (`-` for stdout)
- `--force` overwrite an existing output file
- `--columns <list>` comma-separated columns: `name,id,version,author,jar,loader,side,status`
- `--sort <spec>` comma-separated sort keys; prefix `-` reverses a key
- `--group-by loader|side` group table output
- `--wide` do not truncate table columns
- `--truncate <n>` truncate table cells to `n` characters
- `--copy` copy rendered output to the clipboard
- `--no-summary` omit the instance summary
- `--summary-only` print only the instance summary
- `--no-header` omit headers for data-only output
- `--color auto|always|never` colour mode (default `auto`, honours `NO_COLOR`)
- `--no-color` disable colour

### Filter flags

- `--search <text>` filter by name/id/version/author/jar text
- `--loader <name>` filter by loader (`fabric`, `quilt`, `forge`, `neoforge`,
  `connector`, or `loader/layer`)
- `--side <name>` filter by side (`client`, `server`, `both`)
- `--author <name>` filter by author substring
- `--unrecognised` show only unrecognised jars

### Diagnostics and comparison

- `--check` detect duplicate mod ids across different jars, loader mismatches,
  missing declared dependencies, and wrong Minecraft versions. Findings are
  printed to stderr as `issue[...]` lines, included in JSON `issues`, and cause
  exit code `3` when present.
- `--diff <other-instance>` compare this instance with another instance and
  render only mods that were `added`, `removed`, or `version-changed`.
  Version changes render as `old -> new`; JSON adds `change` and `oldVersion`
  fields. Filters and sorting still apply to the diff view.

### Other flags and commands

- `-v, --verbose` print extra diagnostics to stderr
- `-q, --quiet` suppress warnings
- `-h, --help` show help
- `-V, --version` show version
- `--gui` launch the GUI if `mcmodlist-gui` is installed
- `completion <bash|zsh|fish|powershell>` print a shell completion script

### Config file

`mcmodlist` reads optional defaults from the platform config directory:

```text
~/.config/mcmodlist/config.toml        # Linux/macOS-style path via os.UserConfigDir
```

Supported keys:

```toml
format = "pretty"    # pretty, markdown, csv, json, html, text
columns = "name,id"  # comma-separated column subset
color = "auto"       # auto, always, never
```

Command-line flags override config values. A malformed config prints a warning
to stderr and is ignored rather than crashing.

## Examples

```sh
mcmodlist
mcmodlist ~/Games/Prism/instances/1.20.1-Fabric
mcmodlist /path/to/instance --format markdown --copy
mcmodlist /path/to/instance --format json --output mods.json --force
mcmodlist /path/to/instance --summary-only
mcmodlist /path/to/instance --search sodium --sort -version
mcmodlist /path/to/instance --check
mcmodlist /path/to/instance --diff /path/to/other-instance --format markdown
mcmodlist --list-instances --launcher prism
mcmodlist completion bash > ~/.local/share/bash-completion/completions/mcmodlist
```

## Output

Pretty/text/markdown/html show the instance name, Minecraft version, loader,
loader-count line such as `Fabric(2) Fabric/Connector(1) Forge(3)`, and total.
CSV is data-only. JSON always includes the instance object.

Disabled mods are hidden unless `--include-disabled` is used. Nested mods are
hidden unless `--include-nested` is used, and `--recursive` includes jars in
mods subdirectories. Loader counts, totals, and `counts.*` in JSON reflect the
mods shown after filtering. A footer note reports disabled mods and
unrecognised jars that are excluded from loader totals. Warnings and errors go
to stderr; stdout contains results only. Prompts and progress appear only on a
TTY.

## JSON schema

The Phase 2 schema is unchanged except for additive diff fields. `--diff`
entries may include `change` (`added`, `removed`, `version-changed`) and
`oldVersion` on mods; `--check` populates `issues`.

```json
{
  "tool": {"name": "mcmodlist", "version": ""},
  "instance": {"name": "", "path": "", "launcher": "", "minecraft": "",
               "loader": "", "loaderVersion": ""},
  "counts": {"fabric": 0, "quilt": 0, "forge": 0, "neoforge": 0,
             "connector": 0, "total": 0, "disabled": 0, "unrecognised": 0},
  "mods": [{"name": "", "id": "", "version": "", "authors": [], "jar": "",
            "loader": "", "side": "", "status": "enabled|disabled|nested",
            "change": "added|removed|version-changed",
            "oldVersion": ""}],
  "issues": [{"type": "", "mod": "", "message": ""}],
  "warnings": [{"type": "", "mod": "", "message": ""}]
}
```

`change` and `oldVersion` are omitted unless the mod came from `--diff`.
`issues` is populated by `--check`; `warnings` reports scan problems such as
corrupt or metadata-less jars.

## Exit codes

- `0` ok
- `1` invalid instance or unreadable path
- `2` invalid flags
- `3` `--check` found problems

## Paste into a support post

For a support post, include the tool version, the instance summary, the mod
list, and any check findings:

```sh
mcmodlist --version
mcmodlist /path/to/instance --summary-only
mcmodlist /path/to/instance --format markdown
mcmodlist /path/to/instance --check
```

Paste the command outputs into the post. If the instance is large, attach the
markdown output as a file instead of pasting the whole table.

## GUI

An optional native **Qt 6** GUI is available as a separate binary so the CLI
stays light. It follows the global Qt theme (palette, style, accent colour,
live theme switches under KDE, `qt6ct`, GNOME Qt platform themes, Windows,
macOS). The GUI is a dumb view: all scanning, filtering, sorting, grouping,
counting, and export formatting happen in the Go engine
(`mcmodlist-engine`), which the GUI drives over a versioned JSON-Lines
protocol on stdin/stdout (see `docs/PROTOCOL.md`).

Build both binaries:

```sh
scripts/build-all.sh
# produces:
#   build/mcmodlist-engine
#   build/qtgui/mcmodlist-gui
```

Run the GUI directly, or through the CLI launcher:

```sh
./build/qtgui/mcmodlist-gui
# or, when mcmodlist-gui and mcmodlist-engine are on PATH:
mcmodlist --gui
```

The engine is located in this order: `MCMODLIST_ENGINE` environment variable,
a saved `enginePath` setting (persisted automatically when the env var is
used), next to the GUI executable, then `PATH`.

The start screen lists instances found in known launcher directories and
supports Browse plus drag-and-drop. The main view has live search, loader
chips, side filter, disabled/nested/unrecognised toggles, loader/side
grouping, sortable columns, a details pane, and buttons for Check, Compare
with another instance, Copy as Markdown, and Export (Markdown, JSON, CSV,
text, HTML). Scanning runs in the background with progress and Cancel; window
geometry, splitter sizes, and the last path persist via `QSettings`.

Manual GUI checklist (real desktop — offscreen tests do not cover this):

- [ ] Start screen lists at least one known instance on a machine with a
      launcher install, or Browse/DnD opens a fixture instance.
- [ ] Table row count and loader counts match `mcmodlist <instance> --format json`.
- [ ] Search, loader chips, side filter, and toggles update the table.
- [ ] Selecting a row shows description, authors, license, links, jar path, and
      warnings when present.
- [ ] Check lists `issue[...]` findings; Compare shows added/removed/version-changed.
- [ ] Copy as Markdown places a table on the clipboard; Export writes the
      chosen format via the native save dialog.
- [ ] Switching the system Qt theme (KDE/qt6ct/GNOME) repaints chips, group
      headers, warnings, and badges without restart.
- [ ] Scanning a large instance keeps the window responsive.

## Library

`core` remains usable without the CLI:

```go
report, err := core.Scan(instancePath, core.Options{})
issues := core.Check(report)          // diagnostics for --check
other, err := core.Scan(otherPath, core.Options{})
diff := core.Diff(report, other)      // added/removed/version-changed view
```

`report` contains `Instance`, `Mods`, `Counts`, `Issues`, and `Warnings`.
Scanning is read-only and never aborts on a bad jar.

## Development

```sh
gofmt -l .
go vet ./...
go build ./...
go test ./... -race
```

Qt GUI (requires Qt 6.5+, CMake, a C++20 compiler):

```sh
scripts/build-all.sh
QT_QPA_PLATFORM=offscreen MCMODLIST_ENGINE=$PWD/build/mcmodlist-engine \
    ctest --test-dir build/qtgui --output-on-failure
```
