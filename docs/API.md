# Factory HTTP API

Run `go run ./cmd/api`. The default address is `:8080`. Send JSON request bodies with `Content-Type: application/json`. When the server has any [API role token](../AUTH.md) configured, every `/api/` route requires an `Authorization: Bearer <token>` header. `GET /health` is public. Set `CORS_ORIGIN` to the operator UI origin for cross-origin browser requests.

## Goals and review

| Method and route | Behavior |
| --- | --- |
| POST /api/goals | Submit a description, factory ID and source records. Optional fields include `factory_revision`, `selected_rules` (rule IDs mapped to revisions), and `max_tokens`. Returns the persisted run with its ID. |
| GET /api/goals | List runs, newest first. |
| GET /api/goals/{id} | Read status, source records, decisions, model calls and pinned revisions. |
| DELETE /api/goals/{id} | Cancel a run. |
| GET /api/decisions?run={id} | List decisions with proposals, evidence and before/after values. Omit the filter for all runs. |
| GET /api/decisions/{id}?run={run} | Fetch one decision. An optional run filter narrows the lookup. |
| POST /api/decisions | Send `Run`, decision `ID`, current run revision `Expected`, and `Status` (approved or rejected). Optional `Feedback` creates a versioned rule governed by the run’s replay gate. |
| GET /api/browser-captures/{id} | Fetch the retained text, optional base64 JPEG, source URL and configuration revision referenced by a decision. [Browser tool](BROWSER_CAPTURE.md) |
| GET /api/graph?run={id}&offset=0&limit=100 | Page through source records, links to decisions and typed outgoing edges. Returns `nodes`, `edges` and total record count. |
| GET /api/usage?run={id} | Read token totals, estimated USD, model calls, unpriced calls and tool calls. Named tool counts are returned in `tool_calls_by_name`. Completed and failed continuous CLI runs appear here when they share the API’s durable workspace. Omit the run filter for workspace totals including chat tokens. |

A run filter on decisions, graph or usage returns HTTP 404 for an unknown run. For progress, compare the decision count with the source record count in the goal response. A decision ID contains a slash. Keep the full ID when constructing its lookup URL.

## Versioned definitions

Edits append a revision. Send the last observed revision in `Expected`, or zero to create a definition. A stale edit returns HTTP 409. Fetch history before retrying a stale edit.

| Method and route | Behavior |
| --- | --- |
| GET /api/rules | Return active rules. Add `?history=true` for all revisions or `?history=true&id={id}` for one rule. |
| POST /api/rules | Create or edit a proposal with `ID`, `Description`, `Condition`, `Action` and `Expected`. To activate a revision send `ID`, `Expected` and `Promote` (revision number). |
| GET /api/agents?name={name} | Read agent revision history. Omit the name for all agents. |
| POST /api/agents | Save an agent revision with `Name`, `Role`, `Expected` and optional model tier, tools, configuration or retirement state. |
| GET /api/definitions/{kind}/{id} | Read revision history for a factory, pack, tool or experience. |
| POST /api/definitions/{kind}/{id} | Append a document with `Expected` and a JSON object named `Data`. |
| GET /api/settings/schema | JSON Schema for workspace settings, including defaults and bounds. |
| GET /api/settings | Read gateway name, workspace defaults and revision. Add `?history=true` for settings revisions. |
| POST /api/settings | Save `Expected` and a `Settings` object with max_rounds, max_tokens, max_minutes, replay_gate and observe. New runs snapshot the settings. |

## Chat and feedback

`POST /api/chat` accepts `message`. “Create a factory to …” returns a new factory ID and definition. “Start a run to …” also accepts a `source` object with the same factory, records and budget fields as goal creation. “What is the status of the latest run?” reads the workspace. Status for a particular run can be requested with `Status <run ID>`.

`POST /api/feedback` accepts a `note` and optional `entity_id` for standalone feedback. The response includes the new rule ID, revision and activation state. With the workspace replay gate enabled, feedback creates a proposal for review. With the gate disabled, the rule becomes active immediately. Use the decision endpoint when feedback belongs to a particular review.


### Goal attachments

Upload a file to `POST /api/chat/artifacts` with `name` and base64-encoded `data`, or submit a GitHub `url` to `POST /api/chat/github`. Pass the returned artifact IDs and versions in the chat request’s `artifacts` array. With sources attached, goal creation returns a reviewable proposal. To revise it, send its `proposal` ID and `expected` draft revision with the selected sources. Approve the reviewed revision with `POST /api/chat/proposals/{id}/approve` and `{"version":1}` (substitute the reviewed version). Each criterion cites a source location. [Supported formats and limits](../README.md#goal-chat-attachments)

`GET /api/releases/{id}` returns the immutable proposed release history for a pack name in the current workspace. Each revision contains the Git commit, pack and CI evidence bytes, and their SHA-256 digests. Viewer access suffices. [Publishing a pack](RELEASE.md#publish-a-proposed-pack-revision)
