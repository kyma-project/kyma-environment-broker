---
name: create-pr
description: Creates a pull request from a fork branch to kyma-project/kyma-environment-broker main.
---

# create-pr

Create a pull request from the current fork branch to the upstream `kyma-project/kyma-environment-broker` repository.

## Usage

```
/create-pr [optional hint about what changed]
```

**Examples:**
- `/create-pr`
- `/create-pr Add integration test for autoScalerMin`

---

## What to do

### 1. Confirm intent

Ask the user:

> You are about to open a PR from branch `<current-branch>` to `kyma-project/kyma-environment-broker:main`. Continue?

Stop if they say no.

### 2. Gather changes

Run the following to understand what this branch adds on top of main:

```bash
git log main..HEAD --oneline
git diff main..HEAD
```

Use the commit log and diff to generate a concise bullet-list summary of changes for the PR description.

### 3. Draft the PR body

Read `.github/pull-request-template.md` and use it as the body structure. Fill in the **Description** bullets from the diff.

Rules:
- Never include `Closes #<issue>`, `Fixes #<issue>`, or any issue-closing keywords unless the user explicitly asks.
- Keep bullets factual and concise — derived from the actual diff, not paraphrased vaguely.

### 4. Ask about related issues

Ask the user:

> Would you like to reference any related issues? (e.g. `See also #123`) If yes, provide the issue number(s).

If yes, append the references to the **Related issue(s)** section. Use `See also #<n>` unless the user explicitly asks for a closing keyword.

### 5. Ask about the changelog label

Ask the user which changelog label to apply:

> Which changelog label should this PR have?
> - `kind/feature` — New feature
> - `kind/enhancement` — Enhancement to an existing feature
> - `kind/bug` — Bug fix

Apply exactly one of these labels. No other labels unless the user explicitly requests them.

### 6. Draft the PR title

Derive the title from the commits / diff:

- Imperative mood, title case, no trailing period, ≤72 chars.
- No `feat:`, `fix:`, `chore:` prefixes.
- Examples: `Add integration test for autoScalerMin changes`, `Fix deprovisioning race condition`

Show the user the full title + body and ask: "Shall I open the PR with this title and description?"

### 7. Create the PR

To determine `<fork-owner>` and the PR creator's GitHub username, run:

```bash
git remote get-url origin
gh api user --jq .login
```

Extract `<fork-owner>` from the origin URL. The PR creator's GitHub username comes from `gh api user`.

```bash
gh pr create \
  --repo kyma-project/kyma-environment-broker \
  --base main \
  --head <fork-owner>:<current-branch> \
  --title "<title>" \
  --label "<chosen-label>" \
  --assignee "<github-username>" \
  --body "$(cat <<'EOF'
<body>
EOF
)"
```

Return the PR URL to the user once created.

---

## Rules

- Always target `--base main` on `kyma-project/kyma-environment-broker` — never an upstream feature branch.
- Never add issue-closing keywords (`Closes`, `Fixes`, `Resolves`) unless the user explicitly asks.
- Never use `--no-verify` or bypass any git hooks.
- Apply exactly one `kind/*` label per PR.
