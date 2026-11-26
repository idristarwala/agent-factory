# Clarification

If the goal is unclear or no data source is provided, the Steward asks.

## When it asks

| Missing | Question |
|---------|----------|
| Goal too short or empty | “What would you like to achieve?” |
| No source and no sample | “Where does your data live? BigQuery table, API, or file?” |
| Goal has no domain hint | “Which area — catalog, pricing, inventory, or something else?” (optional) |

## Flow

1. User types in the chat.
2. API calls `NeedsClarification`.
3. If required questions remain, the reply lists them and sets `needs_input: true`.
4. User answers (goal text, or points at a source).
5. Once `CanProceed` is true, the Steward maps the source and starts the factory.

Business users never configure agents or schemas. They answer plain questions.
