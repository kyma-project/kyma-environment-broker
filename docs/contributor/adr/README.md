<!--{"metadata":{"publish":false}}-->

# Architecture Decision Records

Architecture Decision Records (ADRs) are a way to document important architectural decisions made during the development of the project. They provide a clear and concise record of the decision, the context in which it was made, the options considered, and the consequences of the decision.

## Guidelines for Writing ADRs

When writing an ADR, follow these guidelines:

1. To ensure consistency across all KEB-related ADRs, use the [`adr-template`](../../assets/adr-template.md).
2. Number the `adr` files sequentially, following the format `ADR-XXX-title.md`.
3. One ADR must document a single significant decision.
4. Keep the discussion of options and rationale within the pull request that proposes the ADR.
5. On merge, update the ADR status from `Proposed` to the appropriate value. When an ADR supersedes another, reference the superseded ADR explicitly within its content to maintain traceability and context. Change the status of the superseded ADR from `Accepted` to `Superseded by ADR-XXX`.
