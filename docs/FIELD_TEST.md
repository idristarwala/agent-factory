# Field Test Notes (#10)

## Reasoning token budget

Reasoning models count thinking tokens against MaxTokens. The previous hard-coded 800 was too low and caused truncated JSON.

- `WorkerConfig.MaxTokens` defaults to 4000.
- Log the raw reply preview and length on validation failure so finish reasons are visible.

## Output format enforcement

Customer policies can contain their own JSON shapes. The worker should enforce its allowed fields via provider response_format / responseSchema where available, with ParseAndValidate as a backstop.

These changes address the two findings from the 288-item re-run.
