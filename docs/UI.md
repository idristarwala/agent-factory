# Operator workspace

The workspace provides goals, chat, an agent editor and a rule manager. The Figma file also contains designs for the remaining operator screens and configurable experiences. [Design file](https://www.figma.com/design/jPNDukgzOzcM6n1e6UpV9R/Agent-Factory?node-id=28-2)

Run `npm ci` and `npm run dev` in `ui-app`. Configure `VITE_API_URL` for the API and `CORS_ORIGIN` on the server for the browser origin.

## Definition history

Choose **Agents** to edit a definition or retire it. The build version identifies the software release. The server allocates the numeric definition revision. Changing a role, model tier, instructions, tools or retirement state produces a new snapshot. Relaunching the same definition reuses its revision.

Choose **Rules** to save a proposal, inspect its snapshot or activate a revision. Activating an earlier rule appends a rollback revision. Existing history remains available. The active rule can be older than the latest proposal.

The API supports GET and POST on `/api/agents` and `/api/rules`. Request all rule revisions with `?history=true`. Filter rule history by `id` or agent history by `name`. Include the expected head revision in each write, using zero on creation. A concurrent edit returns HTTP 409. Reload before applying the intended change.

The API and runners use PostgreSQL when `DATABASE_URL` is configured. Locally they share SQLite at `FACTORY_DB_PATH`, defaulting to `agent-factory/factory.sqlite` under the user configuration directory. Set `FACTORY_ID` consistently for the API and runner to isolate definitions by factory. API edits record a human actor class. The shared bearer-token authentication does not identify individual people. Store credential references in configuration and keep secret values in the secret store.
