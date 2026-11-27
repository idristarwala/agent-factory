# Original catalog field test

Issue #10 records the original 288-item field test, the grouping accuracy gap and the reasoning-token truncation observed in that evaluation. Keep results from synthetic fixtures separate from measurements on that original dataset. [Field notes and acceptance criteria](https://github.com/idristarwala/agent-factory/issues/10)

The current worker’s default output limit is 16000 tokens. It builds the response schema from the unit’s editable fields, validates returned values and includes the source evidence in the proposal. The field evaluation still requires access to the original records and labels. [Worker schema](SCHEMA.md) · [Project status](STATUS.md)
