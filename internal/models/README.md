# Model Connection

All agents call LLMs exclusively through the `ModelGateway`.

## Tier Enforcement
- **Workers**: only `TierCheap` (Gemini Flash, DeepSeek Flash, Groq-class, etc.).
- **Steward and Judge**: may use `TierFrontier` or `TierCouncil`.
- The gateway rejects any attempt by a worker to request a higher tier.

## How it connects
1. Concrete gateway implementation (added later) talks to one or more providers via their HTTP APIs (OpenAI-compatible endpoints, Gemini API, Anthropic, etc.).
2. Configuration (API keys, base URLs, model name mappings per tier) is loaded from environment or a config file at startup. Never hard-coded.
3. For local development and tests, a mock gateway returns canned responses so the rest of the fabric can be exercised without real API keys or cost.
4. Production gateway can route by tier:
   - cheap → cheapest available provider/model
   - frontier → single high-quality model
   - council → parallel calls + aggregation (only for Judge)

## Next steps
- In-memory mock gateway for tests.
- One real provider implementation (e.g. OpenAI-compatible) behind a feature flag.
- Config struct for model mappings and credentials.
