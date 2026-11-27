# Active Rules Reach the Next Round

`ActiveRulesForRound` returns the current active rules from the engine.

The continuous runner should call this before `CreateWorker` each round so newly learned rules are visible to the next set of workers.

This closes the last gap listed for #13.
