# Customer UI

Customers never see agents, rules, or units of work. They see:

1. **Images** of the items
2. **Decisions** — what changed (before → after)
3. **Feedback** — Looks good / Needs change + optional note

## Open it

```bash
open ui/index.html
# or
python3 -m http.server 8080 --directory ui
```

## What happens on feedback

- "Looks good" marks the decision approved.
- "Needs change" + note is stored as feedback.
- Behind the scenes that feedback becomes a versioned rule for the next pass. The customer never writes a rule.

## Next

Wire the static page to a small API that:
- Serves real graph decisions
- Accepts feedback
- Calls `ruleEngine.Propose` with `Source: "feedback"`
