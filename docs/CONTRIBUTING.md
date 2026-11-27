# Contributing

This repository is proprietary. See LICENSE.

If you have been granted permission to contribute:

1. Open an issue describing the change.
2. Keep changes focused.
3. Run `make ci` before submitting.
4. Do not commit secrets.

Use Go 1.26 and Node.js 24 for the checks. To run them automatically
before each push from this clone, run `git config core.hooksPath .githooks`.

When changing a CLI, API or setting, update its guide and the configuration reference. `make docs-check` checks local Markdown links, Go command paths and documentation for literal Go environment lookups. Verify any changed workflow through its public entry point.
