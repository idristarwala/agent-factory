# Field Test Fixes (#10)

## Truncated JSON

- Default `MaxTokens` raised to 16000 so reasoning tokens and full responses do not hit the limit.
- Validation failure logs include a raw preview and length.

## Failed worker counted as approved

- `GuardDecision` forces `Approved: false` when the worker returns an error.
- The judge is not asked to approve a missing or failed proposal.

These close the two critical findings from the 288-item re-run.
