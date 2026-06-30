# Blackbox-AgentDiff

**Flight Recorder for AI Agents** — Open-source observability platform with execution diffing as a first-class primitive.

Compare two agent executions side-by-side like `git diff` but for reasoning chains. Detect regressions, debug failures, and run CI gates on agent behavior.

## Quickstart

```bash
# Start all services (API + Diff Service + Web UI)
docker compose -f deploy/docker-compose.yml up -d

# Open the UI
open http://localhost:3000
```

## Features (Phase 1 MVP)

- **OTLP Ingest** — Accept traces from any OpenTelemetry-compatible SDK (Langfuse, OpenInference, raw OTel, etc.)
- **Trace Explorer** — Filter, sort, paginate, bulk-delete, and export traces
- **Trace Detail** — Span tree with type icons, status dots, duration bars; detail tabs for input/output/attributes
- **Diff Engine** — Structural diff (span tree alignment via tree-aware LCS), attribute diff, similarity score
- **Diff UI** — Side-by-side diff tree with ADDED/REMOVED/CHANGED/UNCHANGED styling, span diff panel, metric delta table
- **Baselines** — Save any trace as a baseline; compare future runs against it
- **Python SDK** — `blackbox.trace()`, `.generation()`, `.tool()`, `.retrieval()`, auto-instrumentation for OpenAI
- **TypeScript SDK** — Isomorphic (Node + Browser), same API, OTLP export helper
- **Docker Compose** — Single command local dev with SQLite + DuckDB storage

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Client SDK / OTel                       │
│  (Python · TypeScript · any OTLP-compatible framework)      │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS / OTLP
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Go API (port 4000)                        │
│  • /otel/v1/traces  (ingest)                                 │
│  • /api/v1/*        (REST: traces, diffs, projects, keys)   │
│  • SQLite (metadata) + DuckDB (spans)                       │
└────────────┬────────────────────────────┬────────────────────┘
             │                            │
             ▼                            ▼
┌─────────────────────┐         ┌─────────────────────┐
│  Diff Service       │         │  Next.js Web UI     │
│  (Bun/Node, :5001)  │         │  (Next.js 14, :3000)│
│  POST /internal/diff│         │  imports diff-engine│
└─────────────────────┘         └─────────────────────┘
                    ▲                    ▲
                    │ imports            │ imports
                    ▼                    ▼
         ┌────────────────────────────────────────┐
         │     packages/diff-engine (TS)          │
         │  Canonical diff algorithm (Appendix B) │
         └────────────────────────────────────────┘
```

**Why TypeScript for diff?** The algorithm is complex (tree-aware LCS, attribute diff, similarity scoring). Keeping it canonical in TypeScript avoids a Go port drift. The Go API calls the diff-service for server-side diffs; the Next.js frontend imports the package directly for instant client-side diffs.

## Repository Structure

```
blackbox-agentdiff/
├── apps/
│   ├── api/            # Go: ingest + REST API
│   ├── diff-service/   # TS: thin HTTP wrapper around diff-engine
│   └── web/            # Next.js 14 + Tailwind frontend
├── packages/
│   ├── diff-engine/    # TS: canonical structural diff algorithm
│   ├── sdk-python/     # Python SDK
│   └── sdk-typescript/ # TypeScript SDK
├── deploy/
│   └── docker-compose.yml
├── docs/               # MDX docs (quickstart, OTLP integration, self-hosting)
├── .github/workflows/  # CI workflows
├── AGENTS.md           # Dev commands for tooling
├── CONTRIBUTING.md
├── LICENSE             # Apache 2.0
└── README.md
```

## Documentation

- [Quickstart](docs/quickstart.mdx) — Get running in 5 minutes
- [OTLP Integration](docs/otel-integration.mdx) — Send traces from Langfuse, OpenInference, Arize, Braintrust, or raw OTel
- [Self-Hosting](docs/self-hosting.mdx) — Production deployment guide

## SDK Quickstart

### Python

```bash
pip install blackbox-agentdiff
```

```python
import blackbox

blackbox.init()  # reads BLACKBOX_API_KEY, BLACKBOX_PROJECT_ID, BLACKBOX_BASE_URL

with blackbox.trace("support-agent", input={"user_message": "Help me reset my password"}) as trace:
    with trace.generation("draft-reply", model="gpt-4o") as gen:
        response = openai.chat.completions.create(...)
        gen.record(input=messages, output=response.choices[0].message.content,
                   input_tokens=response.usage.prompt_tokens,
                   output_tokens=response.usage.completion_tokens)

    with trace.tool("search_docs", input={"query": "password reset"}) as tool:
        result = search_knowledge_base("password reset")
        tool.record(output=result)

    trace.set_output({"reply": "To reset your password, go to..."})
```

**Auto-instrument OpenAI:**
```python
from blackbox.integrations.openai import instrument_openai
instrument_openai()  # patches openai.chat.completions.create automatically
```

### TypeScript

```bash
npm install blackbox-agentdiff
```

```typescript
import { Blackbox } from 'blackbox-agentdiff';

const bb = new Blackbox(); // reads BLACKBOX_API_KEY, BLACKBOX_PROJECT_ID

const trace = bb.trace('support-agent', { input: { userMessage: 'Help me reset my password' } });

const gen = trace.generation('draft-reply', { model: 'claude-3-5-sonnet' });
const response = await anthropic.messages.create({ ... });
gen.record({ output: response.content[0].text, outputTokens: response.usage.output_tokens });

const tool = trace.tool('search_docs', { input: { query: 'password reset' } });
const result = await searchDocs('password reset');
tool.record({ output: result });

trace.setOutput({ reply: 'To reset your password, go to...' });
await trace.end();
```

**OTLP Export (for existing OTel-instrumented apps):**
```typescript
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-proto';

new OTLPTraceExporter({
  url: process.env.BLACKBOX_BASE_URL + '/otel/v1/traces',
  headers: {
    Authorization: `Bearer ${process.env.BLACKBOX_API_KEY}`,
    'X-Blackbox-Project-ID': process.env.BLACKBOX_PROJECT_ID,
  },
});
```

## API Reference

All endpoints under `/api/v1`, authenticated via `Authorization: Bearer <api_key>`.

| Endpoint | Description |
|----------|-------------|
| `GET /traces` | List traces (filter, paginate, sort) |
| `GET /traces/:id` | Get trace with spans |
| `GET /traces/:id/spans` | Get spans only |
| `DELETE /traces/:id` | Delete trace |
| `DELETE /traces` | Bulk delete (body: `{ids: [...]}`) |
| `POST /traces/search` | Structural search |
| `POST /diffs` | Compute diff (body: `{trace_a_id, trace_b_id, baseline_label?}`) |
| `GET /diffs/:id` | Get cached diff |
| `POST /projects` | Create project |
| `GET /projects` | List projects |
| `POST /projects/:id/api-keys` | Create API key (returns plaintext once) |
| `GET /projects/:id/api-keys` | List API keys |
| `DELETE /projects/:id/api-keys/:keyId` | Revoke key |
| `POST /baselines` | Save trace as baseline |
| `GET /baselines` | List baselines |
| `GET /healthz` / `GET /readyz` | Health checks |

## Configuration (Environment Variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `sqlite:///./data/blackbox.db` | SQLite connection string |
| `DUCKDB_PATH` | `./data/spans.duckdb` | DuckDB file path |
| `SECRET_KEY` | *required* | Secret for signing (32+ chars) |
| `DIFF_SERVICE_URL` | `http://diff-service:5001` | Diff service URL |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

## Development

```bash
# Install dependencies
pnpm install

# Run all checks
pnpm run lint
pnpm run typecheck
pnpm run test

# Build all packages
pnpm run build

# Run API locally (requires Go 1.22+)
cd apps/api && go run ./cmd/server

# Run diff-service locally
cd apps/diff-service && bun run dev

# Run web locally
cd apps/web && pnpm run dev
```

See [AGENTS.md](AGENTS.md) for complete command reference.

## License

Apache 2.0 — see [LICENSE](LICENSE) for details.