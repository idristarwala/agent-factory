"""Exercise the tag workflow's manifest generator offline."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
from tempfile import TemporaryDirectory


root = Path(__file__).resolve().parents[1]
with TemporaryDirectory() as temp:
    checkout = Path(temp)
    files = ["scripts/release-manifest.py", "schemas/pack.schema.json", "schemas/rule.schema.json", "internal/store/schema.go"]
    for name in files:
        target = checkout / name
        target.parent.mkdir(exist_ok=True, parents=True)
        shutil.copyfile(root / name, target)
    dist = checkout / "dist"
    dist.mkdir()
    for name in ["factory", "LICENSE"]:
        (dist / name).touch()
    digest = "sha256:" + "a" * 64
    env = dict(os.environ, DIGEST=digest, GITHUB_REF_NAME="v0.0.0-test", GITHUB_REPOSITORY="example/factory")
    command = [sys.executable, checkout / files[0], dist]

    def generate():
        subprocess.run(command, env=env, check=True)
        return json.loads((dist / "release-manifest.json").read_text())

    original = generate()
    assert generate() == original
    assert original["binaries"] == ["factory"] and original["license"] == "LICENSE"
    assert original["image"] == "ghcr.io/example/factory@" + digest
    assert original["version"] == "v0.0.0-test"
    for name in files[1:]:
        path = checkout / name
        path.write_bytes(path.read_bytes() + b"\n")
        modified = generate()
        key = "pack_spec_version" if name.startswith("schemas/") else "database_schema_version"
        assert modified[key] != original[key]
        assert {**modified, key: original[key]} == original
        shutil.copyfile(root / name, path)
