# Configuration

## Deployment settings

The API and CLI read deployment configuration from their process environment. Restart a process after changing its environment. CLI flags override the corresponding environment defaults. The UI reads its Vite setting at development startup or build time.

| Setting | Type and default | Scope and precedence |
| --- | --- | --- |
| `FACTORY_STORE_MODE` | String, unset | `memory` selects process-local state ahead of all database settings. |
| `DATABASE_URL` | PostgreSQL connection string, unset | Selects PostgreSQL ahead of SQLite. Share the same database across processes that share work. |
| `FACTORY_DB_PATH` | File path, user config directory plus `agent-factory/factory.sqlite` | SQLite path when both higher-priority modes are absent. |
| `FACTORY_ID` | String, unset | Workspace namespace for stored definitions and revisions. Use the same value in the API and CLI. |
| `FACTORY_LISTEN_ADDR` | Host and port, `:8080` | API listener. Use `127.0.0.1:8080` for loopback only. |
| `API_TOKEN` | Secret string, unset | Enables bearer authentication on API routes. Health stays public. |
| `CORS_ORIGIN` | Origin string, unset | Permits the configured browser origin or an explicit wildcard. |
| `VITE_API_URL` | URL, `http://localhost:8080` | Operator UI’s API base URL. |
| `FACTORY_DEFINITION`, `FACTORY_RECORDS` | ID and JSON file path, unset | Defaults for goal CLI flags `-factory` and `-records`. `-revision` defaults to 0, selecting the latest revision. |
| `FACTORY_HTTP_ALLOWED_HOSTS` | Comma-separated hostnames, unset | Restricts outbound tool destinations by exact hostname. An explicitly empty value allows no hosts. Other destination checks still apply. |
| `FACTORY_SOURCE_CONFIG` | YAML path, unset | Enables named read-only source sampling in the API. [Source mapping](MAPPING.md) |
| `FACTORY_EVIDENCE_CONFIG`, `FACTORY_PACK` | YAML path and pack name, both unset | Set both to enable a continuous runner evidence recipe and its pack grants. [Tools](HTTP_TOOLS.md) |
| `FACTORY_MODEL_PRICES` | JSON object, empty | Per-model USD per million input and output tokens. Applied when a call is recorded. |
| `SECRETS_DIR` | Directory, `/etc/secrets` | Central loader checks named files first, then the uppercase environment name with hyphens converted to underscores. Explicit gateway and tool key-file paths use their own configuration. |
| `FACTORY_GITHUB_REPOSITORIES` | Comma-separated owner/repository names, empty | Limits the repositories that receive the GitHub attachment credential on an authenticated API. |
| `FACTORY_GITHUB_TOKEN` | Secret string, unset | Fallback for the `factory-github-token` secret file when importing authorized GitHub sources. |

## Model routing

| Setting | Type and default | Scope and precedence |
| --- | --- | --- |
| `FACTORY_JUDGE_ROUTES` | Comma-separated logical model route names, unset | Continuous multi-judge panel. Provide one distinct configured model per evaluation. [Panel configuration](ENTERPRISE_GATEWAY.md#independent-judge-panels) |
| `LLM_GATEWAY_CONFIG` | YAML path, unset | Selects configured logical routes before environment-only provider selection. [Gateway guide](ENTERPRISE_GATEWAY.md) |
| `LLM_PROVIDER` | Provider name, autodetected | Supports Gemini, Anthropic, OpenAI-compatible, Groq and OpenRouter. With no explicit provider or base URL, a Gemini or generic key selects Gemini. |
| `LLM_BASE_URL`, `LLM_API_KEY` | Endpoint URL and credential, unset | Used together for environment-only OpenAI-compatible endpoints. The generic key can also supply the Gemini credential. |
| `GEMINI_API_KEY` | Secret string, unset | Native Gemini worker or judge credential. |
| `GEMINI_CHEAP_MODEL`, `GEMINI_FRONTIER_MODEL` | Model names, both `gemini-3.5-flash-lite` | Select Gemini worker and frontier models. |
| `ANTHROPIC_API_KEY` | Secret string, unset | Native Anthropic credential. |
| `ANTHROPIC_CHEAP_MODEL`, `ANTHROPIC_FRONTIER_MODEL` | Model names, `claude-3-haiku-20240307` and `claude-3-5-sonnet-20241022` | Environment-only Anthropic routes. Choose models available to your account. |
| `OPENROUTER_API_KEY` | Secret string, unset | Direct OpenRouter access for worker or independent judge. |
| `LLM_CHEAP_MODEL`, `LLM_FRONTIER_MODEL` | Model names | Direct OpenRouter defaults to `gpt-4o-mini`. Custom OpenAI-compatible endpoints default to `llama-3.1-8b-instant`. Frontier defaults to the cheap model. |
| `LLM_JUDGE_PROVIDER`, `LLM_JUDGE_MODEL` | Provider and model names, both unset | Set together for independent judge routing. Otherwise judge calls use the primary provider. |

## Versioned workspace settings

Read `GET /api/settings/schema` for a machine-readable JSON Schema with types, bounds, defaults and descriptions. Read `GET /api/settings` for effective workspace values, then post the complete settings object with its current revision as `Expected`. A successful edit appends a revision. A run pins the settings it reads at startup.

| JSON field | Type and default | Effect |
| --- | --- | --- |
| `max_rounds` | Positive integer, 10 | Continuous round budget. The CLI uses 5 until a settings revision exists. |
| `max_minutes` | Positive integer, 30 | Continuous wall-clock budget, up to 525600 minutes. The CLI uses 10 until a settings revision exists. |
| `max_tokens` | Positive integer, 1000000 | Model token budget. The continuous runner checks at round boundaries. |
| `observe` | Boolean, false | Records an approved decision while skipping its graph write in the continuous runner. |
| `replay_gate` | Boolean, false | Holds learned rule proposals for review or replay under the applicable runner. |
| `worker_count` | Integer 1–256, default 2 | Concurrent continuous workers per round. |
| `max_failures` | Positive integer, 20 | Cumulative worker failures before stopping at the next round boundary. |
| `stagnant_rounds` | Positive integer, 2 | Consecutive unchanged graph comparisons before stopping. |
| `judge_evaluations` | Integer 1–32, default 1 | Distinct judge evaluations per continuous proposal when using a live gateway. Each can incur model cost. |
| `judge_approvals` | Integer 1 through `judge_evaluations`, default 1 | Required number of approving evaluations. |

Stored revisions without quorum fields retain their original two-evaluation, two-approval requirement. Supply distinct judge routes for those workspaces or save a new settings revision to change the policy. Older clients editing the original settings preserve the current values for omitted continuous-runner fields.

Factory schemas, source mappings, rules, tool grants and evidence recipes have their own scoped contracts. [Definition API](API.md) · [Source mappings](MAPPING.md) · [Tool limits](HTTP_TOOLS.md)

### Independent judge model

Set `LLM_PROVIDER` for the worker and other ordinary calls. To route only judge calls to a different model, set both `LLM_JUDGE_PROVIDER` (gemini or openrouter) and `LLM_JUDGE_MODEL`. The judge uses its provider’s dedicated runtime credential, `GEMINI_API_KEY` or `OPENROUTER_API_KEY`. Run history records the role, served model and tokens for each worker and judge call.

For a Gemini worker with a GPT-4o judge, use `LLM_PROVIDER=gemini`, `LLM_JUDGE_PROVIDER=openrouter`, and `LLM_JUDGE_MODEL=openai/gpt-4o`. For the reverse, use `LLM_PROVIDER=openrouter`, `LLM_CHEAP_MODEL=openai/gpt-4o`, `LLM_JUDGE_PROVIDER=gemini`, and `LLM_JUDGE_MODEL=gemini-3.5-flash-lite`. With both credentials available, `make test-mixed-judges` executes both configurations through human approval.

## Factory memory files

Set `FACTORY_STORE_MODE=memory` for an entirely process-local API, including runs, definitions and memory files. This takes precedence over database settings. Each process starts empty. For shared durable state, leave that variable unset and set `DATABASE_URL` to PostgreSQL. Without either setting the API uses SQLite.

Memory uses relative virtual paths scoped to a factory. The following calls use the same API authentication as goals.

| Operation | Request |
| --- | --- |
| Create a file | POST `/api/memory/desk` with `{"path":"notes/routing.txt","content":"Route access requests to Identity.","expected":0}` |
| Revise a file | POST the same path with updated content and its current revision as `expected` |
| List files beneath a directory | GET `/api/memory/desk?directory=notes` |
| Read latest or historical content | GET `/api/memory/desk?path=notes/routing.txt` with optional `&version=1` |
| Attach memory to a run | Include `"memory":[{"path":"notes/routing.txt","version":1}]` in the goal request |

Omitting the version from a goal's memory selection pins the latest revision when the run starts. Up to 16 selected files (64 KiB combined) are copied into its plan. Worker and judge receive identical memory evidence with the path, revision and author. Paths are virtual and directories are prefixes. Access to a factory's memory API follows the workspace's existing token policy.

Human review feedback automatically creates a file at `reviews/{run-id}/{decision-index}.txt`, available to later runs by explicit selection. Notes larger than the 32 KiB file limit have a reference to the full decision. Ordinary runs use only the memory selected for them.

## Storage contract for API changes

Source profiling and previews are transient. Steward mapping drafts and approved mappings are immutable configuration documents. Runs store the chosen mapping revision, normalized record snapshot and pinned memory. File-like memory has its own immutable revision collection. Adding a factory domain or editable record property uses these existing representations.

PostgreSQL and SQLite initialize from shared table definitions. For work units, the additive `unit_payloads` table carries the JSON work scope, budgets, field constraints and evidence. Status and attached proposal updates use independent columns on `units`. Older rows without a payload remain readable with the data originally stored. Inline model image bytes are transient; evidence labels and digests remain available.

For an endpoint that introduces durable state, define its identity, lifetime, scope, revision or conflict behavior and atomic updates in the storage contract first. Exercise that contract in RAM and PostgreSQL, with a legacy-row check for migrations. Introduce a physical migration when the existing representation cannot enforce the required behavior or serve the needed query.

Continuous runs store a chronological `audit` array in the run revision. A worker proposal includes its mutation and evidence references. Judge entries record the verdict and reason. Runner entries distinguish observed, applied and failed writes. Match an entry’s `trace_id` to the `unit_id` on each model call to inspect retries, quorum evaluations and token use for that work unit.

## Per-agent model limits

Edit these fields on the worker or judge in **Agents**, or send them as string values in the agent definition’s `Config` object through `POST /api/agents`. Supply the current head revision as `Expected`. Workspace runs use the pinned agent revision. The continuous CLI resolves its judge at startup and its worker when the steward creates the worker.

| Config key | Role | Type and default | Effect |
| --- | --- | --- | --- |
| `max_output_tokens` | Worker or judge | Positive integer encoded as a string. Worker default 16000, judge default 300. | Sets the output limit on each model request. Choose a value supported by the provider and model. |
| `max_attempts` | Worker | Integer string from 1 to 16, default 2 | Maximum worker attempts to obtain a valid proposal. Configured gateway transport retries apply separately. |

For example, `{"max_output_tokens":"1024","max_attempts":"1"}` requests a single worker attempt with at most 1024 output tokens. Omit a limit to use its default. The workspace run token budget still applies across calls. [Versioned definitions](API.md#versioned-definitions)

The continuous CLI accepts repeated `-field FIELD` arguments before its goal. They select existing editable record properties for this invocation. A narrow literal task can only select a field within the supplied list. The runner owns `status` for continuous completion tracking. [Write scope and examples](CONTINUOUS.md#choose-editable-fields)
