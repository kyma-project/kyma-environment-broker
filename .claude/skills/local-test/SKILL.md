---
name: local-test
description: Fetches a KEB PR, generates targeted test cases from the diff, spins up a local k3d cluster, installs KEB, runs the tests, then tears down the cluster.
---

# local-test

Run end-to-end local tests for a KEB pull request on a k3d cluster.

## Usage

```
/local-test <PR number or URL>
```

**Examples:**
- `/local-test 3201`
- `/local-test https://github.com/kyma-project/kyma-environment-broker/pull/3201`

---

## Safety rules (read first, apply throughout)

- **Always check your kubectl context before any cluster operation.** The cluster name is `keb-test-pr-<PR>`. If the context points somewhere else, stop and ask the user.
- **Never delete a cluster without a second context check.** Confirm the context still points to `keb-test-pr-<PR>` immediately before running `k3d cluster delete`.
- **Never use `kubectl delete`, `k3d cluster delete`, or destructive `helm` commands against a context you did not create in this session.**
- If any step produces an unexpected error, pause, show the user the output, and ask how to proceed — do not silently skip.

---

## Step 1 — Resolve the PR number

Extract the PR number from the argument:
- If a plain number, use it directly.
- If a full GitHub URL, extract the number from the path.
- If nothing was passed, run:

```bash
gh pr view --repo kyma-project/kyma-environment-broker --json number --jq .number
```

Call this `<PR>` for the rest of the skill. Set `CLUSTER_NAME=keb-test-pr-<PR>`.

---

## Step 2 — Fetch the PR diff and generate test cases

Fetch the PR details and diff:

```bash
gh pr view <PR> --repo kyma-project/kyma-environment-broker --json title,body,files
gh pr diff <PR> --repo kyma-project/kyma-environment-broker
```

Analyze the diff and produce a numbered **Test Plan** — a concrete list of OSB API calls (provision / update / deprovision) or kubectl observations that directly exercise the changed code paths. Show it to the user before continuing.

Rules for building the test plan:
- Each test case must map to a specific changed file or function.
- Use the Azure plan (`plan_id: 4deee563-e5ec-4731-b9b1-53b42d855f0c`, region `northeurope`) as the default unless the diff touches a different provider.
- If the diff touches deprovisioning, include a deprovision test case.
- If the diff touches update, include an update test case.
- If the diff touches only non-runtime code (docs, CI, tests), say so explicitly and ask the user whether to proceed with a basic smoke test (provision → deprovision) or skip.
- Keep the list short (≤5 cases). Quality over quantity.

Present the test plan as:
```
Test Plan for PR-<PR>: <PR title>

1. <test name>
   What: <one sentence>
   How: <exact API call or kubectl command>
   Pass criterion: <what you will check>

2. ...
```

Ask the user: **"Does this test plan look correct? Shall I proceed?"** Stop if the user says no.

---

## Step 3 — Create the k3d cluster

**Context check first.** Run:

```bash
kubectl config current-context
```

Show the result to the user. Then create the cluster:

```bash
k3d cluster delete "${CLUSTER_NAME}" 2>/dev/null || true
k3d cluster create "${CLUSTER_NAME}" --k3s-arg "--tls-san=0.0.0.0@server:0"
kubectl cluster-info
```

Verify the context switched to the new cluster:

```bash
kubectl config current-context
```

Confirm it contains `${CLUSTER_NAME}` before continuing. If it does not, stop and tell the user.

---

## Step 4 — Prepare values.yaml

Edit `scripts/values.yaml` directly. Apply only the overrides that the diff requires. Common cases:
- If the diff changes provisioning step timeouts, set `provisioning.maxStepProcessingTime` accordingly.
- If the diff touches worker counts, set `provisioning.workersAmount` / `update.workersAmount` / `deprovisioning.workersAmount`.
- Otherwise leave the defaults unchanged.

Show the user which keys were changed (if any) and their new values. `make install` reads `scripts/values.yaml` directly, so changes here take effect immediately.

---

## Step 5 — Verify the PR image exists

Before installing, confirm the Docker image for this PR was actually built. Run:

```bash
curl -s -o /dev/null -w "%{http_code}" \
  "https://europe-docker.pkg.dev/v2/kyma-project/dev/kyma-environment-broker/manifests/PR-<PR>"
```

A `200` means the image exists. Anything else (typically `404` or `401`) means it does not.

If the status code is not `200`, **stop immediately** and tell the user:

> The image `europe-docker.pkg.dev/kyma-project/dev/kyma-environment-broker:PR-<PR>` does not exist yet.
> The PR build job must be triggered first. Once the build succeeds, re-run this skill.

Do not attempt to install KEB with a missing image.

---

## Step 6 — Install KEB

Run:

```bash
make install VERSION=PR-<PR>
```

This uses `scripts/installation.sh` which:
1. Creates namespaces (`kcp-system`, `kyma-system`, `istio-system`, `garden-kyma-dev`)
2. Installs Istio CRDs and PostgreSQL
3. Applies gardener credentials, CRDs, secrets, and bindings
4. Deploys the KEB Helm chart from `europe-docker.pkg.dev/kyma-project/dev` at tag `PR-<PR>`
5. Waits up to 120 s for the broker pod to become `Ready`

If the command fails, show the full output and stop. Do not attempt to work around a failed install.

After a successful install, port-forward KEB in the background:

```bash
kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 5432:5432 &
PORT_FORWARD_PID=$!
sleep 3
```

---

## Step 7 — Pre-test context check

**Hard stop before running any test.** Run:

```bash
kubectl config current-context
k3d cluster list
```

Verify:
1. Current context still points to `${CLUSTER_NAME}`.
2. `k3d cluster list` shows `${CLUSTER_NAME}` as running.

Print both outputs to the user. If either check fails, stop immediately and do not run tests.

---

## Step 8 — Execute test cases

Work through each test case from the Test Plan in order. For each case:

1. **Announce** the test case number and name.
2. **Run** the exact API call or kubectl command.
3. **Wait** for the operation to reach a terminal state using the appropriate pattern:

   **Provisioning** — poll until `succeeded` or `failed`:
   ```bash
   while true; do
     STATUS=$(curl -s http://localhost:8080/runtimes \
       --header 'X-Broker-API-Version: 2.16' \
       | jq -r --arg iid "<instance_id>" '.data[] | select(.instanceID==$iid) | .status.provisioning.state')
     echo "Provisioning status: $STATUS"
     [ "$STATUS" = "succeeded" ] && break
     [ "$STATUS" = "failed" ] && { echo "FAIL"; break; }
     sleep 5
   done
   ```

   **Simulating KIM/KLM** (needed for provisioning to complete):
   ```bash
   RUNTIME_ID=$(curl -s http://localhost:8080/runtimes \
     --header 'X-Broker-API-Version: 2.16' \
     | jq -r --arg iid "<instance_id>" '.data[] | select(.instanceID==$iid) | .runtimeID')
   make run-provisioning-flow RUNTIME_ID="${RUNTIME_ID}"
   ```

   **Update** — same poll loop using `.status.update.state`.

   **Deprovision** — poll until the instance disappears from `/runtimes` or `.status.deprovisioning.state` reaches `succeeded`.

4. **Evaluate** the pass criterion from the Test Plan.
5. **Record** `PASS` or `FAIL` with a one-line reason.

Accumulate results; continue to the next case even on failure.

---

## Step 9 — Pre-cleanup context check

**Hard stop before deleting the cluster.** Run:

```bash
kubectl config current-context
k3d cluster list
```

Print both outputs. Verify:
1. Current context still points to `${CLUSTER_NAME}`.
2. `k3d cluster list` shows `${CLUSTER_NAME}`.

If the context no longer matches, **do not delete anything**. Tell the user exactly what context was found and ask them to manually clean up.

---

## Step 10 — Remove the cluster

Kill the port-forward (best effort) and delete the cluster:

```bash
kill "${PORT_FORWARD_PID}" 2>/dev/null || true
k3d cluster delete "${CLUSTER_NAME}"
```

Confirm deletion:

```bash
k3d cluster list
```

---

## Step 11 — Print summary

Print a structured summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Local Test Summary — PR-<PR>: <PR title>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Cluster : keb-test-pr-<PR>  [deleted]
KEB tag : PR-<PR>

Results:
  1. <test name> ............. PASS / FAIL
     <one-line reason if FAIL>
  2. ...

Overall: PASS / FAIL (<n>/<total> passed)

Notes:
  <any warnings, skipped cases, or follow-up items>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

If any test failed, suggest the most likely next step (check logs with `kubectl logs -n kcp-system deploy/kcp-kyma-environment-broker`, re-run a specific case, etc.).
