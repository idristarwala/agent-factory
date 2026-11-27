# Rendered page evidence

Workspace factories can read a page referenced by a record, use its rendered text and an optional screenshot during evaluation, and retain the exact capture for later review and replay. The browser uses a fresh context for each record.

## Install the optional tool

On the workspace API host, install Node.js 24 and a sandbox-capable Chromium installation. Run the following commands from the repository root.

```sh
cd extensions/browser-capture
npm ci
npx playwright-core install --with-deps chromium
export FACTORY_PLAYWRIGHT_SCRIPT="$PWD/capture.mjs"
```

Start the API in that environment. For a separately installed Chromium, set `FACTORY_CHROMIUM_EXECUTABLE` to its absolute executable path. The renderer launches Chromium with its sandbox enabled. The API supplies record and tool configuration through standard input. Model keys, API tokens and user browser sessions are excluded from the child environment. Keep the capture script and its dependencies under operator control.

## Configure a factory

Save the following documents through the versioned definition API. Use each document's current revision as `Expected` when changing it.

| Definition | Example Data |
|---|---|
| `tool/page-reader` | `{"kind":"playwright","side_effect":"read-only","url_field":"reference_url","origins":["https://docs.example.org"],"screenshot":true,"timeout_ms":10000,"max_text_bytes":8192}` |
| `pack/desk` | `{"grants":[{"pack":"desk","tools":["page-reader"],"side_effects":["read-only"]}]}` |
| `factory/desk` | `{"record_type":"ticket","fields":{"team":"string"},"browser_tool":"page-reader"}` |

Submit a ticket with `reference_url` and `team` in its attributes. A second factory can grant the same tool to its own pack and select different editable fields. A workspace run pins factory, pack and tool revisions at creation. Later edits apply to new runs.

Origins are exact HTTP or HTTPS origins with optional ports. The source URL, redirects and downloaded subresources must satisfy the origin policy. The renderer fetches GET resources without cookies or credentials and limits redirects to five, total requests to 64 and downloaded content to 4 MiB. It blocks writes, pop-up and frame navigations, workers and WebSockets. The initial document must be HTML. The default deadline is 10 seconds (range 1–60000 ms), and rendered text defaults to 8192 bytes (range 1–32768). A requested screenshot captures the 1024 × 768 viewport in JPEG, up to 512 KiB. A failed capture fails the run before the model evaluates the record.

## Inspect or replay a capture

Open the record's decision and expand **Rendered page capture**. The UI reads the immutable capture by ID. The capture contains the source and final URLs, UTC capture time, tool and pack revisions, visible text and optional JPEG. The worker and judge receive the same capture identifier, text and image. The run counts capture attempts under the configured tool name. Held-out rule replay reads the saved capture even if the source changes.

Harnesses can read the decision's `browser_capture` ID and fetch `GET /api/browser-captures/{id}` using the existing API credentials. The `image` field contains base64 JPEG when configured. [API](API.md) · [Configuration](CONFIGURATION.md)
