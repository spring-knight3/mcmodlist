# mcmodlist engine protocol

The GUI engine (`mcmodlist-engine`) speaks UTF-8 JSON-Lines on stdin/stdout.
One JSON object per line, newline terminated, no raw newlines inside a message.
stdin carries requests; stdout carries responses and events; stderr is logs
only. Closing stdin makes the engine exit cleanly.

## Envelope

Every message has `"v": 1`.

```json
{"v":1,"id":7,"method":"scan","params":{"path":"/tmp/instance"}}
{"v":1,"id":7,"result":{"scanId":"scan-1","summary":{}}}
{"v":1,"id":7,"error":{"code":"path_not_found","message":"...","detail":{}}}
{"v":1,"id":7,"event":"scan-progress","data":{"done":1,"total":9}}
```

`id` correlates requests, responses, and events. For malformed JSON that cannot
be parsed, the engine replies with `"id":0` and `invalid_request`.

## Methods

| Method | Params | Result |
|---|---|---|
| `hello` | optional `{client,clientVersion}` | `{protocol,engineVersion,methods}` |
| `listInstances` | optional `{launcher}` | `{instances:[InstanceOption]}` |
| `inspectPath` | `{path}` | `{valid,path,instance}` or a typed error |
| `scan` | `{path}` | async: `scan-progress` events, then `{scanId,summary}` |
| `cancel` | `{id}` | `{cancelled}` |
| `view` | `{scanId,view}` | `{state}` |
| `details` | `{scanId,rowId}` | `{mod,warnings}` |
| `check` | `{scanId}` | `{issues}` |
| `compare` | `{scanId,path}` | async diff summary; keeps the scan as compare/check base |
| `export` | `{scanId,format,view}` | `{format,content}` |
| `markdown` | `{scanId,view}` | `{content}` |
| `shutdown` | `{}` | `{bye:true}` |

`view` accepts the Phase 5 view state: `search`, `loader`, `side`, `author`,
`sort`, `groupBy` (`""`, `loader`, or `side`), `showDisabled`, `showNested`,
and `unrecognisedOnly`. Rows are display-ready and in final order; `groups`
contains group boundaries when `groupBy` is set. `loaderOptions` is computed
from the unfiltered scan so chips remain stable while a filter is active.

`export`/`markdown` render through the same renderer as the CLI for
`markdown`, `json`, `csv`, `text`, and `html`.

## Errors

Closed codes: `invalid_request`, `unsupported_version`, `path_not_found`,
`not_an_instance`, `unreadable`, `scan_cancelled`, `unknown_scan`, `internal`.

## Engine rules

- Scans are kept in memory keyed by `scanId` (default keep 4). `compare`
  updates the scan's current diff report but keeps the original scan as the
  check/compare base.
- Requests may overlap; `view` stays responsive while a scan runs. Scan and
  compare are cancellable with `cancel`.
- Row ids are stable within a scan and are derived from mod identity fields.
