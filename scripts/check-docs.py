"""Check repository-local guide links, runnable Go paths and deployment variable coverage."""
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import unquote, urlsplit


def tracked(pattern):
    paths = subprocess.check_output(["git", "ls-files", "-z", pattern]).decode()
    return [Path(path) for path in paths.split("\0") if path]


def headings(path):
    titles = re.findall(r"^#{1,6} (.+)$", path.read_text(), re.MULTILINE)
    return {re.sub(r"[^\w -]", "", title.lower()).replace(" ", "-") for title in titles}


def check_guides():
    for guide in tracked("*.md") + [Path("llms.txt")]:
        text = guide.read_text()
        for href in re.findall(r"\]\(([^)\s]+)\)", text):
            link = urlsplit(href)
            if link.scheme or link.netloc:
                continue
            target = guide.parent / unquote(link.path) if link.path else guide
            if not target.exists():
                yield f"{guide}: missing link target {href}"
            elif link.fragment and target.suffix == ".md" and unquote(link.fragment) not in headings(target):
                yield f"{guide}: missing heading for {href}"
        for cmd in re.findall(r"go run \./(cmd/[\w-]+)", text):
            if not (Path(cmd) / "main.go").is_file():
                yield f"{guide}: missing Go command {cmd}"


def check_environment():
    reference = Path("docs/CONFIGURATION.md").read_text()
    for path in tracked("*.go"):
        if path.name.endswith("_test.go"):
            continue
        for key in re.findall(r'os\.(?:Getenv|LookupEnv)\("([A-Z0-9_]+)"\)', path.read_text()):
            if f"`{key}`" not in reference:
                yield f"{path}: document {key} in docs/CONFIGURATION.md"


def check_cli_and_frontend_settings():
    reference = Path("docs/CONFIGURATION.md").read_text()
    for directory in ("goal", "factory", "continuous-swarm"):
        for path in tracked(f"cmd/{directory}/*.go"):
            if path.name.endswith("_test.go"):
                continue
            code = path.read_text()
            names = re.findall(r'\bflags?\.(?:String|Int|Bool|Func|Duration|Float64)\("([a-z][a-z0-9-]*)"', code)
            options = {"-" + name for name in names}
            options.update(re.findall(r'(?:==|!=)\s*"(--[a-z][a-z0-9-]*)"', code))
            for option in options:
                if f"`{option}`" not in reference:
                    yield f"{path}: document {option} in docs/CONFIGURATION.md"

    for pattern in ("ui-app/src/*", "extensions/*"):
        for path in tracked(pattern):
            if path.suffix not in {".js", ".jsx", ".mjs", ".ts", ".tsx"} or ".test." in path.name:
                continue
            for key in re.findall(r'(?:import\.meta|process)\.env\.([A-Z][A-Z0-9_]*)', path.read_text()):
                if f"`{key}`" not in reference:
                    yield f"{path}: document {key} in docs/CONFIGURATION.md"


if __name__ == "__main__":
    errors = sorted(set(check_guides()) | set(check_environment()) | set(check_cli_and_frontend_settings()))
    for error in errors:
        print(error, file=sys.stderr)
    sys.exit(bool(errors))
