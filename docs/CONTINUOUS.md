# Running the Factory Continuously

The factory is designed to run until the goal is achieved.

## Local continuous run

```bash
go run ./cmd/continuous-swarm "Correct the title and mark the record fixed"
```

The ContinuousRunner:
1. Asks the Steward to interpret remaining problems into units of work
2. Enqueues them
3. Workers claim, propose, and Judge evaluates
4. Approved changes are applied to the graph
5. Learning loop reviews decisions and can propose new rules
6. Repeats until no more units of work or MaxRounds is reached

## Run history

The command executes against its built-in fixture and publishes the terminal run to the workspace store. Inspect the proposals, judge verdicts, write outcomes and usage through the API. [Observability](OBSERVABILITY.md)

## Goal-driven

State a clear goal. The Steward turns it into units of work. The loop continues until the graph reports no remaining problems or the round limit is hit.

## Choose editable fields

Use a repeated `-field` flag before the goal to give the continuous worker an explicit write scope. For example, `go run ./cmd/continuous-swarm -field title 'Improve the title using the record as evidence'` authorizes one field on the bundled fixture. Other properties remain available to the judge and worker as evidence. A compact goal such as `Edit only title` names a single literal field. A goal with no explicit scope can use the sole editable property of a single-field record. A multi-field record with ambiguous scope stops before a model call.

The worker rejects an entire reply containing an out-of-scope property. A successful continuous write can also update the runner-managed completion status. Workspace factories declare their editable fields and types in the pinned factory definition. [Worker output schema](SCHEMA.md)

After each round the smart steward checks the updated graph. Completing the last record on the final allowed round produces a completed run. If work remains at the limit, the CLI saves the failed run and exits with a nonzero status.

Set `max_record_failures` in the versioned workspace settings to bound retries on an individual record. The count advances once when a worker execution exhausts its attempts. Other records can continue. When every remaining unit is blocked, the run reports the per-record limit and stops. A new run starts fresh failure counts. [Configuration](CONFIGURATION.md)
