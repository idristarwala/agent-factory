# Agent Factory

**Domain-agnostic multi-agent execution fabric.**

Give it a goal in plain language. A Steward decomposes it into units of work, specialized agents propose changes to a central graph, a Judge approves or rejects against versioned rules, and the system continues until the outcome is achieved — improving itself from every rejection.

## Quick Start

```bash
./scripts/setup.sh
export GEMINI_API_KEY=your-key
./scripts/run-local.sh "Fix product types and weights"
```

Or with Docker:

```bash
./scripts/docker-up.sh
```

## Documentation

Full documentation lives in [`docs/`](docs/README.md):

- [Getting Started](docs/RUNNING.md)
- [Architecture & Concepts](docs/ARCHITECTURE.md)
- [Domain Packs & Data Mapping](docs/DOMAIN_PACKS.md)
- [Operations & Observability](docs/OPERATIONS.md)
- [API & UI](docs/API.md)
- [Diagrams](docs/ER_DIAGRAMS.md)

## What makes it different

- **Agent-first** — no external orchestration frameworks
- **Governed** — nothing reaches the graph without Judge approval
- **Self-improving** — rejections automatically become new rules and specialist agents
- **Domain-agnostic** — catalog is one pack; the same system works for pricing, inventory, compliance, or any Entity-based problem
- **Concurrent** — claimable work queue with parallel workers
- **Observable** — traces, metrics, and immutable audit log

## License

Proprietary. All rights reserved. See [LICENSE](LICENSE).

### Goal chat attachments

Attach up to four files, including PDFs, Word DOCX documents, Markdown or UTF-8 text files (32 KiB each) and PNG, JPEG or WebP screenshots (512 KiB each) in Factory chat. PDFs and DOCX files may be up to 512 KiB with up to 32 KiB of extracted text. PDF evidence cites physical page numbers and DOCX evidence cites body paragraphs. Image-only PDFs require OCR or a separate screenshot. The API container includes Poppler. For a standalone API binary, install `pdftotext` to enable PDF uploads. The files stay selected across messages in that chat view. Upload a changed file with the same name to create a new revision, or remove it from the next message. Creating a factory from chat pins the selected revisions in its definition.

In Goal builder, select **Include pinned goal attachments in worker and judge evidence** to use those revisions in a run. The run records the file names, revisions and content digests. PostgreSQL persists the files across restarts. In-memory mode retains them for the current process.
