# Metrics

Query `GET /api/usage?run={id}` for a run’s token totals, model attempts, failed attempts, tool counts and estimated cost. Model responses without configured prices are counted separately. The unfiltered endpoint includes workspace totals. [API reference](API.md) · [Pricing configuration](CONFIGURATION.md)

Use the run and decision records to inspect approvals, rejections and their reasons. Continuous run audit entries include a timestamp and work-unit correlation ID. [Observability guide](OBSERVABILITY.md)
