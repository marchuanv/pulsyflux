# CI Workflow Guide

## Rules

- The GitHub Actions CI workflow MUST be identical regardless of where it runs — GitHub Actions, local via Act, or any other runner. No conditional logic, no fallbacks, no environment-specific branches.
- The workflow MUST run successfully locally using `act` with `-self-hosted` mode on Windows before being pushed to GitHub.
- Use a single `build-and-test` job for the Go and Node.js stages so artifacts don't need to be passed between jobs. This avoids Act artifact compatibility issues and keeps the workflow consistent everywhere.
- The publish job is separate because it has different trigger conditions (tag-only) and permissions (OIDC).
- Always test workflow changes locally with `run-ci.ps1` before committing.
- Use `--ignore-scripts` with `pnpm install` in CI to skip the postinstall build (CI builds Go and Node.js artifacts explicitly in separate steps).
