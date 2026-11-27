# Run a factory

Import an explicit definition, then run it against a source file. From the project root, with a model gateway configured, try the service-desk example.

```sh
go run ./cmd/factory import factory service-desk 0 examples/service-desk/factory.json
go run ./cmd/goal -factory service-desk -revision 1 -records examples/service-desk/records.json 'Assign requests using the routing guide'
```

The importer takes the current revision as its fourth argument after the command name and kind/ID. Zero creates a new definition. The runner prints the selected revision and emits JSON decisions to standard output. Source data stays in its input file.

The HTTP API uses the same factory definitions and evaluator. Start the API, open Goal builder, and supply the definition ID, goal, and source records. Pending changes can be reviewed before application to the run's record snapshot. Use Settings to enable observe mode for subsequent runs.

Set `FACTORY_DB_PATH` or `DATABASE_URL` and `FACTORY_ID` consistently across the CLI and API. Set `FACTORY_MODEL_PRICES` to a JSON object keyed by model name with numeric `input_per_million` and `output_per_million` USD rates. Run details retain token counts and the estimate calculated for each successful model call. Unpriced calls are identified separately.

For API-managed runs, `GET /api/usage?run=<id>` reports input, output and total tokens, model attempts, failed attempts, known estimated cost and unpriced responses. It also reports tool invocation counts by name. Reading pinned goal artifacts is one logical tool invocation per run and collecting mapped images is one per evaluated record that needs images; failed invocations count. A shared worker/judge input is counted once. Dashboard run totals exclude chat and design traffic; the unfiltered API retains these session tokens separately as `non_run_tokens` and includes them in the legacy `tokens_used` total. Earlier run snapshots have no recorded tool counts.
