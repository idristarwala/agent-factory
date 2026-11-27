# First run from an external harness

From a checkout with Go 1.26, build the CLI once. Both examples below use the standard library, the same two binaries and the service-desk fixture. Set `GEMINI_API_KEY` in the launching environment and run from the repository root. A release installation can supply the same binaries on `PATH`. [Release guide](RELEASE.md)

```sh
bin=$(mktemp -d)
sh scripts/build-binaries.sh "$bin"
export PATH="$bin:$PATH"
export LLM_PROVIDER=gemini
```

Each example creates an isolated SQLite workspace. Set `FACTORY_STORE_MODE=sqlite` with a shared file for local harnesses, or `FACTORY_STORE_MODE=postgres` with `DATABASE_URL` for production. Two subprocesses need the same durable store. The import creates revision 1 with an expected prior revision of 0. The goal command selects that revision explicitly.

## Python

```python
import json
import os
from pathlib import Path
import subprocess
import tempfile

with tempfile.TemporaryDirectory() as workspace:
    env = dict(os.environ, FACTORY_DB_PATH=str(Path(workspace) / "factory.sqlite"))
    for key in ("DATABASE_URL", "FACTORY_STORE_MODE"):
        env.pop(key, None)
    subprocess.run(["factory", "import", "factory", "desk", "0",
                    "examples/service-desk/factory.json"], env=env, check=True)
    result = subprocess.run(["agent-factory", "-factory", "desk", "-revision", "1",
                             "-records", "examples/service-desk/records.json",
                             "Assign the support team from the routing guide"],
                            env=env, check=True, capture_output=True, text=True)
    for line in result.stdout.splitlines():
        print(json.dumps(json.loads(line)))
```

## Node.js

```js
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const workspace = mkdtempSync(join(tmpdir(), "factory-"));
const env = { ...process.env, FACTORY_DB_PATH: join(workspace, "factory.sqlite") };
delete env.DATABASE_URL;
delete env.FACTORY_STORE_MODE;
try {
  execFileSync("factory", ["import", "factory", "desk", "0",
    "examples/service-desk/factory.json"], { env });
  const output = execFileSync("agent-factory", ["-factory", "desk", "-revision", "1",
    "-records", "examples/service-desk/records.json",
    "Assign the support team from the routing guide"], { env, encoding: "utf8" });
  for (const line of output.trimEnd().split("\n")) console.log(JSON.stringify(JSON.parse(line)));
} finally {
  rmSync(workspace, { recursive: true, force: true });
}
```

## Output and review

Read `entity_id`, `factory_revision`, `proposal` and `decision.Approved` from each JSON line. Diagnostics go to stderr. A successful process can return a rejected proposal. Inspect `decision.Reason` before using a proposed change. Check the process exit status as later records can fail after earlier results have been emitted.

Replace the example files with your own record type, editable fields and records. The CLI evaluates the input file and writes decisions to stdout. For a persisted run with a human approval step, use the HTTP goals and decisions endpoints. [Example definition](../examples/service-desk/factory.json) · [Example records](../examples/service-desk/records.json) · [HTTP API](API.md)
