# Inspect a run

Read `GET /api/goals/{id}` for the run snapshot and `GET /api/usage?run={id}` for model and tool usage. The operator dashboard reads this history from the workspace store. Use the same database and factory scope in the API and continuous CLI to inspect a CLI run after it exits. [API reference](API.md)

Continuous runs publish their journal at termination, including failed runs. Each worker proposal records its mutation, entity and work-unit trace ID. The judge records its decision and reason. The runner records whether it applied the change, observed it, or encountered a write failure. Match the journal’s `trace_id` to `calls[].unit_id` to inspect individual model calls and quorum evaluations. Earlier run revisions can lack these fields.

Workspace goal runs expose proposals, review state, evidence and before/after values through the decisions endpoint. To estimate cost, configure model-specific input and output prices before the run. [Run and usage guide](RUNNING.md)
