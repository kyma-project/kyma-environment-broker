# GitHub Actions workflows

## ESLint workflow

This [workflow](/.github/workflows/run-eslint.yaml) runs the ESLint for `*.js` files located in `testing/e2e/skr` when they are changed.

The workflow:
- checks out code 
- invokes `make lint -C testing/e2e/skr`

## Markdown link check workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) checks for broken links in all Markdown files. It is triggered:
- as a periodic check that runs daily at midnight on the main branch in the repository 
- on every pull request that creates new or introduces changes to the existing Markdown files
