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
