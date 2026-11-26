# Customer UI

Customers never see agents, rules, or units of work. They see:

1. **Images** of the items
2. **Decisions** — what changed (before → after)
3. **Feedback** — Looks good / Needs change + optional note

## Run the React app

```bash
cd ui-app
npm install
npm run dev
```

Open the URL Vite prints (usually http://localhost:5173).

## Build

```bash
cd ui-app
npm run build
```

## What happens on feedback

- "Looks good" marks the decision approved.
- "Needs change" + note is stored as feedback.
- Behind the scenes that feedback becomes a versioned rule for the next pass. The customer never writes a rule.
