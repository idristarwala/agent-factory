# Rules in continuous runs

At the start of each round, the runner reads active rule revisions from the rule engine. The steward builds workers with that list and the judge evaluates their proposals against the same list. A rule activated during the round takes effect in the following round. A worker records the rule revisions it used on its proposed mutation.

## Replay a reviewed workspace run

Select a completed run in the same workspace where a human approved every record's proposed patch. Use the exact run ID and terminal revision. Held-out record IDs must be disjoint from the continuous graph. Records with external image URLs are ineligible because their contents can change.

New workspaces require review or replay by default. Existing workspaces retain their saved policy. To use replay, enable the versioned workspace setting `replay_gate` and configure a model gateway. Set `FACTORY_REPLAY_RUN=RUN_ID@REVISION` when starting `continuous-swarm`. With the gate enabled and the reference unset, new rules await review. With a reference, the runner creates an inactive revision and evaluates each held-out record twice. The baseline uses currently active rules. The candidate adds the proposed rule. Both use the historical goal, before-values, factory plan, worker and judge definitions, pinned memory, and goal artifacts. The currently configured gateway supplies the models. Replay evaluates proposals in scratch graphs.

A case passes when the judge approves a worker patch matching the human-approved patch. Automatic promotion requires at least one new pass and zero regressions. Pass `-review-replay` to keep a successful candidate pending for a human. Both outcomes append a revision with the baseline rule versions and per-record results. A provider or budget failure leaves the first revision pending and surfaces the error. An identical proposal can retry an interrupted evaluation. A completed replay is retained when the same pattern recurs.

Replay calls count toward continuous usage. The runner checks remaining tokens and wall time before each replay call. One in-flight response can consume the remaining budget. Inspect the run's call records for unit IDs beginning with `replay/` and the rule's revision history for the comparison.

## Learned specialists

The continuous runner counts each recurring worker failure or judge rejection pattern across rounds. Once the pattern reaches the workspace's `specialist_after` threshold, the steward registers a versioned worker with the generic worker's current configuration and tool list. Configure the threshold under Settings or through the settings API.

On the next round, a record that encountered that pattern can use the specialist if the associated rule is active. The worker receives the current rule revision as its focus. Unaffected records use the generic worker. A pending rule cannot trigger dispatch. Retiring the specialist restores generic routing. Later steward observations preserve existing human edits. The proposed mutation records the actual agent name, agent revision and active rule revisions.

When reviewing a proposal, read `Updates[].Props` as a partial update. Omitted fields retain their values. A separate `complete_on_apply` marker tells the runtime to mark the record fixed when it applies an approved mutation. The audit preserves the worker's proposed fields.

## Customer-supplied policies

`rules.IngestPolicies` imports approved customer policies as active revisions. Pass policies from `rules.ParseSimpleTree` through the same function to activate their conditions and actions. Workers and judges see them through `ToAgentsRules`. Supply a stable `Policy.ID` when revising an existing policy. An empty ID creates a separate rule. Importing is an explicit activation decision. Use an inactive proposal for changes awaiting review.
