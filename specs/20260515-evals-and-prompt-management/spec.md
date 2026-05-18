# Spec: Evals & Prompt Management

> Status: **Phases 1 / 2A / 2B / 3A / 3B shipped** · Created: 2026-05-15 · Last update: 2026-05-18 · Owner: Camilo
> Origin: dogfooding Lelemon on the Venpu WhatsApp agent surfaced the gap — see "Why now".

## 0. Resume guide (read this first after a restart)

**Where everything lives**
- Branch: `feat/evals-and-prompt-management` (pushed to `origin`, off `main`).
- PR draft URL: `https://github.com/lelemondev/lelemon/pull/new/feat/evals-and-prompt-management`
- Single commit: `0ea365c` · 67 files · +20 075 / -13.

**One-shot sanity check** (run from `lelemon/`):
```bash
cd apps/server  && go test ./... && cd ../..
cd ee/server    && go build ./... && cd ../..
cd apps/web     && npx vitest run && pnpm build
```
Expected: 8 Go packages green (eval handler suite ~14 s), EE compiles, 95 vitest cases pass (5 files), Next build succeeds with the 4 new dataset routes + 4 new prompt routes.

**Code map**

| Domain | Entity | Store | Service | Handlers | Frontend |
|---|---|---|---|---|---|
| Datasets | `pkg/domain/entity/dataset.go` | `pkg/infrastructure/store/{sqlite,postgres,clickhouse}/dataset.go` | `pkg/application/dataset/` | `handler/{dataset,dashboard_dataset}.go` | `app/dashboard/datasets/**` + `components/traces/AddToDatasetDialog.tsx` |
| Evals | `entity/eval.go` | `store/{sqlite,postgres,clickhouse}/eval.go` | `pkg/application/eval/` (incl. `scoring.go`) | `handler/{eval,dashboard_eval}.go` | `app/dashboard/datasets/[id]/evals/**` |
| Prompts | `entity/prompt.go` | `store/{sqlite,postgres,clickhouse}/prompt.go` | `pkg/application/prompt/` | `handler/{prompt,dashboard_prompt}.go` | `app/dashboard/prompts/**` + `lib/diff.ts` |
| Cross-cutting | `entity/errors.go` (`ErrUnsupported`) · `entity/eval.go` (`EvalRunFilter.PromptVersionID`) · `entity/trace.go` (`TraceFilter.PromptVersionID`) | three stores extend `ListTraces` and `ListEvalRuns` | wiring in `apps/server/cmd/server/main.go` AND `ee/server/cmd/server/main.go` | `handler/responses.go` (`writeJSON`/`writeJSONError`) | `app/dashboard/layout.tsx` (Datasets + Prompts nav) · `lib/api.ts` |

**Shipped (this branch)**
- **Phase 1** — Datasets from traces (entity + 3 stores + ISP service + API-key & dashboard surfaces + "Add to dataset from span" UI + CSV/JSON import). Side-effect: enabled SQLite `foreign_keys=on` (latent bug fix — every `ON DELETE CASCADE` in the schema was a no-op).
- **Phase 2A** — Deterministic eval engine (4 built-in scorers: `exact_match`, `contains`, `json_path` with 6 ops, `regex`). Run lifecycle (start → post → finalize) with server-side scoring, anti-leak across datasets/tenants, idempotent SQL-transactional finalize. Dashboard for inspection, API-key for the SDK harness loop. **E2E test** in `handler/eval_harness_e2e_test.go` covers the full CI workflow.
- **Phase 2B** — `client_reported` scorer (TDD red→green). The SDK supplies verdicts for scorers the platform won't run server-side (LLM-as-judge with the customer's own provider key, domain-specific assertions). Built-in scorers still server-side — clients cannot override. Missing client scores produce an error result, not a silent pass.
- **Phase 3A** — Prompts + immutable versions. `UNIQUE(prompt_id, version)` mapped to `ErrConflict` (typed `pgconn.PgError` for PG, string-match for SQLite). `createdBy` threaded from JWT user (dashboard) or nil (API-key). `EvalRunFilter.PromptVersionID` closes the loop on the eval side.
- **Phase 3B** — Trace metadata filter (SQLite `JSON_EXTRACT`, Postgres `metadata->>`, ClickHouse `JSONExtractString`) wired across all three stores + handler + frontend (URL `?promptVersionId=` + clearable banner). Unified line-diff between versions (LCS, hand-rolled in `lib/diff.ts`, 11 vitest cases). Trace-count payoff card now shows real numbers and links into the filtered traces view.

**Next step when we resume — pick one of these (in recommended order)**

1. **Phase 4 (prompt registry — runtime fetch)** is the cheapest valuable next step. Adds a `GET /api/v1/prompts/{name}/versions/{label}` endpoint and an SDK helper so customers don't have to hard-code UUIDs. No new infra subsystem. Estimated 1 session. Start in `application/prompt/service.go` with a `GetByLabel(projectID, promptName, versionLabel)` method, RED-GREEN as usual.

2. **Phase 2C (server-side LLM-as-judge)** is the biggest unlock but the largest scope. Sketch:
   - Add a `provider_credentials` table (project-scoped, encrypted at rest) — entity + repo + store (sqlite/postgres real, clickhouse unsupported).
   - Add an outbound `llmclient` package with provider implementations (OpenAI first, then Anthropic). Pluggable interface, SSRF safeguards moot here since URLs are vendor-fixed.
   - Reuse `ingest.AsyncService` worker shape for queued judge execution.
   - New scorer type `llm_judge`; config carries provider, model, judge prompt template; the eval-run worker invokes the model and stores the score + cost.
   - Estimated 2–3 sessions. Phase 2B's `client_reported` covers the use case meanwhile.

3. **Phase 5 (A/B / canary)** is mostly SDK-side. Backend just gains a per-prompt allocation table; SDK reads it and decides which version to use per request. Estimated 1–2 sessions; defer until 4 + 2C are in.

4. **Phase 3B polish** (small, optional): side-by-side diff mode toggle, syntax-highlight templates, and an expression-based index on `metadata->>'prompt_version_id'` when prod volume warrants. Not blocking anything.

**Known minor items (not blockers, but worth a sweep)**
- `apps/server/data/lelemon.db-wal` is tracked in `main` — should be `.gitignore`'d. I reverted my dirty WAL out of the commit; the root issue is pre-existing.
- Several Go `★ minmax`/`★ interface{}→any` linter suggestions in OTHER files in the stores (pre-existing). My new code uses `max()` and `any` consistently.
- Pre-existing `react-hooks/set-state-in-effect` errors in `lib/auth-context.tsx` and a handful of EE components. My new code avoids the pattern via derived state.

**Deferred — with reasons (unchanged)**
- **Phase 2C** (server-side LLM provider invocation) — see next-step #2 for the concrete plan. Needs key management vault + outbound HTTP + worker queue + cost metering.
- **Phase 4** (prompt registry / runtime SDK fetch) — see next-step #1.
- **Phase 5** (A/B / canary traffic split) — see next-step #3.

---

## 1. Why now

Lelemon today is a **pure LLM observability platform**: it captures traces, tokens,
cost, tool usage. That is step 1 of a loop — not the whole loop.

```
prod traces  →  curate dataset  →  eval prompt vN vs vN+1  →  ship the better one  →  observe  →  repeat
  (we have)       (gap)              (gap)                      (gap)                  (we have)
```

**The product argument:** the three pieces are not three features — they are one
workflow. Observability *is the raw material* for evals (you build eval datasets
out of real prod traces). Prompt management *is the thing you change*, and you
can only know if a change helped by tying prompt version → traces → eval scores.
A platform that only does step 1 is a thermometer without a treatment: the
customer has the data but must leave the platform to act on it — and once they
leave (to Langfuse / Braintrust / LangSmith) they take their observability with
them. Tracing alone is the commoditizable piece; the **loop is the moat**.

**The market already converged.** Langfuse (observability → prompts + evals),
LangSmith (all-in-one), Braintrust (evals → observability), Helicone, Arize —
everyone lands on the same trinity: **traces + datasets/evals + prompt versioning**.

**The dogfooding signal.** Lelemon's own design-partner case, the Venpu WhatsApp
agent, hit this wall directly:
- The agent is fully traced with `@lelemondev/sdk`.
- A production bug ("agent says no stock when there is") required: (a) versioning
  the agent's system prompt, (b) running evals to validate a fix.
- Lelemon could do neither → the work was done with throwaway scripts in the Venpu
  repo. That is exactly the churn path every customer will take.

The Venpu eval work (see §6) is, in effect, the **MVP spec for this feature** —
generalized below.

---

## 2. Goals / Non-goals

### Goals
- Let a user **curate a dataset** from real production traces (the data Lelemon
  already stores) without leaving the platform.
- Let a user **run an eval** of an LLM step (prompt, tool-calling decision, output)
  against a dataset, with per-case pass/fail and an aggregate score.
- Let a user **version a prompt** and tie every trace + every eval run to a prompt
  version, so "which prompt produced this conversation / this score" is answerable.
- Close the loop **inside Lelemon** so the customer's daily iteration lives here.

### Non-goals (for the initial scope)
- Full prompt **CMS** with non-engineer editing + runtime fetch/registry. This is a
  separable concern; design for it later (§5, Phase 4). Engineers keeping prompts
  in their own repo is fine — Lelemon just needs to *version and link*, not *host*.
- A/B / canary traffic splitting. Later (Phase 5).
- Auto-optimization / prompt-tuning agents. Out of scope.
- Replacing the customer's CI. Lelemon provides the eval engine + API; gating in
  CI is the customer's wiring.

---

## 3. Core concepts (data model)

Build on the existing multi-tenant + trace store. New entities:

| Entity | Purpose | Notes |
|---|---|---|
| **Dataset** | Named collection of eval cases, scoped to a project. | e.g. "agent vehicle search". |
| **Dataset item** | One eval case: `input` (JSON), optional `expected` (JSON), `metadata`. | Can be created from a trace/span (carry `source_trace_id` + `source_span_id`) or authored manually. |
| **Eval** | A definition of *how* to score: which dataset, which target (a prompt / a tool-calling step / an output), which scorers. | Scorers: exact-match, contains, JSON-field assertions, LLM-as-judge, custom (webhook). |
| **Eval run** | One execution of an Eval. Records per-item results, aggregate score, cost, duration, and the **prompt version** under test. | Comparable across runs → regression view. |
| **Prompt** | A named prompt, project-scoped. | A prompt is a series of **versions**. |
| **Prompt version** | Immutable snapshot: content (template + variables), `version` label, changelog note, `created_at`, `created_by`. | Spans reference `prompt_version_id`. |

Key relationships:
- `span.metadata.prompt_version_id` — the LLM call declares which prompt version ran. **It belongs on the span, not the trace** — a trace can contain several LLM calls with different prompts (see code review).
- `dataset_item.source_trace_id` + `source_span_id` — provenance: this case came from this real trace/span.
- `eval_run.prompt_version_id` — this run tested this version.

This is what makes the loop queryable: *trace → version → eval score → next version*.

> **Code-review reality check.** `entity.Trace` has **no `input`/`output`** — those live on `entity.Span`
> (the `traces` table only stores id/project/name/session/user/status/tags/metadata; the schema doc in
> `lelemon/CLAUDE.md` is stale on this point). So a dataset case is seeded from a **span**, not a trace.
> Spans already carry a `metadata map[string]any` (JSON column in all three stores), which is why
> `prompt_version_id` can start life as span metadata with zero migration.

---

## 4. Reference implementation (Venpu) — the MVP shape

Two evals were built against the Venpu agent. They define the two eval *archetypes*
Lelemon must support:

### Archetype A — deterministic tool/output eval (capa 1)
- **Target:** a deterministic function (here: the `search_vehicle` tool / search
  query). Given an input, output is reproducible.
- **Dataset:** ~30 cases across categories — regression (real failed prod traces),
  typos, accents, natural-language, numeric filters, browse, no-match, multi-tenancy
  isolation, error handling.
- **Scorers:** structural assertions — `expectMinResults`, `expectZero`,
  `expectFindText`, `expectMessageContains`, `expectArgsInclude`.
- **Cost:** ~free, fast, repeatable → good CI gate.
- **Lelemon must support:** datasets, structural scorers, run history, category
  rollups, a pass/fail exit signal.

### Archetype B — LLM decision eval (capa 2)
- **Target:** the LLM itself — given a realistic user message, does it call the
  right tool with sane args (and *not* call it for off-topic/greetings)?
- **Dataset:** ~14 cases — explicit, filters, natural-language, typos, URL,
  negative (greeting / off-topic / job-seeker / location).
- **Scorers:** `expectTool`, `expectNotTool`, `expectArgsInclude`. Run at
  `temperature: 0` for reproducibility; note the live temp differs.
- **Cost:** real model tokens (~$0.14 for 14 cases on Sonnet). → on-demand, not CI.
- **Lelemon must support:** invoking the model with the *real* system prompt +
  tool schema, capturing tool calls, scoring decisions, recording token cost per run.

Both archetypes share: dataset → run → per-case result → aggregate + categories →
cost → comparable across runs. That convergence is the MVP API surface.

---

## 5. Phased scope

### Phase 1 — Datasets from traces (closest to what exists)
- `Dataset` + `DatasetItem` entities, project-scoped, multi-tenant.
- **"Add to dataset" from a span** in the web UI — the single highest-leverage
  action. Pre-fills `input` from the span (input/output live on spans, not traces —
  see §3); user adds/edits `expected`.
- Manual dataset item authoring + CSV/JSON import.
- API + SDK: create/read datasets and items.
- *Deliverable:* a customer can turn real failures into a test set without leaving Lelemon.

### Phase 2 — Eval engine + runs
- `Eval` + `EvalRun` entities. Built-in scorers: exact, contains, JSON-field
  assertion, numeric compare, LLM-as-judge, custom webhook.
- Two execution modes matching §4: **deterministic** (customer's target via webhook
  / SDK harness) and **LLM** (Lelemon invokes the model — reuse existing provider
  cost-calculation code).
- Run view: per-case pass/fail, aggregate score, category rollup, cost, duration.
- Run-to-run **comparison / regression view**.
- API returns a machine-readable pass/fail → customers wire it into their own CI.

### Phase 3 — Prompt versioning (tied to the loop)
- `Prompt` + `PromptVersion` entities (immutable versions + changelog note).
- SDK: attach `promptVersion` to traces (metadata today; first-class field next).
- UI: prompt version list, **diff between versions**, and the payoff view —
  *for a version: its traces, its eval runs, its scores*.
- *Deliverable:* "which prompt version produced this conversation / this score" is
  one click.

### Phase 4 — Prompt registry/CMS (optional, only if pulled)
- Host prompt content, fetch at runtime via SDK, non-engineer editing.
- Adds a runtime dependency to the customer's agent — opt-in, clearly separated.

### Phase 5 — A/B / canary
- Serve version A to X% of traffic, compare eval/online metrics.

---

## 6. Technical grounding (maps onto the current stack)

- **Storage split — corrected.** There is no standalone "Postgres" in the codebase. The
  server wires **two `repository.Store` instances**: `primaryStore` (users, projects — from
  `DATABASE_URL`) and `analyticsStore` (traces, spans — defaults to `primaryStore`, or a
  separate DB via `ANALYTICS_DATABASE_URL`). Datasets/evals/prompts are small relational data
  → they belong in **`primaryStore`**, which is **SQLite by default** and Postgres in
  production (`store.New()` also allows `clickhouse://` as primary — see decision Q7 below).
  Eval runs *read* traces/spans through the `TraceStore` interface on `analyticsStore`
  (which is the thing that may be ClickHouse).
- **`repository.Store` is monolithic.** It composes `ProjectStore + TraceStore +
  AnalyticsStore + UserStore`, and *every* backend (sqlite/postgres/clickhouse) implements
  the whole thing. Adding `DatasetStore`/`EvalStore`/`PromptStore` sub-interfaces means all
  three backends must satisfy them (sqlite + postgres are near-identical SQL; ClickHouse is
  the awkward one — no real `UPDATE`). Alternative precedent: the **EE store** (`ee/server/
  infrastructure/store`) opens its own raw `*sql.DB`, runs `MigrateEnterprise`, and wraps
  `coreStore` — it never touches `repository.Store`. Decision in Q7.
- **`prompt_version_id`:** start as **span metadata** (zero migration — `metadata` is already
  a JSON column everywhere, the SDK already sends it). Add a ClickHouse data-skipping index on
  `JSONExtractString(metadata,'prompt_version_id')` only when query volume needs it. Promote to
  a nullable column in Phase 3: `ALTER TABLE ... ADD COLUMN` is metadata-only/instant in both
  ClickHouse and Postgres, and historical rows are *not* backfilled (readers fall back
  column → metadata). Net migration cost ≈ zero.
- **Go server (Chi, Clean Architecture):** new modules `dataset`, `eval`, `prompt`
  following existing module conventions + multi-tenant isolation rules
  (`.claude/rules/multi-tenant.md`). Eval execution is async/queued — runs can be
  long and cost money. Precedent exists: `ingest.NewAsyncService` already runs a worker pool
  (`application/ingest/worker.go`) — eval runs can reuse that shape.
- **SDK (`lelemondev-sdk`, `-python`):** (a) `promptVersion` field on trace/span;
  (b) optional eval-harness helper so customers can register a deterministic target
  (Archetype A); (c) later, prompt fetch (Phase 4).
- **Web (React):** "Add to dataset" affordance on the trace view; datasets list +
  item editor; eval run view + comparison; prompt version list + diff.
- **EE split:** decide which pieces are OSS vs `ee/` (cf. existing `enterprise.md`
  rule). Suggestion: core datasets/evals OSS; advanced (A/B, large-scale run history,
  LLM-judge templates) EE — matches the Langfuse model.

---

## 7. Open questions / decisions needed

> Resolved against the actual codebase during the 2026-05-15 review. Marked **DECIDED**
> where the code makes the answer clear; **OPEN** where it's still a product call.

1. **Deterministic target execution (webhook vs SDK-driven)** — **DECIDED: SDK-driven for
   the MVP.** The server has *no outbound HTTP / webhook machinery* (the only webhook is
   *inbound* Lemon Squeezy in EE). Building safe outbound execution (retries, timeouts, SSRF
   protection) is real infra. The SDK already has a batching ingestion client — adding an
   eval-harness helper that POSTs per-case results to `POST /eval-runs/:id/results` is far
   smaller and matches the non-goal "don't replace the customer's CI". Webhook-driven
   execution → later / EE.
2. **LLM-as-judge (templates vs BYO)** — **DECIDED: BYO-prompt in OSS, template library in
   EE.** The server already has provider invocation + `service.NewPricingCalculator()` (used
   by ingest/trace), so "invoke model with a user-supplied judge prompt, score, record cost"
   is OSS-cheap. A curated judge-template library is a clean EE differentiator (§6 already
   suggested it).
3. **Prompt version source of truth** — **DECIDED: store-for-audit, don't-serve-at-runtime
   (Phase 3); host+serve = Phase 4.** Lelemon *does* store version content (needed for the
   diff view), but the customer's repo stays the runtime source. The word "reference" in §5
   is misleading — corrected here. Runtime fetch + caching + the SDK runtime dependency stay
   Phase 4.
4. **`prompt_version_id` — metadata vs column** — **DECIDED: span metadata now, nullable
   column in Phase 3.** Goes on the **span**, not the trace (a trace has many LLM calls).
   Zero-migration to start; promotion DDL is instant in CH + PG; no historical backfill. See §6.
5. **OSS vs EE boundary** — **DECIDED (proposed):**
   - **OSS:** all six entities + CRUD; built-in deterministic scorers; BYO LLM-judge;
     SDK-driven eval runs; run history + comparison; "Add to dataset from span" UI; prompt
     diff view.
   - **EE:** curated LLM-judge templates; A/B / canary (Phase 5); long-horizon run retention
     + advanced regression analytics; webhook-driven execution. Matches the Langfuse model.
6. **Pricing** — **DECIDED: BYO-key for OSS** (self-hosted; Lelemon holds no provider keys
   today and has no key vault). For hosted Enterprise, **pass-through metered** is the natural
   model — it can ride the existing EE `UsageStore.Increment` metering + Lemon Squeezy billing,
   but the key-management piece doesn't exist yet → **OPEN / deferred to EE billing work**.
7. **ClickHouse as `primaryStore`** *(new — surfaced by the review)* — `store.New()` allows
   `DATABASE_URL=clickhouse://`, which would put datasets/evals/prompts on ClickHouse (bad fit:
   no `UPDATE`, `ReplacingMergeTree` hacks). **OPEN — recommendation:** implement the new
   stores for **sqlite + postgres**, and have the ClickHouse implementation return
   `entity.ErrUnsupported` for these methods + document "evals/datasets require a SQLite or
   Postgres primary store". Revisit only if a real CH-primary deployment appears.

---

## 8. Success criteria

- A Venpu engineer can: open a failed agent trace → "add to dataset" → run the
  vehicle-search eval → see 30/30 → bump the prompt version → re-run → compare —
  **entirely inside Lelemon**, no throwaway scripts.
- `prompt_version` is queryable on every Venpu agent trace.
- The capa 1 + capa 2 evals from §4 are reproducible as Lelemon Evals.
- A customer can call the eval API from CI and block a merge on a red score.

---

## 9. Phase 1 breakdown — Datasets from traces

**Scope:** `Dataset` + `DatasetItem`, project-scoped; "Add to dataset" from a **span** in the
trace view; manual authoring + CSV/JSON import; dashboard CRUD API. **Entirely OSS** — the
only `ee/` touch is one wiring line in `ee/server/cmd/server/main.go` (EE builds its own
`RouterConfig`). The OSS/EE line only becomes a real decision in Phase 2+.

### Schema (two new tables, in `primaryStore` — sqlite + postgres)

`datasets`: `id` PK · `project_id` (FK projects, **every query filters by it**) · `name` ·
`description` · `created_at` · `updated_at` · index `(project_id, created_at DESC)`.

`dataset_items`: `id` PK · `dataset_id` (FK datasets `ON DELETE CASCADE`) · `project_id`
(denormalized → 1-hop tenant checks) · `input` (JSON) · `expected` (JSON, nullable) ·
`metadata` (JSON) · `source_trace_id` (nullable) · `source_span_id` (nullable) ·
`created_at` · `updated_at` · index `(dataset_id)`.

### Files to touch, in order

1. **`pkg/domain/entity/dataset.go`** *(new)* — `Dataset`, `DatasetItem`, `NewDatasetItem`,
   `DatasetFilter`, `DatasetUpdate`, `DatasetItemUpdate`. Mirror `trace.go` style. Add
   `ErrUnsupported` to `entity/errors.go`.
2. **`pkg/domain/repository/interfaces.go`** — add `DatasetStore` sub-interface (all methods
   take `projectID`), compose into `Store`.
3. **`pkg/infrastructure/store/{sqlite,postgres,clickhouse}/dataset.go`** *(new files)* +
   append `CREATE TABLE` statements to each `Migrate()`. sqlite first (dev default), then
   postgres, then clickhouse = stubs returning `ErrUnsupported` (Q7). Add `dataset_test.go`
   following the existing `store_test.go` pattern.
4. **`pkg/application/dataset/`** *(new)* — `service.go` (`Service` holds the dataset store
   **and** a `TraceStore` to read spans for "add from trace"), `dto.go`. Method
   `AddItemFromTrace` fetches the span via the trace store, pulls `input`/`output`, writes the
   item with `source_trace_id` + `source_span_id`.
5. **`pkg/interfaces/http/handler/dataset.go`** *(new)* — `DatasetHandler`; dashboard routes
   under `/dashboard/projects/{id}/datasets...` incl. `POST .../items/from-trace` and
   `POST .../items/import`. Use the `verifyProjectOwnership` helper (not the inline
   list+loop). Body-size limit is already global (5 MB).
6. **`pkg/interfaces/http/router.go`** — add `DatasetSvc` to `RouterConfig`, register routes
   in the dashboard (session-auth) group.
7. **`apps/server/cmd/server/main.go`** *and* **`ee/server/cmd/server/main.go`** — construct
   `dataset.NewService(primaryStore, analyticsStore)` and pass it into `RouterConfig`
   (easy to miss the EE one).
8. **`apps/web/src/lib/api.ts`** — `Dataset` / `DatasetItem` interfaces (camelCase, follow
   `trace/response.go` tag convention) + `dashboardAPI` methods.
9. **`apps/web/src/app/dashboard/datasets/page.tsx`** + **`datasets/[id]/page.tsx`** *(new)*
   + nav link in **`dashboard/layout.tsx`**.
10. **`apps/web/src/components/traces/AddToDatasetDialog.tsx`** *(new)* — wired into
    **`SpanDetail.tsx`** (input/output live on the span, not the trace). Pre-fills `input`
    from `span.input` / `span.userInput`, `expected` from `span.output` (editable). shadcn
    `Dialog` + `Select`.

*Deliverable:* a Venpu engineer can open a failed agent span → "Add to dataset" → curate the
case → it lands in a project-scoped dataset, no throwaway scripts. Unblocks Phase 2.
