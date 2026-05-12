---
name: pr-review
description: Reviews a PR against KEB conventions (step interface, storage access, docs metadata, FIPS compliance, etc.).
---

# pr-review

Review a pull request with KEB-specific context in mind.

## Usage

```
/pr-review <PR number or URL>
```

**Examples:**
- `/pr-review 3178`
- `/pr-review https://github.com/kyma-project/kyma-environment-broker/pull/3178`

---

## What to do

Fetch the PR diff and review it through the lens of KEB conventions. Structure your review as:

### Summary
One paragraph: what the PR does and whether the approach makes sense.

### Issues (if any)

List only real problems — not style nits unless they violate a hard rule. For each issue:
- **File:line** — description of the problem
- Severity: `blocking` | `suggestion`

### KEB checklist

Evaluate every item below. **Only print items that have a note or failure** — skip items that pass cleanly. If everything passes, write a single line: `KEB checklist: all clear`.

Use this format for printed items only:
- ⚠️ **[category]** — description of the note
- ❌ **[category]** — description of the failure

**Process steps (if any new/modified steps):**
- Step implements `Name()` and `Run(operation, log)` correctly
- `Name()` return value matches file name: `create_runtime_resource_step.go` → `"Create_Runtime_Resource"`
- `operationManager.RetryOperation` used for transient failures; `RetryOperationWithoutFail` used in deprovisioning when exhausting retries without failing the operation
- `kebError` dependency type matches the external system called: `KEBDependency` for internal logic, `InfrastructureManagerDependency` / `LifeCycleManagerDependency` for external components
- Conditional steps use a `StepCondition` function, not inline `if` in `Run`
- New step has a unit test using `storage.NewMemoryStorage()` and `fixture.*` helpers

**Storage:**
- Storage is accessed only through interfaces from `internal/storage/storage.go`
- No direct DB driver imports outside `internal/storage/driver/`

**Documentation:**
- Any new/modified `.md` file in `docs/` (excluding `docs/assets/`) has `<!--{"metadata":{"publish":true/false}}-->` on line 1
- `CLAUDE.md` updated if project structure, conventions, or build/test workflow changed

**Tests:**
- Tests use `testify/assert` and `testify/require` (not Ginkgo)
- Operations constructed via `fixture.FixProvisioningOperation` / `fixture.FixDeprovisioningOperation` / `fixture.FixInstance` — never constructed manually
- No mocked storage in unit tests — use `storage.NewMemoryStorage()`
- `make test` would pass (FIPS140-compliant crypto only: `GODEBUG=fips140=only`; no `crypto/md5`, `crypto/sha1`)

**General:**
- No speculative abstractions or features beyond the PR scope
- Imports cleaned up (no unused imports left from changes)
- No backwards-compatibility shims for removed code
- If a DB schema change is included, migration files exist in `cmd/schemamigrator/` with `.up.sql` and `.down.sql` counterparts

### Verdict
`Approve` / `Request changes` / `Comment` — with one sentence explaining why.

---

## Tone

- Be direct and specific. Point to file and line numbers.
- Don't praise code that is merely correct — save positive comments for non-obvious good decisions.
- Don't suggest improvements outside the PR scope.
