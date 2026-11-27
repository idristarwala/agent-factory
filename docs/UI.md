# Operator workspace

Use the workspace to define a factory in chat, start a goal, track run usage, inspect records and decisions, and approve or reject a change with feedback. The rule manager lets a reviewer activate feedback for a later run. Agents and Settings provide versioned definition and budget controls. The Figma file covers the operator screens and configurable experiences. [Design file](https://www.figma.com/design/jPNDukgzOzcM6n1e6UpV9R/Agent-Factory?node-id=28-2)

Run `npm ci` and `npm run dev` in `ui-app`. Configure `VITE_API_URL` for the API and `CORS_ORIGIN` on the server for the browser origin.

When the API requires a bearer token, enter the workspace API token on the connection screen. The browser keeps it in memory for the current page. Reloading the page or choosing **Disconnect** clears it.

## Definition history

Choose **Agents** to edit a definition or retire it. The build version identifies the software release. The server allocates the numeric definition revision. Changing a role, model tier, instructions, tools or retirement state produces a new snapshot. Relaunching the same definition reuses its revision.

Choose **Rules** to save a proposal, inspect its snapshot or activate a revision. Activating an earlier rule appends a rollback revision. Existing history remains available. The active rule can be older than the latest proposal.

The API supports GET and POST on `/api/agents` and `/api/rules`. Request all rule revisions with `?history=true`. Filter rule history by `id` or agent history by `name`. Include the expected head revision in each write, using zero on creation. A concurrent edit returns HTTP 409. Reload before applying the intended change.

The API and runners use PostgreSQL when `DATABASE_URL` is configured. Locally they share SQLite at `FACTORY_DB_PATH`, defaulting to `agent-factory/factory.sqlite` under the user configuration directory. Set `FACTORY_ID` consistently for the API and runner to isolate definitions by factory. API edits record a human actor class. The shared bearer-token authentication does not identify individual people. Store credential references in configuration and keep secret values in the secret store.

## Versioned configuration

**Settings** controls the continuous runner's wall clock, token and round limits, replay gate and observe mode. The runner reads a snapshot at startup. The connected gateway is configured by deployment environment. GET `/api/settings?history=true` retrieves the previous settings snapshots.

Use `/api/definitions/{kind}/{id}` to store and read factory, pack, tool and experience configuration snapshots. POST a JSON object with `Expected` and `Data`. GET retrieves the ordered revisions. A declaration can be stored independently of runtime activation. File-based integrations need to resolve the chosen revision when loading a definition.
