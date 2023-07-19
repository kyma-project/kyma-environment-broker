# GitHub Actions workflows

## ESLint workflow

This [workflow](/.github/workflows/run-eslint.yaml) runs the ESLint for `*.js` files located in `testing/e2e/skr` when they are changed.

The workflow:
- checks out code 
- invokes `make lint -C testing/e2e/skr`
