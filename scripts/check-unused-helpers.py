"""Report newly unreachable exported internal Go helpers against a Git revision."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile


def unused(root):
    output = subprocess.check_output([
        "go", "run", "golang.org/x/tools/cmd/deadcode@v0.44.0",
        "-json", "-filter", r"^github\.com/idristarwala/agent-factory/internal/", "./...",
    ], cwd=root)
    return {
        f"{package['Path']}.{fn['Name']}"
        for package in json.loads(output) or []
        for fn in package["Funcs"]
        if fn["Name"].rsplit(".", 1)[-1][:1].isupper()
    }


def main(base):
    root = Path(subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip())
    with tempfile.TemporaryDirectory() as temp:
        checkout = Path(temp) / "base"
        subprocess.run(["git", "worktree", "add", "--detach", str(checkout), base], check=True)
        try:
            before = unused(checkout)
        finally:
            subprocess.run(["git", "worktree", "remove", str(checkout)], check=True)
    added = sorted(unused(root) - before)
    for name in added:
        print(f"No production entry point reaches {name}")
    return bool(added)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
