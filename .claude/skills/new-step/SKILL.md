---
name: new-step
description: Scaffolds a new KEB provisioning/deprovisioning/update step with implementation and test file.
---

# new-step

Scaffold a new KEB process step (provisioning, deprovisioning, or update) with its implementation file and test file.

## Usage

```
/new-step <StepName> [provisioning|deprovisioning|update]
```

**Examples:**
- `/new-step ValidateQuota provisioning`
- `/new-step ArchiveAuditLog deprovisioning`
- `/new-step SyncLabels update`

If the operation type is omitted, ask the user which one they want.

---

## What to do

Given `<StepName>` and the operation type, produce two files:

### 1. Implementation file

**Path:** `internal/process/<operation_type>/<snake_case_name>_step.go`

Use this template (replace `{{StepName}}`, `{{snake_case_name}}`, `{{Pascal_Snake_name}}`, `{{package}}`):

- `{{StepName}}` — PascalCase struct name, e.g. `ValidateQuota`
- `{{snake_case_name}}` — file name part, e.g. `validate_quota`
- `{{Pascal_Snake_name}}` — `Name()` return value: PascalCase words joined by underscores, e.g. `Validate_Quota`
- `{{package}}` — Go package name (`provisioning`, `deprovisioning`, or `update`)

```go
package {{package}}

import (
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type {{StepName}}Step struct {
	operationManager *process.OperationManager
}

func New{{StepName}}Step(os storage.Operations) *{{StepName}}Step {
	step := &{{StepName}}Step{}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *{{StepName}}Step) Name() string {
	return "{{Pascal_Snake_name}}"
}

func (s *{{StepName}}Step) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	// TODO: implement
	return operation, 0, nil
}
```

**Key rules:**
- `kebError` dependency: use `KEBDependency` as default; change to `InfrastructureManagerDependency`, `LifeCycleManagerDependency`, etc. if the step calls an external system — ask the user if unsure.
- Add only the dependencies actually needed (storage, k8sClient, etc.). Don't add fields speculatively.
- The package name matches the directory: `provisioning`, `deprovisioning`, or `update`.

### 2. Test file

**Path:** `internal/process/<operation_type>/<snake_case_name>_step_test.go`

Use this template:

Use the fixture function that matches the operation type:
- provisioning → `fixture.FixProvisioningOperation("op-id", "instance-id")`
- deprovisioning → `fixture.FixDeprovisioningOperation("op-id", "instance-id")`
- update → `fixture.FixUpdatingOperation("op-id", "instance-id")`

```go
package {{package}}

import (
	"log/slog"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test{{StepName}}Step_HappyPath(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := New{{StepName}}Step(st.Operations())

	op := fixture.Fix{{OperationType}}Operation("op-id", "instance-id")
	require.NoError(t, st.Operations().InsertOperation(op))

	// when
	result, retry, err := step.Run(op, slog.Default())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	_ = result
}
```

**Key rules:**
- Use `fixture.FixProvisioningOperation` / `fixture.FixDeprovisioningOperation` / `fixture.FixInstance` from `internal/fixture/`.
- Use `storage.NewMemoryStorage()` — never PostgreSQL in unit tests.
- Use `slog.Default()` for the logger. If the package already defines a `fixLogger()` helper (check other `_test.go` files in the same package), use that instead — but do not redefine it.
- Use `require` for setup (fatal on failure), `assert` for outcome checks.

### 3. Registration reminder

After creating the files, remind the user:

> **Next step:** Register the new step in `cmd/broker/<operation_type>.go` by adding an entry to the `<operation_type>Steps` slice:
>
> ```go
> {
>     step: <package>.New{{StepName}}Step(db.Operations()),
> },
> ```
>
> Add it at the position in the pipeline where it logically belongs. If you want a conditional step, add a `condition` field (provisioning only).

---

## Conventions to follow

- Step names use PascalCase for the struct; `Name()` returns the step name with words separated by underscores (e.g. struct `ValidateQuotaStep` → `Name()` returns `"Validate_Quota"`, struct `ApplyKymaStep` → `Name()` returns `"Apply_Kyma"`).
- Do not add error handling for impossible paths.
- Do not add comments unless the logic is non-obvious.
- Match the style of existing steps in the same package — read one before generating.
