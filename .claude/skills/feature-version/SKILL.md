---
name: feature-version
description: Find which KEB release version(s) a feature or enhancement shipped in, given a natural language description.
---

# feature-version

Search KEB GitHub release notes to find which version(s) a feature, enhancement, or bug fix first appeared in.

## Usage

```
/feature-version <natural language description>
```

**Examples:**
- `/feature-version HA zones for additional worker pools`
- `/feature-version audit log access parameter`
- `/feature-version dynamic node volume sizes`
- `/feature-version Azure zone discovery`

---

## What to do

1. **Fetch all release tags** — run:
   ```
   gh release list --repo kyma-project/kyma-environment-broker --limit 100 --json tagName,createdAt --order desc
   ```
   This gives you all releases newest-first.

2. **Fetch release bodies in batches of 10** — for each tag in the batch, run:
   ```
   gh release view <tag> --repo kyma-project/kyma-environment-broker --json tagName,body
   ```
   Run up to 10 of these in parallel per batch.

3. **Scan each batch semantically** — read each release body and match against the user's query. Release bodies contain PR titles grouped under headings like "New feature", "Enhancement", "Fixed bugs". Look for PRs whose titles are semantically related to the query — exact keyword match is not required, use meaning.

4. **Stop early when appropriate** — if you've scanned several consecutive releases with no matches after finding at least one match, you can stop. If no matches after scanning all releases, say so clearly.

5. **Check for notable-changes enrichment** — for each matched release tag, check if `notable-changes/<tag>/notable-change.md` exists locally (use the Read tool). If it exists, include a short excerpt.

6. **Present results** in this format for each matching release:

   ```
   ## v<version>

   **Matching PRs:**
   - [PR title](https://github.com/kyma-project/kyma-environment-broker/pull/<n>)

   **Notable change:** (only if notable-changes/<version>/notable-change.md exists)
   <1-3 sentence excerpt from the notable change>
   ```

   If no matches found: say "No matching release found for: `<query>`" and suggest rephrasing or trying different keywords.

---

## Rules

- Always use `kyma-project/kyma-environment-broker` as the repo — never the fork.
- Fetch up to 100 releases. KEB versions start at 1.x — there are no 0.x releases to worry about.
- Do not guess PR numbers. Extract them only from the release body markdown links.
- If the notable-change file does not exist for a matched version, omit the "Notable change" section entirely — do not mention its absence.
- List all matching releases, not just the earliest. The user may want to know if a feature was backported or if related enhancements shipped in later versions too.
- Relevance over recency — rank by how well the PR title matches the query, not by how recent the release is.