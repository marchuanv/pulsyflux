---
inclusion: auto
---

# CI Workflow Guide

## Rules

- The GitHub Actions CI workflow MUST be identical regardless of where it runs — GitHub Actions, local via Act, or any other runner. No conditional logic, no fallbacks, no environment-specific branches.
- The workflow MUST run successfully locally using `act` with `-self-hosted` mode on Windows before being pushed to GitHub.
- Use a single `build-and-test` job for Go stages. The `publish` job is separate — it only runs on push to main after build-and-test passes.
- Always test workflow changes locally with `run-ci.ps1` before committing.

## Workflow Structure

- `build-and-test` — checkout, setup Go, download deps, verify checksums, govulncheck, vet, test
- `publish` — auto-bumps patch version from latest tag, creates and pushes new tag, hits Go module proxy

## Environment

- `GONOSUMDB: ""` — no exclusions from checksum database
- `GOFLAGS: -mod=readonly` — prevents accidental go.mod changes in CI
- `permissions: contents: write` — needed for tag creation in publish job

## Publish Flow

1. Runs only on push to `main` (not PRs)
2. Finds latest `v*` tag, bumps patch (or starts at `v0.1.0`)
3. Creates and pushes new tag
4. Requests `github.com/pulsyflux/broker@<tag>` from `proxy.golang.org` to cache the version
