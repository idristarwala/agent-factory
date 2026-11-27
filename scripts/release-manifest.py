"""Generate the release manifest from the tagged checkout and built assets."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]


def schema_version(*paths):
    # Hash a canonical inventory so file boundaries and names are unambiguous.
    inventory = {
        path: hashlib.sha256((ROOT / path).read_bytes()).hexdigest() for path in paths
    }
    encoded = json.dumps(inventory, sort_keys=True, separators=(",", ":")).encode()
    return "sha256:" + hashlib.sha256(encoded).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("dist", type=Path)
    dist = parser.parse_args().dist
    images = {}
    repository = os.environ["IMAGE_REPOSITORY"]
    for role in ("engine", "worker", "connector"):
        key = role.upper() + "_DIGEST"
        digest = os.environ.get(key, "")
        if not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
            parser.error(f"{key} must be a SHA-256 registry digest")
        images[role] = f"{repository}@{digest}"
    output = dist / "release-manifest.json"
    binaries = sorted(p.name for p in dist.iterdir() if p.is_file() and p.name not in {"LICENSE", output.name})
    if not binaries or not (dist / "LICENSE").is_file():
        parser.error("release assets must include binaries and LICENSE")
    manifest = {
        "version": os.environ["GITHUB_REF_NAME"],
        "image": images["engine"],
        "images": images,
        "binaries": binaries,
        "license": "LICENSE",
        "pack_spec_version": schema_version("schemas/pack.schema.json", "schemas/rule.schema.json"),
        "database_schema_version": schema_version("internal/store/schema.go"),
    }
    output.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
