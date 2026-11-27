# Configuration

## Deployment settings

The API and CLI read deployment configuration from their process environment. Restart a process after changing its environment. CLI flags override the corresponding environment defaults. The UI reads its Vite setting at development startup or build time.

| Setting | Type and default | Scope and precedence |
| --- | --- | --- |
| `FACTORY_STORE_MODE` | String, unset | Select `memory`, `sqlite`, or `postgres` at process startup. Explicit selection takes precedence over inference. Unknown values fail startup. |
| `DATABASE_URL` | PostgreSQL connection string, unset | Required for explicit PostgreSQL. Selects PostgreSQL when mode is unset. Share the same database across processes that share work. |
| `FACTORY_DB_PATH` | File path, user config directory plus `agent-factory/factory.sqlite` | Used whenever SQLite is selected. |
| `FACTORY_SQLITE_BUSY_TIMEOUT_MS` | Integer milliseconds, `5000` | SQLite only. Each connection waits up to this long for a lock. Zero returns immediately. Range 0–2147483647. Restart after changing. |
| `FACTORY_ID` | String, unset | Workspace namespace for stored definitions and revisions. Use the same value in the API and CLI. |
| `FACTORY_LISTEN_ADDR` | Host and port, `:8080` | API listener. Use `127.0.0.1:8080` for loopback only. |
| `FACTORY_API_READ_TIMEOUT` | Go duration, `15s` | API server deadline for reading a request, including its body. Zero disables this deadline. |
| `FACTORY_API_WRITE_TIMEOUT` | Go duration, `30s` | API server response write deadline. Allow time for synchronous model calls such as chat and source proposals. Zero disables this deadline. |
| `FACTORY_API_IDLE_TIMEOUT` | Go duration, `60s` | Time to wait for the next request on a keep-alive connection. Zero inherits the read timeout. |
| `COMPOSE_FILE` | Compose file path(s), unset | Used by Docker Compose for `factory up`. An explicit `-f FILE` takes precedence. |
| `API_TOKEN` | Secret string, unset | Admin bearer token. Any configured role enables authentication. Tokens must be distinct. Health stays public. |
| `FACTORY_OPERATOR_TOKEN` | Secret string, unset | Operator bearer token for running factories and reviewing work. |
| `FACTORY_VIEWER_TOKEN` | Secret string, unset | Viewer bearer token for read access to workspace data. |
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
| `FACTORY_PLAYWRIGHT_SCRIPT` | Absolute path, unset | Enable rendered page capture for workspace factories that select a granted browser tool. The API resolves Node.js through its process `PATH`. [Configuration and limits](BROWSER_CAPTURE.md) |
| `FACTORY_CHROMIUM_EXECUTABLE` | Executable path, unset | Optional system Chromium. Defaults to the browser installed by playwright-core. |
| `FACTORY_REPLAY_RUN` | Reviewed workspace run reference `RUN_ID@REVISION`, unset | With `replay_gate=true`, compare new steward rules against human-reviewed records. Pass `-review-replay` to require human approval after a gain. [Replay procedure](RULES_REFRESH.md) |
| `LLM_RETRY_ATTEMPTS` | Integer 1–16, default 3 | Total transport attempts per environment-routed model call, including the first. Set to 1 for a single attempt. |
| `LLM_RETRY_BASE_DELAY` | Positive Go duration, default 250ms | Initial backoff after a retryable HTTP 429 or 5xx. |
| `LLM_RETRY_MAX_DELAY` | Go duration from the base delay to 24h, default 5s | Return the provider error when the next delay exceeds this ceiling. |
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

## Configuration files

| Input | Scope and effective time | Field reference |
| --- | --- | --- |
| Model gateway YAML | Read at process startup when selected by `LLM_GATEWAY_CONFIG`. Model settings select a protocol and logical route. Authenticated requests reread mounted keys. | [Gateway routes, headers, authentication and retries](ENTERPRISE_GATEWAY.md) |
| Evidence tool and source YAML | Loaded by the selected API or runner at startup. A continuous run applies the selected pack's grants and context recipe when gathering evidence. | [Tool fields and limits](HTTP_TOOLS.md) · [Tool schema](../schemas/evidence-tools.schema.json) |
| Pack and rule YAML or JSON | The CLI reads the selected pack files for each validation or working-tree fixture invocation. The receipt workflow reads the committed revision. Omitted rule activation defaults to active. | [Pack schema](../schemas/pack.schema.json) · [Rule schema](../schemas/rule.schema.json) |
| Factory definition JSON | An import creates a revision. A workspace run selects and pins a definition revision when it starts. The definition describes record fields, worker role and evidence inputs. | [Factory definitions](DOMAIN_PACKS.md#configure-your-own-factory) · [Browser evidence settings](BROWSER_CAPTURE.md#configure-a-factory) |

## CLI options

Options below apply to a single invocation. Place named flags before positional paths or the goal. The management commands accept plain paths unless a different input mode is selected.

| Command | Options | Default and effect |
| --- | --- | --- |
| `agent-factory` | `-factory`, `-records`, `-revision` | Factory definition ID and entity-array JSON path default to `FACTORY_DEFINITION` and `FACTORY_RECORDS` and must be supplied by flag or environment. Integer revision defaults to 0 (latest). The remaining words form the goal. |
| `continuous-swarm` | `-field`, `-review-replay` | Repeat the string field selector to constrain edits. With no selector, the runner derives scope from the goal. Review defaults to false. Enabling it requires an active replay gate and human approval after successful replay. |
| `factory up` | `-f` | Optional Compose file path. Overrides `COMPOSE_FILE` and project discovery. |
| `factory validate` | `--rule`, `--tools` | Select one mode and provide one or more files. Default mode validates packs from named paths or local discovery. |
| `factory test` | `--receipt` | Default mode tests working-tree fixtures. Receipt mode takes one pack file, tests the committed snapshot and prints JSON. |
| `factory tools import` | `-operation`, `-server`, `-key-file`, `-key-env` | Optional strings. Select one operation ID or import eligible GET operations. A fixed HTTP(S) base overrides the document's server selection. For one authenticated operation, choose a mounted file path or a process environment variable name. |
| `factory publish` | `-pack`, `-evidence`, `-expected` | All required. Supply the committed pack path, passing JSON receipt path and expected integer revision. Use 0 for an initial proposal. |
| `factory readiness` | `--gateway` | Default mode checks the store and configured evidence tools and sources. The switch adds local model route and credential checks. |

The [harness guide](HARNESSES.md) covers goal execution and JSON output. The [release guide](RELEASE.md) shows the fixture-to-publication handoff.

## Versioned workspace settings

Read `GET /api/settings/schema` for a machine-readable JSON Schema with types, bounds, defaults and descriptions. Read `GET /api/settings` for effective workspace values, then post the complete settings object with its current revision as `Expected`. A successful edit appends a revision. A run pins the settings it reads at startup.

| JSON field | Type and default | Effect |
| --- | --- | --- |
| `max_rounds` | Positive integer, 10 | Continuous round budget. The CLI uses 5 until a settings revision exists. |
| `max_minutes` | Positive integer, 30 | Continuous wall-clock budget, up to 525600 minutes. The CLI uses 10 until a settings revision exists. |
| `max_tokens` | Positive integer, 1000000 | Continuous token allowance. Each model call reserves remaining output capacity. Reported usage is charged before another call can reuse it. Provider input and reasoning can exceed the output reservation. |
| `observe` | Boolean, false | Records an approved decision while skipping its graph write in the continuous runner. |
| `replay_gate` | Boolean, true in a new workspace | Holds learned rule proposals for review or replay under the applicable runner. |
| `worker_count` | Integer 1–256, default 2 | Concurrent continuous workers per round. |
| `max_record_failures` | Positive integer, 3 | Block a record after this many failed worker executions in one continuous run. Each execution can make multiple model attempts. |
| `max_failures` | Positive integer, 20 | Cumulative worker failures before stopping at the next round boundary. |
| `specialist_after` | Positive integer, 5 | Occurrences of the same pattern in a run before the steward registers a specialist. [Dispatch and review](RULES_REFRESH.md) |
| `stagnant_rounds` | Positive integer, 2 | Consecutive unchanged graph comparisons before stopping. |
| `judge_evaluations` | Integer 1–32, default 1 | Distinct judge evaluations per continuous proposal when using a live gateway. Each can incur model cost. |
| `judge_approvals` | Integer 1 through `judge_evaluations`, default 1 | Required number of approving evaluations. |

Stored revisions without quorum fields retain their original two-evaluation, two-approval requirement. Supply distinct judge routes for those workspaces or save a new settings revision to change the policy. Older clients editing the original settings preserve the current values for omitted continuous-runner fields.

Factory schemas, source mappings, rules, tool grants and evidence recipes have their own scoped contracts. [Definition API](API.md) · [Source mappings](MAPPING.md) · [Tool limits](HTTP_TOOLS.md)

### Independent judge model

Set `LLM_PROVIDER` for the worker and other ordinary calls. To route only judge calls to a different model, set both `LLM_JUDGE_PROVIDER` (gemini or openrouter) and `LLM_JUDGE_MODEL`. The judge uses its provider’s dedicated runtime credential, `GEMINI_API_KEY` or `OPENROUTER_API_KEY`. Run history records the role, served model and tokens for each worker and judge call.

For a Gemini worker with a GPT-4o judge, use `LLM_PROVIDER=gemini`, `LLM_JUDGE_PROVIDER=openrouter`, and `LLM_JUDGE_MODEL=openai/gpt-4o`. For the reverse, use `LLM_PROVIDER=openrouter`, `LLM_CHEAP_MODEL=openai/gpt-4o`, `LLM_JUDGE_PROVIDER=gemini`, and `LLM_JUDGE_MODEL=gemini-3.5-flash-lite`. With both credentials available, `make test-mixed-judges` executes both configurations through human approval.

## Factory memory files

Set `FACTORY_STORE_MODE=memory` for an entirely process-local API, including runs, definitions and memory files. This takes precedence over database settings. Each process starts empty. For shared durable state, leave that variable unset and set `DATABASE_URL` to PostgreSQL. Without either setting the API uses SQLite. Explicit `FACTORY_STORE_MODE=sqlite` uses the local file even if a PostgreSQL URL is present. Set `FACTORY_STORE_MODE=postgres` and `DATABASE_URL` in production to require PostgreSQL. Changing the selector chooses a different store on restart. Moving existing data is a separate operation. `factory readiness` opens the selected store, initializes its schema, and reports the backend name.

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

Gateway retry settings resolve at process startup. With `LLM_GATEWAY_CONFIG`, each model route reads its YAML retry policy and defaults. Without that file, all model routes use the environment policy. [Per-route example](ENTERPRISE_GATEWAY.md#retry-policy)

## Check deployment readiness

Run `factory readiness` with the same environment and mounted files as the deployment. It opens the workspace store and checks any configured continuous evidence recipe, pack grants (including `propose`), API source configuration and tool credentials referenced by `auth.key_file` or `auth.key_env`. A failed check exits nonzero. Both a missing and an empty tool key fail the check. The CLI reports the tool name without printing credential values. Runtime calls reread the configured file or process variable.

With tool and source configuration unset, plain readiness checks the store alone. Add `--gateway` to build the model routes using the API’s configuration precedence. With `LLM_GATEWAY_CONFIG`, the check also reads every configured authenticated gateway’s key file and loads the configured CA. Environment-only mode checks the selected providers and required keys. To validate model credentials or upstream permissions, run a representative goal against that deployment. The readiness command makes no tool or model requests.

### Verified storage deployments

| Mode | Tested topology | Operating guidance |
| --- | --- | --- |
| `memory` | One API process | Use for disposable sessions. A restart resets state and each replica has its own workspace. |
| `sqlite` | Two API processes opening one file on a local filesystem | Use for a single-host durable workspace. Protect the database and its directory with OS permissions. The bundled Go driver supplies SQLite 3.54.1 in this build. |
| `postgres` | Two API processes connecting to PostgreSQL 16.16 | Use for replicas sharing a durable workspace. Supply `DATABASE_URL` from a deployment secret. See the [Helm deployment guide](DEPLOYMENT.md). |

Both durable drivers create missing tables when opening the store. Run `factory readiness` with the deployment's settings to exercise initialization and check the effective driver. Back up durable data before upgrading. The measurements below cover a local filesystem and a single PostgreSQL host.

At revision `9217fd1`, two API processes raced to update the same definition through 20 successive expected revisions on each durable backend. Each round returned one success and one HTTP 409. Across the 40 requests per backend, median and p95 latency were 4.6 and 10.8 ms on SQLite and 4.2 and 10.7 ms on PostgreSQL. These are local measurements. A reviewed service-desk run and its revision history survived an API restart on both stores. [SQLite field results](https://github.com/idristarwala/agent-factory/issues/144) · [PostgreSQL field results](https://github.com/idristarwala/agent-factory/issues/143)

SQLite allows one writer at a time. Each Factory SQLite store uses one connection and applies the configured busy timeout (five seconds by default) to every new connection. If another writer holds the lock beyond the timeout, the store can return a busy error. Retry after the writer releases the lock. A revision conflict means a newer revision already exists. Read the current history and reconcile the edit before resubmitting.

On PostgreSQL 16.16, an API at `9575419` kept serving while an API built from the source merged at `14a1c9b` added the work-unit payload table to the same database. The new process read the old process's first memory-file revision and wrote the second. The old process read that revision and wrote a third, which the new process read. The test covered this additive migration. Assess each schema change before a mixed-version rollout.

A rolled-back configuration insert left zero rows. On an analyzed table with 8,000 configuration revisions, PostgreSQL used the composite primary-key index to fetch one definition's four revisions in order. The local execution took 0.109 ms. CI also reopens both durable stores and checks the exact bytes and SHA-256 of two pinned evidence revisions and the complete ordered decision audit. [Verification test](../internal/factory/durable_evidence_test.go)

For workspace goals that pin a pack release, observation also depends on the factory definition. A new pack pin starts in observe mode. A compatible `pack_promotion` reference permits approved edits when workspace observation is off. The effective mode is stored with each run. [Observation and promotion](RELEASE.md#promote-a-reviewed-observation)
