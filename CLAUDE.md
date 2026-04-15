# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```gig
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

## 5. Project-Specific Guidelines

Kyma Environment Broker (KEB) is an Open Service Broker that provisions SAP BTP, Kyma runtime on third-party cloud providers. It orchestrates:
- **Infrastructure Manager (KIM)** — provisions cluster infrastructure by creating `Runtime` Kubernetes resources
- **Kyma Lifecycle Manager (KLM)** — installs Kyma on clusters by creating `Kyma` Kubernetes resources

The overall flow: user sends provisioning request → KEB creates `Runtime` + `Kyma` CRs → KIM provisions cluster → KLM manages Kyma modules.

### Commands

```bash
# Run all tests (includes FIPS140 flags)
make test

# Run a single test package
go test ./internal/broker/...

# Run a single test by name
go test -run TestFunctionName ./internal/broker/

# Lint
make go-lint

# Full verification (tests + lint + go mod tidy check) — mirrors GitHub Actions
make verify

# Auto-fix lint issues and tidy modules
make fix

# Check go.mod/go.sum consistency
make checks
```

### Architecture

#### Entry Points (`cmd/`)

| Binary | Purpose |
|--------|---------|
| `cmd/broker/` | Main broker — handles provision/deprovision/update via OSB API |
| `cmd/expirator/` | Expires trial/free environments after TTL |
| `cmd/environmentscleanup/` | Cleans up stale environments |
| `cmd/runtimereconciler/` | Reconciles runtime state |
| `cmd/schemamigrator/` | PostgreSQL schema migrations |
| `cmd/deprovisionretrigger/` | Retries failed deprovisionings |
| `cmd/subaccountsync/` | Syncs subaccount data |
| `cmd/accountcleanup/` | Cleans up orphaned accounts |

#### Key Internal Packages (`internal/`)

- **`broker/`** — Core OSB API endpoints (provision, deprovision, update, bind, catalog). Endpoints delegate to operation managers in `process/`.
- **`process/`** — Workflow orchestration via step-based state machines. Sub-packages: `provisioning/`, `deprovisioning/`, `update/`. Each operation is a sequence of steps run by a worker.
- **`storage/`** — PostgreSQL persistence via `internal/storage/postsql/`. Main entities: `Instance`, `Operation`, `RuntimeState`, `Orchestration`, `Binding`.
- **`provider/`** — Cloud provider abstractions. Each supported hyperscaler implements the provider interface.
- **`hyperscalers/`** — Hyperscaler account pool (HAP) logic.
- **`config/`** — Configuration loaded from ConfigMaps; uses `vrischmann/envconfig` for env vars.
- **`events/`** — CloudEvent publishing for broker lifecycle events.
- **`metrics/`** — Prometheus metrics registration and collection.
- **`expiration/`** — Trial and free plan TTL enforcement.
- **`suspension/`** — Handles suspension/unsuspension of instances.

#### Shared Code (`common/`)

Packages shared across binaries: `common/gardener/`, `common/runtime/`, `common/hyperscaler/`, `common/storage/`, `common/deprovision/`, `common/events/`, `common/pagination/`.

#### Storage Layer

PostgreSQL accessed through `internal/storage/`. The `postsql/` sub-package contains the real implementations; `memory/` contains in-memory implementations used in tests. Storage interfaces are defined in `internal/storage/driver.go` (or similar root files).

#### Configuration

Loaded at startup from environment variables via `vrischmann/envconfig`. Extensive configuration is documented in `docs/contributor/02-30-keb-configuration.md`.

### Testing Conventions

- Tests use `testify/assert` and `testify/require` — not Ginkgo.
- Integration-style tests that need a database use PostgreSQL.
- Fast, short unit tests use the in-memory storage from `internal/storage/driver/memory/`.
- Test suites use `TestMain` or `suite_test.go` files for shared setup.
