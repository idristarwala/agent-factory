# Running the Factory Continuously

The factory is designed to run until the goal is achieved.

## Local continuous run

```bash
go run ./cmd/continuous-swarm "Fix all product titles and group variants"
```

The ContinuousRunner:
1. Asks the Steward to interpret remaining problems into units of work
2. Enqueues them
3. Workers claim, propose, and Judge evaluates
4. Approved changes are applied to the graph
5. Learning loop reviews decisions and can propose new rules
6. Repeats until no more units of work or MaxRounds is reached

## Making an agent session continuous

In this chat you can keep saying:
- "Continue until all open issues are closed"
- "Keep iterating on the remaining gaps in issue #9"
- "Run the continuous swarm on the hard catalog and fix what it finds"

The system will keep working across turns. For fully unattended long runs, use the ContinuousRunner binary or docker-compose with a long-running process.

## Goal-driven

State a clear goal. The Steward turns it into units of work. The loop continues until the graph reports no remaining problems or the round limit is hit.
