# Implementation Plan: pnpm Migration

## Overview

Migrate the `@pulsyflux/broker` Node.js package from npm to pnpm with supply-chain security hardening, source-distribution packaging, unified namespace, and CI/CD pipeline. Tasks are ordered so each step builds on the previous: configuration files first, then code changes, then build scripts, then CI/CD, then integration tests.

## Tasks

- [ ] 1. Remove npm artifacts and set up pnpm configuration files
  - [ ] 1.1 Delete `nodejs-api/package-lock.json` and `nodejs-api/node_modules/`
    - Remove the npm lockfile and existing node_modules directory
    - _Requirements: 1.1, 1.2, 2.1_

  - [ ] 1.2 Create `nodejs-api/.npmrc` with pnpm settings
    - Add `save-exact=true`, `save-prefix=`, and `engine-strict=true`
    - Do NOT add `shamefully-hoist=true` or `node-linker=hoisted`
    - _Requirements: 11.3, 27.1, 27.2, 27.3, 27.4_

  - [ ] 1.3 Create `nodejs-api/pnpm-workspace.yaml` with security settings
    - Configure `allowBuilds` for `@pulsyflux/broker` and `zig-build`
    - Set `strictDepBuilds: true`
    - Set `minimumReleaseAge: 1440` with `minimumReleaseAgeExclude: ['@pulsyflux/*']`
    - Set `trustPolicy: no-downgrade`
    - Set `blockExoticSubdeps: true`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.8, 24.1, 24.2, 24.3, 25.1, 25.2, 25.3, 25.4, 26.1, 26.2, 26.4_

  - [ ] 1.4 Update `nodejs-api/.gitignore` for pnpm
    - Ensure `node_modules/` is excluded
    - Remove `package-lock.json` from ignored files if present
    - Ensure `pnpm-lock.yaml` is NOT excluded
    - Keep `.bin/`, build artifacts, and other existing ignores
    - _Requirements: 9.1, 9.2, 9.3_

- [ ] 2. Update `nodejs-api/package.json` for migration
  - [ ] 2.1 Rename package and update namespace fields
    - Change `name` from `pulsyflux-broker` to `@pulsyflux/broker`
    - Add `publishConfig: { "access": "public" }`
    - Update `repository.url` to `https://github.com/pulsyflux/broker.git`
    - Update `homepage` to `https://github.com/pulsyflux/broker#readme`
    - Update `bugs.url` to `https://github.com/pulsyflux/broker/issues`
    - _Requirements: 30.3, 30.5, 30.6, 30.8, 30.9, 30.10, 30.11, 8.8_

  - [ ] 2.2 Update engines, packageManager, and lifecycle scripts
    - Set `engines.node` to `>=24.0.0`
    - Add `packageManager` field with `pnpm@10.x.x` (use actual latest pnpm 10 version)
    - Add `preinstall` script: `pnpm dlx only-allow pnpm`
    - Add `postinstall` script: `node postinstall.mjs`
    - _Requirements: 14.1, 14.2, 17.1, 17.2, 17.3, 17.4, 17.5, 8.3, 8.4, 8.5, 8.10_

  - [ ] 2.3 Promote build dependencies and pin versions
    - Move `node-addon-api`, `node-api-headers`, and `zig-build` from `devDependencies` to `dependencies`
    - Keep `jasmine` and `tsx` as `devDependencies`
    - Pin `zig-build` to a specific commit hash: `github:solarwinds/zig-build#<commit-sha>`
    - Pin all dependency versions as exact (no `^` or `~` prefixes)
    - _Requirements: 32.1, 32.2, 32.3, 32.4, 13.1, 13.2, 8.6, 8.9_

  - [ ] 2.4 Update `files` field for source distribution
    - Replace `.bin/release/` with source files: `addon.cc`, `build.mjs`, `registry.mjs`, `postinstall.mjs`
    - Keep `types/`, `README.md`, `LICENSE`
    - Do NOT include `broker_lib.go` or any binary artifacts
    - Keep `main` and `exports` pointing to `.bin/release/registry.mjs` (produced by postinstall)
    - _Requirements: 18.1, 18.2, 18.3, 18.4, 18.5, 18.6, 18.7, 8.2, 8.7_

- [ ] 3. Checkpoint - Verify package.json and config files
  - Ensure all configuration files are correct, ask the user if questions arise.

- [ ] 4. Harden DLL loading in `nodejs-api/addon.cc`
  - [ ] 4.1 Replace bare `LoadLibraryA` call with absolute path resolution
    - Remove the initial `LoadLibraryA("broker_lib.dll")` bare filename call
    - Resolve the absolute path of `broker_lib.dll` from the addon's own module handle using `GetModuleHandleExA` and `GetModuleFileNameA`
    - Use a single `LoadLibraryA` call with the full absolute path
    - Do NOT fall back to bare filename or Windows default DLL search order
    - If the DLL is not found at the expected absolute path, throw a descriptive `Napi::Error` with the expected path
    - _Requirements: 15.1, 15.2, 15.3, 15.4_


- [ ] 5. Create postinstall build script and update Go module
  - [ ] 5.1 Create `nodejs-api/postinstall.mjs` — consumer-side build pipeline
    - Implement Go version check: parse `go version` output, compare against minimum from `go.mod` (≥1.25)
    - Extract the version comparison logic into a testable function (e.g., `checkGoVersion`, `compareVersions`)
    - Fetch Go module: run `go get github.com/pulsyflux/broker@<pinned-version>`
    - Read pinned Go module version from `package.json` metadata (e.g., `goModuleVersion` field)
    - Build Go shared library: run `go build -buildmode=c-shared -o .bin/release/broker_lib.dll`
    - Build C++ addon: run `node build.mjs`
    - Verify outputs exist: `broker_lib.dll` and `broker_addon.node` in `.bin/release/`
    - Exit non-zero with descriptive error on any step failure
    - Do NOT set `GONOSUMCHECK`, `GONOSUMDB`, or `GOFLAGS=-insecure`
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10, 5.11, 5.12, 5.13, 21.1, 21.2, 21.3, 21.4, 31.1, 31.2, 31.3, 20.3, 20.4, 20.5_

  - [ ]* 5.2 Write property test: Go version check correctness (Property 1)
    - **Property 1: Go version check correctness**
    - Use fast-check to generate version tuples (major, minor, patch)
    - For any valid `go version goX.Y.Z <platform>` string, `checkGoVersion` returns true iff version ≥ minimum
    - Minimum 100 iterations
    - **Validates: Requirements 31.2, 31.3**

  - [ ]* 5.3 Write property test: Go version check rejects malformed input (Property 2)
    - **Property 2: Go version check rejects malformed input**
    - Use fast-check to generate arbitrary strings that do not match `go version` format
    - `checkGoVersion` must return false (reject) for all malformed inputs
    - Minimum 100 iterations
    - **Validates: Requirements 31.2**

  - [ ] 5.4 Update `go.mod` module path to canonical GitHub organization path
    - Change `module pulsyflux` to `module github.com/pulsyflux/broker`
    - _Requirements: 19.1, 23.1, 23.4, 30.4, 30.7_

  - [ ] 5.5 Update Go import paths in `broker_lib.go`
    - Change `import "pulsyflux/broker"` to `import "github.com/pulsyflux/broker/broker"`
    - _Requirements: 19.2_

  - [ ] 5.6 Update Go import paths in `broker/server.go` and `broker/client.go`
    - Change `import tcpconn "pulsyflux/tcp-conn"` to `import tcpconn "github.com/pulsyflux/broker/tcp-conn"`
    - Update any other internal imports referencing the old `pulsyflux` module path
    - _Requirements: 19.2_

- [ ] 6. Checkpoint - Verify postinstall script and Go module changes
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Run pnpm install and verify dependency resolution
  - [ ] 7.1 Run `pnpm install` in `nodejs-api/` and verify lockfile generation
    - Verify `pnpm-lock.yaml` is generated
    - Verify all dependencies resolve: `jasmine`, `node-addon-api`, `node-api-headers`, `tsx`, `zig-build`
    - Verify `node-addon-api` and `node-api-headers` are accessible via `require.resolve`
    - If hoisting is needed for native build deps, add minimal hoisting overrides to `.npmrc` (only for specific packages, not `shamefully-hoist`)
    - _Requirements: 3.1, 3.2, 4.1, 4.2, 4.3, 4.4, 4.5, 10.1, 10.2, 10.3, 11.1, 11.2, 12.1, 12.2, 12.3_

  - [ ] 7.2 Verify build pipeline compatibility
    - Run `node build.mjs` to confirm C++ addon compiles
    - Run `pnpm exec jasmine` to confirm test suite passes
    - _Requirements: 7.1, 7.2_

- [ ] 8. Create GitHub Actions CI/CD workflow
  - [ ] 8.1 Create `.github/workflows/ci.yml` with Go stage
    - Trigger on push to main, PRs to main, and version tags `v*.*.*`
    - Set up Go from `go.mod` version
    - Run `go mod download`, `go vet ./...`, `go test ./...`
    - Install and run `govulncheck ./...`
    - Compile Go shared library: `go build -buildmode=c-shared -o broker_lib.dll broker_lib.go`
    - Do NOT set `GONOSUMCHECK`, `GONOSUMDB`, or `GOFLAGS=-insecure`
    - _Requirements: 34.1, 34.2, 34.3, 34.4, 34.9, 34.12, 34.13, 22.1, 22.2, 22.3, 22.4, 20.3, 20.4, 20.5_

  - [ ] 8.2 Add Node.js stage to CI workflow
    - Depends on Go stage
    - Set up Node.js v24 and pnpm (from `packageManager` field)
    - Run `pnpm install --frozen-lockfile` in `nodejs-api/`
    - Run `node build.mjs`, `pnpm exec jasmine`, `pnpm audit`
    - _Requirements: 34.5, 34.10, 34.11, 28.1, 28.2, 28.3, 29.1, 29.2, 29.3_

  - [ ] 8.3 Add integration test stage to CI workflow
    - Depends on Go and Node.js stages
    - Run integration tests from `test/` directory
    - _Requirements: 34.6, 36.6_

  - [ ] 8.4 Add publish stage to CI workflow
    - Only runs on version tags `v*.*.*` and after all previous stages pass
    - Set `permissions: id-token: write` for OIDC provenance
    - Run `pnpm publish --provenance --access public`
    - Use `NODE_AUTH_TOKEN` from secrets
    - Support skipping publish during local testing (no credentials = no publish)
    - _Requirements: 34.7, 34.8, 16.1, 16.2, 16.3, 16.4, 35.6_

- [ ] 9. Checkpoint - Verify CI workflow structure
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 10. Create integration test directory and tests
  - [ ] 10.1 Create `test/go/` — Go consumer verification tests
    - Create a test that sets up a temporary Go module
    - Run `go get github.com/pulsyflux/broker@<version>` to fetch the published Go module
    - Import and exercise the broker package
    - Verify `go build -buildmode=c-shared` produces a valid shared library
    - _Requirements: 36.1, 36.2_

  - [ ] 10.2 Create `test/nodejs/` — Node.js consumer verification tests
    - Create a test that sets up a temporary directory with minimal `package.json`
    - Run `pnpm install @pulsyflux/broker`
    - Verify postinstall produces `broker_lib.dll` and `broker_addon.node`
    - Import `@pulsyflux/broker` and verify Server/Client classes work
    - _Requirements: 36.1, 36.3_

  - [ ] 10.3 Create integration test runner script
    - Make integration tests runnable via a single command (e.g., `pnpm test:integration` or shell script)
    - Add CI pipeline validation tests verifying each stage produces expected outputs
    - _Requirements: 36.4, 36.5_

- [ ] 11. Add Act support for local CI testing
  - [ ] 11.1 Add Act configuration and documentation
    - Optionally create `.actrc` with default runner image and platform mappings
    - Optionally create `.secrets.example` documenting required secrets (without actual values)
    - Add documentation in README or CONTRIBUTING guide on running `act push` / `act pull_request`
    - Ensure CI workflow is compatible with Act (provide fallbacks for GitHub-specific features)
    - _Requirements: 35.1, 35.2, 35.3, 35.4, 35.5_

- [ ] 12. Final checkpoint - Verify complete migration
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate the Go version comparison logic (Properties 1 and 2 from the design)
- Requirements 12 (content-addressable storage), 33 (npm org 2FA), 30.1/30.2 (org creation) are pnpm/npm platform behaviors or manual org setup steps — they are not coding tasks but are covered by the configuration that enables them
