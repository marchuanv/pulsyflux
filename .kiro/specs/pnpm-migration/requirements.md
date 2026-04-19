# Requirements Document

## Introduction

This document defines the requirements for migrating the Node.js side of the PulsyFlux project from npm to pnpm as the package manager. The primary motivation for this migration is package security, driven by the escalating scale and sophistication of npm supply-chain attacks. In 2025 alone, Sonatype identified 454,600+ new malicious packages (bringing the cumulative total to 1.233 million), with over 99% targeting the npm ecosystem — a 75% year-over-year increase. High-profile incidents underscore the urgency: the Shai-Hulud worm (September–November 2025) was a self-replicating npm worm that abused preinstall scripts to steal developer credentials and spread across 796+ packages with 132 million monthly downloads, compromising maintainer accounts from Zapier, PostHog, and Postman; the Axios hijack (March 2026) saw North Korean group UNC1069 use stolen npm credentials to publish malicious versions of the `axios` package (100M+ weekly downloads) containing a phantom dependency that deployed a cross-platform RAT; and 36 malicious Strapi CMS plugins (2026) disguised as legitimate plugins contained payloads for Redis/PostgreSQL exploitation, reverse shells, and credential harvesting. The chalk/debug compromise (2025) demonstrated that even major packages can be compromised through stolen credentials and phishing, injecting runtime code rather than install scripts.

npm's flat `node_modules` layout allows any package to access any other installed package — even packages it does not explicitly declare as dependencies. These undeclared accesses, known as phantom dependencies, create a supply-chain attack surface: a malicious or compromised transitive dependency can silently import and exploit packages it was never intended to reach. pnpm eliminates this risk through a strict, symlink-based `node_modules` structure where each package can only access its own declared dependencies, and through a content-addressable store that verifies package integrity at the file level.

Beyond dependency isolation, pnpm v10 provides a comprehensive set of supply-chain security features that directly mitigate the attack patterns observed in recent incidents. pnpm blocks lifecycle scripts by default unless packages are explicitly allowlisted via `allowBuilds` (with `strictDepBuilds: true` enforcement), which directly prevents Shai-Hulud-style preinstall/postinstall attacks. The `blockExoticSubdeps` setting prevents transitive dependencies from resolving to non-registry sources (git repos, tarball URLs), mitigating the Axios attack pattern where a phantom dependency was injected pointing to a malicious source. The `minimumReleaseAge` setting delays installing newly published versions by a configurable time window — since the Axios attack lasted only ~3 hours, a 24-hour delay would have prevented exposure entirely. The `trustPolicy: no-downgrade` setting catches account takeover scenarios by rejecting packages whose trust level has decreased compared to previous releases. These features, combined with frozen lockfile enforcement in CI and exact version pinning, form a defense-in-depth strategy against the full spectrum of npm supply-chain attacks.

The migration must configure pnpm's script allowlist to support the project's own native build pipeline — Go shared library compilation and C++ addon compilation — which will be integrated as lifecycle scripts so that `pnpm install` compiles native artifacts locally on the consumer's machine. The published package will contain only source files; no pre-built binaries will be shipped through the npm registry.

The migration also addresses additional supply-chain hardening: npm provenance attestation for published packages, pinning GitHub-hosted dependencies to immutable references, hardening native DLL loading against hijacking, enforcing a supported Node.js engine floor, enforcing pnpm as the sole permitted package manager, blocking exotic transitive dependencies, enforcing trust policy for account takeover detection, configuring minimum release age for malware detection windows, pinning exact dependency versions, enforcing frozen lockfile in CI, and running vulnerability audits.

A key design decision is that the published npm package must NOT contain pre-built binaries or Go source files. The current `files` field in `package.json` includes `.bin/release/` which ships `broker_lib.dll`, `broker_addon.node`, and `registry.mjs` as pre-built artifacts. After migration, the published package will instead contain only the C++ source, build script, registry module, types, and documentation. The Go source code (broker package and cgo bindings) will be published as a proper Go module at `github.com/pulsyflux/broker` and fetched via `go get` during the consumer-side `postinstall` lifecycle script. Consumers who run `pnpm install` and allow the postinstall lifecycle script will first fetch the Go module, then compile the Go shared library and C++ addon locally on their machine. This eliminates the risk of shipping tampered binaries through the npm registry and keeps the Go source in its canonical location as a Go module.

Both the npm package and Go module are published under a unified `pulsyflux` namespace. On npm, the package is scoped as `@pulsyflux/broker` under the `pulsyflux` npm organization. On GitHub/Go, the module is hosted under the `pulsyflux` GitHub organization at `github.com/pulsyflux/broker`. This unified namespace provides consistent branding, prevents typosquatting within the scope, and makes it easier for consumers to identify official packages. In the future, the `pulsyflux` npm organization will host additional scoped packages (e.g., `@pulsyflux/broker`, potentially others), and the `pulsyflux` GitHub organization will host additional Go modules (e.g., `github.com/pulsyflux/broker`, potentially others).

The migration covers the main `@pulsyflux/broker` package in `nodejs-api/`. The goal is to replace all npm artifacts (lockfiles, node_modules) with pnpm equivalents, enforce strict dependency isolation, enforce pnpm-only installation, exclude pre-built binaries from the published package, integrate the native build as consumer-side lifecycle scripts, and harden the supply chain across the board. Integration tests for CI pipeline validation and package installation verification (both Go and Node.js consumers) are housed in a dedicated test directory at `pulsyflux/test/`.

## Glossary

- **Migration_Tool**: The set of scripts, commands, and manual steps used to perform the npm-to-pnpm migration
- **Nodejs_Api_Package**: The `@pulsyflux/broker` package (npm scoped name) located at `pulsyflux/nodejs-api/`, containing the native C++ addon, Go shared library bindings, and ES module registry. Previously named `pulsyflux-broker`; renamed to use the `pulsyflux` Npm_Organization scope
- **Integration_Test_Directory**: The directory at `pulsyflux/test/` containing integration tests for CI pipeline validation and package installation verification for both Go and Node.js consumers
- **Lockfile**: A file that pins exact dependency versions for reproducible installs — `package-lock.json` for npm, `pnpm-lock.yaml` for pnpm
- **Node_Modules**: The `node_modules/` directory containing installed dependencies
- **Build_Pipeline**: The sequence of steps to (1) fetch the Go_Module source via `go get github.com/pulsyflux/broker`, (2) compile the Go shared library (`go build -buildmode=c-shared -o .bin/release/broker_lib.dll`) from the fetched Go_Module source, (3) compile the C++ addon (`node build.mjs` via zig-build), and produce native artifacts (`broker_lib.dll`, `broker_addon.node`) in `.bin/release/`. Under the Source_Distribution model, this pipeline runs on the consumer's machine during `pnpm install`, not at publish time. The Go source is fetched from the Go_Module at install time, not bundled in the Published_Package
- **GitHub_Dependency**: A dependency resolved from a GitHub repository URL rather than the npm registry (e.g., `zig-build` from `github:solarwinds/zig-build`)
- **Phantom_Dependency**: A package that code can import at runtime despite it not being declared in that package's `package.json` dependencies — possible under npm's flat `node_modules` layout but prevented by pnpm's strict isolation
- **Content_Addressable_Store**: pnpm's global store where package files are stored by the hash of their contents, ensuring that installed files match their expected integrity checksums
- **Strict_Isolation**: pnpm's default `node_modules` layout where each package can only access dependencies it explicitly declares in its own `package.json`, enforced via symlinks
- **Lifecycle_Script**: A script defined in `package.json` under `scripts` that runs automatically at specific points during `npm install` or `pnpm install` (e.g., `preinstall`, `install`, `postinstall`)
- **Script_Allowlist**: pnpm's security mechanism that blocks Lifecycle_Scripts by default and only permits them for packages explicitly listed in the `pnpm.onlyBuiltDependenciesFile` or `.npmrc` configuration
- **Provenance_Attestation**: A cryptographically signed statement generated during CI/CD publishing (via `npm publish --provenance`) that links a published package version to its source commit and build workflow, enabling consumers to verify the package origin
- **DLL_Hijacking**: An attack where a malicious DLL is placed in a directory searched by the Windows DLL loading order (e.g., the current working directory), causing the application to load the attacker's DLL instead of the legitimate one
- **Engine_Floor**: The minimum Node.js version specified in the `engines` field of `package.json`, used by package managers to reject installs on unsupported runtimes
- **Preinstall_Guard**: A `preinstall` Lifecycle_Script that checks the `npm_config_user_agent` environment variable or uses a tool like `only-allow` to reject installation attempts from package managers other than pnpm
- **Published_Package**: The set of files included in the tarball uploaded to the npm registry, as determined by the `files` field in `package.json`
- **Source_Distribution**: A packaging model where the Published_Package contains only source files and build scripts, and consumers compile native artifacts locally during installation via Lifecycle_Scripts
- **Go_Module**: The Go package published at `github.com/pulsyflux/broker` under the `pulsyflux` GitHub_Organization, containing the broker implementation (`broker` package) and cgo bindings (`broker_lib.go`). Consumers fetch this module via `go get` during the Build_Pipeline rather than receiving Go source bundled in the Published_Package. Previously hosted at `github.com/marchuanv/pulsyflux`; relocated to the GitHub_Organization for unified namespace consistency
- **Npm_Organization**: An npm organization (e.g., `pulsyflux`) that provides a namespace for scoped packages (e.g., `@pulsyflux/broker`), ensuring package name ownership and preventing Typosquatting within the scope
- **GitHub_Organization**: A GitHub organization (e.g., `pulsyflux`) that provides a namespace for repositories and Go modules (e.g., `github.com/pulsyflux/broker`), serving as the Go equivalent of npm's scoped packages for unified branding and module path ownership
- **Go_Checksum_Database**: The append-only, cryptographically-verifiable log at sum.golang.org that stores checksums for all public Go modules, ensuring every consumer of a given module version receives identical code
- **Go_Module_Proxy**: The default module mirror at proxy.golang.org that caches Go modules for performance and availability; used by the Go toolchain since Go 1.13
- **Govulncheck**: The official Go vulnerability scanning tool that uses static analysis to identify known vulnerabilities in functions actually called by the code, backed by the Go vulnerability database at vuln.go.dev
- **Typosquatting**: An attack where a malicious package is published with a name similar to a legitimate package, tricking developers into installing the malicious version
- **Minimum_Release_Age**: pnpm setting that delays installing newly published package versions by a configurable number of minutes, providing a safety window for malware detection before the version becomes installable
- **Trust_Policy**: pnpm setting that prevents installation of packages whose trust level has decreased compared to previous releases, catching scenarios where a previously CI/CD-published package is suddenly published with a regular token (potential account takeover)
- **Exotic_Subdep**: A transitive dependency that resolves to a non-registry source such as a git repository or direct tarball URL, which can be exploited to inject malicious code from untrusted sources
- **Frozen_Lockfile**: A CI installation mode (`--frozen-lockfile`) that installs exactly the versions recorded in the lockfile and fails if the lockfile is out of date, preventing unexpected dependency changes
- **CI_Workflow**: The unified GitHub Actions workflow file that orchestrates the full build, test, and publish pipeline from Go module compilation through Node.js addon build, testing, and npm publishing
- **Act**: An open-source CLI tool (`nektos/act`) that runs GitHub Actions workflows locally using Docker containers, enabling developers to test CI/CD pipelines without pushing to GitHub

## Requirements

### Requirement 1: Remove npm Lockfiles

**User Story:** As a developer, I want the npm lockfile removed from the Nodejs_Api_Package, so that the project no longer contains conflicting package manager artifacts.

#### Acceptance Criteria

1. WHEN the migration is performed, THE Migration_Tool SHALL delete `package-lock.json` from the Nodejs_Api_Package directory
2. WHEN the migration is complete, THE Nodejs_Api_Package directory SHALL NOT contain a `package-lock.json` file

### Requirement 2: Remove Existing node_modules

**User Story:** As a developer, I want the existing npm-managed `node_modules` directory removed, so that pnpm can perform a clean install without conflicts from npm's dependency tree layout.

#### Acceptance Criteria

1. WHEN the migration is performed, THE Migration_Tool SHALL delete the `node_modules/` directory from the Nodejs_Api_Package directory

### Requirement 3: Generate pnpm Lockfiles

**User Story:** As a developer, I want a pnpm lockfile generated for the Nodejs_Api_Package, so that dependency versions are pinned for reproducible installs using pnpm.

#### Acceptance Criteria

1. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL generate a `pnpm-lock.yaml` file in that directory
2. WHEN `pnpm install` completes in the Nodejs_Api_Package directory, THE pnpm tool SHALL create a `node_modules/` directory containing all resolved dependencies

### Requirement 4: Resolve All Dependencies

**User Story:** As a developer, I want all existing dependencies (including GitHub-hosted packages) to resolve correctly under pnpm, so that the project builds and runs without missing modules.

#### Acceptance Criteria

1. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve and install the `jasmine` devDependency
2. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve and install the `node-addon-api` dependency (regular dependency, not devDependency — see Requirement 32)
3. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve and install the `node-api-headers` dependency (regular dependency, not devDependency — see Requirement 32)
4. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve and install the `tsx` devDependency
5. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve and install the `zig-build` GitHub_Dependency from `github:solarwinds/zig-build` (regular dependency, not devDependency — see Requirement 32)

### Requirement 5: Integrate Native Build as Consumer-Side Lifecycle Scripts

**User Story:** As a package consumer, I want `pnpm install` to fetch the Go module, compile the Go shared library, and compile the C++ addon locally on my machine, so that I get native binaries built for my environment without relying on pre-built artifacts or bundled Go source from the npm registry.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` SHALL define a `postinstall` script that executes the full Build_Pipeline
2. WHEN a consumer runs `pnpm install` and the Nodejs_Api_Package `postinstall` Lifecycle_Script is permitted to execute, THE script SHALL first fetch the Go_Module by running `go get github.com/pulsyflux/broker` or by setting up a temporary Go workspace that depends on the Go_Module
3. WHEN a consumer runs `pnpm install` and the Nodejs_Api_Package `postinstall` Lifecycle_Script is permitted to execute, THE script SHALL compile the Go shared library from the fetched Go_Module source by running `go build -buildmode=c-shared -o .bin/release/broker_lib.dll` against the `broker_lib.go` cgo bindings obtained from the Go_Module
4. WHEN a consumer runs `pnpm install` and the Nodejs_Api_Package `postinstall` Lifecycle_Script is permitted to execute, THE script SHALL compile the C++ addon by running `node build.mjs`
5. THE `postinstall` Lifecycle_Script SHALL execute the Build_Pipeline steps in the following order: (a) fetch the Go_Module, (b) build the Go shared library, (c) build the C++ addon
6. WHEN the `postinstall` Lifecycle_Script completes on the consumer's machine, THE Build_Pipeline SHALL have produced `broker_lib.dll` in `.bin/release/`
7. WHEN the `postinstall` Lifecycle_Script completes on the consumer's machine, THE Build_Pipeline SHALL have produced `broker_addon.node` in `.bin/release/`
8. IF the Go_Module fetch fails on the consumer's machine, THEN THE `postinstall` Lifecycle_Script SHALL exit with a non-zero exit code and report the fetch error
9. IF the Go shared library compilation fails on the consumer's machine, THEN THE `postinstall` Lifecycle_Script SHALL exit with a non-zero exit code and report the compilation error
10. IF the C++ addon compilation fails on the consumer's machine, THEN THE `postinstall` Lifecycle_Script SHALL exit with a non-zero exit code and report the compilation error
11. THE `postinstall` Lifecycle_Script SHALL require Go and a C++ toolchain to be available on the consumer's machine as build prerequisites
12. THE `postinstall` Lifecycle_Script SHALL NOT require Go source files to be bundled in the Published_Package; all Go source SHALL be obtained from the Go_Module at install time
13. WHEN the `postinstall` Lifecycle_Script runs `go get` on the consumer's machine, THE Go toolchain SHALL verify module checksums against sum.golang.org by default. THE `postinstall` Lifecycle_Script SHALL NOT set `GONOSUMCHECK`, `GONOSUMDB`, or `GOFLAGS=-insecure` environment variables

### Requirement 6: Configure pnpm Lifecycle Script Security

**User Story:** As a developer, I want pnpm's lifecycle script blocking configured to allow only trusted packages to run install scripts, so that the native build pipeline executes during `pnpm install` while untrusted third-party packages remain blocked from running arbitrary code. This directly mitigates Shai-Hulud-style worm attacks that abuse preinstall/postinstall scripts to steal credentials and propagate across packages.

#### Acceptance Criteria

1. THE Nodejs_Api_Package SHALL include a pnpm configuration using the pnpm v10 `allowBuilds` setting (via `package.json` `pnpm.allowBuilds` or equivalent configuration) that defines a Script_Allowlist of packages permitted to run Lifecycle_Scripts — the deprecated `onlyBuiltDependencies` setting SHALL NOT be used
2. THE pnpm configuration SHALL set `strictDepBuilds` to `true` to enforce that only explicitly allowlisted packages can run Lifecycle_Scripts, with no implicit fallback
3. THE Script_Allowlist SHALL include the Nodejs_Api_Package itself (to allow its own `postinstall` script to run)
4. THE Script_Allowlist SHALL NOT include packages that do not require Lifecycle_Scripts for correct operation
5. WHEN `pnpm install` is executed, THE pnpm tool SHALL block Lifecycle_Scripts for all packages not listed in the Script_Allowlist
6. WHEN `pnpm install` is executed, THE pnpm tool SHALL permit Lifecycle_Scripts for all packages listed in the Script_Allowlist
7. IF a new dependency requires Lifecycle_Scripts, THEN THE developer SHALL explicitly add that dependency to the Script_Allowlist before the scripts will execute
8. THE pnpm configuration SHALL NOT set `dangerouslyAllowAllBuilds` to `true`, as this disables all Lifecycle_Script blocking and negates the security benefit

### Requirement 7: Preserve Build Pipeline Compatibility

**User Story:** As a developer, I want the native addon build pipeline to work identically after migration, so that `node build.mjs` continues to compile the C++ addon using zig-build.

#### Acceptance Criteria

1. WHEN `node build.mjs` is executed in the Nodejs_Api_Package directory after pnpm install, THE Build_Pipeline SHALL compile the C++ addon and produce `broker_addon.node` in `.bin/release/`
2. WHEN `pnpm exec jasmine` is executed in the Nodejs_Api_Package directory, THE Nodejs_Api_Package SHALL run the Jasmine test suite from `spec/`

### Requirement 8: Update package.json for Migration

**User Story:** As a developer, I want the `package.json` files updated for the migration (lifecycle scripts, engine floor, dependency pinning, files field, pnpm enforcement), so that the published package contains only source files and enforces pnpm-only installation.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` SHALL retain all existing dependency declarations and metadata not explicitly modified by other requirements in this document
2. THE Nodejs_Api_Package `package.json` `main` field SHALL point to `.bin/release/registry.mjs`, and the `exports` field SHALL point to `./.bin/release/registry.mjs` for the `"import"` condition — this path references a file that is produced by the postinstall Build_Pipeline (Requirement 5) and does NOT exist in the Published_Package tarball; it only exists after the postinstall Lifecycle_Script completes successfully on the consumer's machine
3. THE Nodejs_Api_Package `package.json` SHALL contain the `postinstall` script defined in Requirement 5
4. THE Nodejs_Api_Package `package.json` SHALL contain the `preinstall` Preinstall_Guard script defined in Requirement 17
5. THE Nodejs_Api_Package `package.json` SHALL contain the updated `engines` field defined in Requirement 14
6. THE Nodejs_Api_Package `package.json` SHALL contain the pinned `zig-build` dependency defined in Requirement 13
7. THE Nodejs_Api_Package `package.json` SHALL contain the updated `files` field defined in Requirement 18
8. THE Nodejs_Api_Package `package.json` SHALL contain the `publishConfig` field defined in Requirement 30 AC#6
9. THE Nodejs_Api_Package `package.json` SHALL contain the dependency promotions defined in Requirement 32
10. THE Nodejs_Api_Package `package.json` SHALL include a `packageManager` field specifying the pnpm version (e.g., `pnpm@<version>`) to signal the intended package manager

### Requirement 9: Update .gitignore for pnpm

**User Story:** As a developer, I want `.gitignore` updated to account for pnpm-specific files, so that generated artifacts are properly excluded from version control.

#### Acceptance Criteria

1. WHEN the migration is complete, THE Nodejs_Api_Package `.gitignore` SHALL exclude `node_modules/` from version control
2. WHEN the migration is complete, THE Nodejs_Api_Package `.gitignore` SHALL NOT list `package-lock.json` as an ignored file
3. WHEN the migration is complete, THE Nodejs_Api_Package `.gitignore` SHALL NOT exclude `pnpm-lock.yaml` from version control — the lockfile must be committed per Requirement 28 AC#3

### Requirement 10: Handle pnpm Strict Node_Modules Structure

**User Story:** As a developer, I want the pnpm `node_modules` structure to be compatible with the native addon build, so that `node-addon-api` and `node-api-headers` are accessible to the C++ compilation step.

#### Acceptance Criteria

1. WHEN `pnpm install` completes in the Nodejs_Api_Package directory, THE `node-addon-api` package SHALL be accessible via `require.resolve('node-addon-api')` or equivalent path resolution
2. WHEN `pnpm install` completes in the Nodejs_Api_Package directory, THE `node-api-headers` package SHALL be accessible via `require.resolve('node-api-headers')` or equivalent path resolution
3. IF `node-addon-api` or `node-api-headers` are not resolvable due to pnpm strict mode, THEN THE Migration_Tool SHALL configure a `.npmrc` file with appropriate hoisting settings to make the packages accessible

### Requirement 11: Enforce Strict Dependency Isolation

**User Story:** As a developer, I want pnpm's strict dependency isolation enforced by default, so that phantom dependencies are prevented and the project's supply-chain attack surface is minimized.

#### Acceptance Criteria

1. WHEN `pnpm install` completes in the Nodejs_Api_Package directory, THE Node_Modules structure SHALL only allow the Nodejs_Api_Package to import packages declared in its own `package.json`
2. IF a `.npmrc` file is created for hoisting overrides, THEN THE `.npmrc` file SHALL limit hoisting to only the specific packages required by the Build_Pipeline (e.g., `node-addon-api`, `node-api-headers`)
3. THE `.npmrc` file SHALL NOT set `shamefully-hoist=true` or `node-linker=hoisted`, as these settings disable Strict_Isolation

### Requirement 12: Verify Package Integrity via Content-Addressable Storage

**User Story:** As a developer, I want installed packages verified through pnpm's content-addressable store, so that tampered or corrupted package files are detected before they enter the project.

#### Acceptance Criteria

1. WHEN `pnpm install` is executed, THE pnpm tool SHALL store package files in the Content_Addressable_Store indexed by content hash
2. WHEN a package file is linked from the Content_Addressable_Store into Node_Modules, THE pnpm tool SHALL verify that the file content matches its stored hash
3. IF a package file fails integrity verification, THEN THE pnpm tool SHALL reject the install and report the integrity mismatch

### Requirement 13: Pin GitHub Dependency to Immutable Reference

**User Story:** As a developer, I want the `zig-build` GitHub dependency pinned to a specific commit hash or tag, so that the resolved code cannot change between installs and a compromised or force-pushed branch cannot silently alter the dependency.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` SHALL specify the `zig-build` dependency with a pinned commit hash or tag (e.g., `github:solarwinds/zig-build#<commit-sha>` or `github:solarwinds/zig-build#<tag>`)
2. THE Nodejs_Api_Package `package.json` SHALL NOT specify the `zig-build` dependency as an unpinned branch reference (e.g., `github:solarwinds/zig-build` without a commit or tag qualifier)
3. WHEN `pnpm install` is executed in the Nodejs_Api_Package directory, THE pnpm tool SHALL resolve `zig-build` to the exact commit or tag specified in the pinned reference
4. IF the pinned commit or tag is not found in the `solarwinds/zig-build` repository, THEN THE pnpm tool SHALL fail the install and report the unresolvable reference

### Requirement 14: Enforce Node.js Engine Floor

**User Story:** As a developer, I want the `engines` field in `package.json` updated to require Node.js v24, so that the package is not installed or used on older runtimes with known security vulnerabilities.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` `engines.node` field SHALL specify `>=24.0.0` as the minimum required Node.js version
2. THE Nodejs_Api_Package `package.json` `engines.node` field SHALL NOT specify a minimum version below `24.0.0`
3. WHEN `pnpm install` is executed with a Node.js version below `24.0.0`, THE pnpm tool SHALL warn or reject the install based on the `engine-strict` configuration

### Requirement 15: Harden DLL Loading Path

**User Story:** As a developer, I want the C++ addon to load `broker_lib.dll` using a full absolute path derived from the addon's own location, so that the Windows DLL search order cannot be exploited to load a malicious DLL from the current working directory or other search paths.

#### Acceptance Criteria

1. THE C++ addon (`addon.cc`) SHALL resolve the absolute path of `broker_lib.dll` relative to the directory containing `broker_addon.node` before calling `LoadLibraryA`
2. THE C++ addon SHALL NOT call `LoadLibraryA` with a bare filename (e.g., `LoadLibraryA("broker_lib.dll")`) as the first loading attempt
3. WHEN the C++ addon loads `broker_lib.dll`, THE addon SHALL use the full absolute path as the sole argument to `LoadLibraryA`
4. IF `broker_lib.dll` is not found at the expected absolute path, THEN THE C++ addon SHALL throw a descriptive error indicating the expected path and SHALL NOT fall back to the Windows default DLL search order

### Requirement 16: Publish with Provenance Attestation

> **Note:** Provenance publishing is executed by the unified CI_Workflow's publish stage (Requirement 34, Stage 3).

**User Story:** As a maintainer, I want the `@pulsyflux/broker` package published with npm provenance attestation from CI, so that consumers can cryptographically verify that a published version was built from a specific source commit in a trusted CI environment.

#### Acceptance Criteria

1. WHEN the Nodejs_Api_Package is published to the npm registry from CI, THE publish command SHALL include the `--provenance` flag (e.g., `npm publish --provenance` or `pnpm publish --provenance`)
2. WHEN the Nodejs_Api_Package is published with provenance, THE npm registry SHALL attach a Provenance_Attestation linking the package version to the source commit and CI workflow
3. THE CI workflow SHALL run in a supported environment for npm provenance (GitHub Actions, GitLab CI, or another OIDC-capable CI provider)
4. THE Nodejs_Api_Package `package.json` SHALL include a `repository` field pointing to the source repository under the `pulsyflux` GitHub_Organization (e.g., `https://github.com/pulsyflux/broker.git`)

### Requirement 17: Enforce pnpm-Only Installation

**User Story:** As a maintainer, I want `npm install` to be rejected when consumers attempt to install the package, so that only pnpm is used and the strict dependency isolation guarantees are not bypassed.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` SHALL define a `preinstall` script that acts as a Preinstall_Guard to reject non-pnpm package managers
2. WHEN a consumer runs `npm install` in a project that depends on the Nodejs_Api_Package, THE Preinstall_Guard SHALL detect that the package manager is not pnpm by inspecting the `npm_config_user_agent` environment variable or using an equivalent detection mechanism (e.g., the `only-allow` package)
3. WHEN the Preinstall_Guard detects a non-pnpm package manager, THE Preinstall_Guard SHALL exit with a non-zero exit code and print a descriptive error message instructing the user to use pnpm
4. WHEN a consumer runs `pnpm install`, THE Preinstall_Guard SHALL allow installation to proceed without error
5. THE Nodejs_Api_Package `package.json` SHALL include a `packageManager` field specifying the pnpm version (e.g., `pnpm@<version>`) to provide an additional signal to Node.js corepack and tooling that pnpm is the intended package manager
6. NOTE: The Preinstall_Guard runs as a Lifecycle_Script. When a consumer uses npm (which does NOT block lifecycle scripts by default), the Preinstall_Guard executes and rejects the install. When a consumer uses pnpm, the Preinstall_Guard only runs if the package is in the Script_Allowlist (Requirement 6) — but since pnpm is the allowed package manager, the guard passes. If the consumer uses pnpm but has NOT allowlisted the package, the Preinstall_Guard will not run, but this is acceptable because pnpm is the correct package manager

### Requirement 18: Exclude Pre-Built Binaries and Go Source from Published Package

**User Story:** As a maintainer, I want the published npm package to contain only the C++ source, build script, registry module, types, and documentation, so that no pre-built binaries or Go source files are shipped through the npm registry — consumers fetch Go source from the Go_Module and build everything locally.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` `files` field SHALL NOT include `.bin/release/` or any path that would cause `broker_lib.dll`, `broker_addon.node`, or other pre-built binary artifacts to be included in the Published_Package
2. THE Nodejs_Api_Package `package.json` `files` field SHALL NOT include `broker_lib.go` — the Go source SHALL be fetched from the Go_Module during the `postinstall` Lifecycle_Script, not bundled in the Published_Package
3. THE Nodejs_Api_Package `package.json` `files` field SHALL include `addon.cc` (C++ source for the native addon)
4. THE Nodejs_Api_Package `package.json` `files` field SHALL include `build.mjs` (zig-build compilation script)
5. THE Nodejs_Api_Package `package.json` `files` field SHALL include `registry.mjs` (ES module wrapper)
6. THE Nodejs_Api_Package `package.json` `files` field SHALL include `types/` (TypeScript declarations)
7. THE Nodejs_Api_Package `package.json` `files` field SHALL include `README.md` and `LICENSE`
8. WHEN `pnpm pack` or `pnpm publish` is executed, THE resulting tarball SHALL contain only the files listed in the `files` field and `package.json` itself
9. WHEN `pnpm pack` or `pnpm publish` is executed, THE resulting tarball SHALL NOT contain `broker_lib.dll`, `broker_addon.node`, `broker_lib.go`, or any other compiled binary artifact or Go source file

### Requirement 19: Publish Go Code as a Go Module

**User Story:** As a maintainer, I want the Go code (broker package and cgo bindings) published as a proper Go module at `github.com/pulsyflux/broker` under the `pulsyflux` GitHub_Organization, so that the `postinstall` lifecycle script can fetch the Go source via `go get` and consumers do not need Go source bundled in the npm package.

#### Acceptance Criteria

1. THE Go_Module SHALL be published at `github.com/pulsyflux/broker` with a `go.mod` file declaring the canonical module path `module github.com/pulsyflux/broker` at the repository root (the current `go.mod` declares `module pulsyflux` which is a local/non-canonical path and must be updated to the full GitHub_Organization path for proper Go module publishing and checksum database compatibility)
2. THE Go_Module SHALL contain `broker_lib.go` at the root of the repository (alongside `go.mod`), as it declares `package main` and contains cgo exports compiled with `go build -buildmode=c-shared`. The `broker/` package (`server.go`, `client.go`) SHALL be in a `broker/` subdirectory. After the module path changes from `pulsyflux` to `github.com/pulsyflux/broker`, `broker_lib.go` SHALL import the broker subpackage as `github.com/pulsyflux/broker/broker`
3. WHEN a new version of the Go_Module is released, THE maintainer SHALL create a Git tag following Go module versioning conventions (e.g., `v1.0.0`, `v1.1.0`) so that `go get` can resolve the module at a specific version
4. WHEN `go get github.com/pulsyflux/broker@<version>` is executed, THE Go toolchain SHALL fetch the Go_Module source including the `broker/` package and `broker_lib.go`
5. THE Go_Module `go.mod` file SHALL declare all required dependencies (e.g., `github.com/google/uuid`) so that `go get` transitively resolves them
6. IF the Go_Module is not reachable at `github.com/pulsyflux/broker` or the requested version tag does not exist, THEN THE `go get` command SHALL fail with a descriptive error

### Requirement 20: Enforce Go Module Checksum Verification

**User Story:** As a developer, I want the Go_Module and all its dependencies verified through Go's checksum database (sum.golang.org), so that tampered or substituted module code is detected before it enters the build.

#### Acceptance Criteria

1. THE Go_Module repository SHALL contain a `go.sum` file with cryptographic hashes for all dependencies
2. THE `go.sum` file SHALL be committed to version control
3. THE `GONOSUMCHECK` environment variable SHALL NOT be set during builds or in CI
4. THE `GONOSUMDB` environment variable SHALL NOT be set for public dependencies
5. THE `GOFLAGS` environment variable SHALL NOT contain `-insecure` during builds
6. WHEN `go get` or `go mod download` is executed, THE Go toolchain SHALL verify module checksums against the Go_Checksum_Database at sum.golang.org

### Requirement 21: Pin Go Module Version in Build Pipeline

**User Story:** As a developer, I want the postinstall lifecycle script to fetch a specific tagged version of the Go_Module rather than the latest or an untagged commit, so that new malicious versions cannot automatically propagate into consumer builds.

#### Acceptance Criteria

1. THE postinstall Lifecycle_Script SHALL fetch the Go_Module at a specific version tag (e.g., `go get github.com/pulsyflux/broker@v1.0.0`)
2. THE postinstall Lifecycle_Script SHALL NOT fetch the Go_Module at `@latest` without a pinned version
3. THE Go_Module version used by the postinstall Lifecycle_Script SHALL be configurable (e.g., stored in a version file or `package.json` metadata) so that the version can be updated deliberately
4. WHEN the Go_Module version is updated, THE change SHALL be visible in version control for code review

### Requirement 22: Run Go Vulnerability Scanning

> **Note:** Govulncheck is run as part of the unified CI_Workflow's Go stage (Requirement 34, Stage 1).

**User Story:** As a developer, I want the Go_Module scanned for known vulnerabilities using Govulncheck as part of the CI pipeline, so that vulnerable dependencies are detected before a new version is published.

#### Acceptance Criteria

1. THE CI pipeline SHALL run `govulncheck ./...` against the Go_Module source
2. IF Govulncheck reports vulnerabilities in functions actually called by the code, THEN THE CI pipeline SHALL fail the build
3. THE Govulncheck tool SHALL use the official Go vulnerability database at vuln.go.dev
4. WHEN a new version of the Go_Module is being prepared for release, THE vulnerability scan SHALL run before the version tag is created

### Requirement 23: Protect Against Go Module Typosquatting

**User Story:** As a maintainer, I want the project to take measures against Typosquatting attacks on the Go_Module, so that consumers are protected from malicious packages impersonating the legitimate module (as documented in the BoltDB and disk-wiping malware attacks in the Go ecosystem).

#### Acceptance Criteria

1. THE Go_Module SHALL use the canonical module path `github.com/pulsyflux/broker` matching the repository URL under the `pulsyflux` GitHub_Organization
2. THE postinstall Lifecycle_Script SHALL hardcode the exact Go_Module path and SHALL NOT accept user-configurable module paths
3. THE Go_Module README SHALL document the canonical import path to help consumers verify they are using the correct module
4. THE Go_Module `go.mod` SHALL declare the module path as `module github.com/pulsyflux/broker` (the current declaration of `module pulsyflux` is a local/non-canonical path that must be updated to the full GitHub_Organization path for proper Go module publishing)

### Requirement 24: Configure pnpm Minimum Release Age

**User Story:** As a developer, I want pnpm configured to delay installing newly published package versions by at least 24 hours, so that the project is protected from the window between malware publication and detection — as demonstrated by the Axios hijack (March 2026) where malicious versions were live for ~3 hours before removal, and the Shai-Hulud worm which spread across 796+ packages before detection.

#### Acceptance Criteria

1. THE pnpm configuration SHALL set `minimumReleaseAge` to at least `1440` (24 hours in minutes) to delay installing newly published versions, providing a safety window for malware detection before the version becomes installable
2. WHERE the project's own packages (e.g., `@pulsyflux/broker`) need to be installed immediately during development, THE `minimumReleaseAgeExclude` setting MAY exclude those packages from the Minimum_Release_Age check
3. THE Minimum_Release_Age setting SHALL be configured in the pnpm workspace configuration or `.npmrc` file, and SHALL apply to the Nodejs_Api_Package (either via workspace-level configuration or a per-package `.npmrc` file)

### Requirement 25: Block Exotic Transitive Dependencies

**User Story:** As a developer, I want pnpm configured to block transitive dependencies from resolving to non-registry sources (git repos, tarball URLs), so that the Axios-style attack pattern — where a phantom dependency pointing to a malicious external source was injected into a hijacked package — is prevented for all transitive dependencies.

#### Acceptance Criteria

1. THE pnpm configuration SHALL set `blockExoticSubdeps` to `true`
2. WHEN `pnpm install` is executed, THE pnpm tool SHALL reject any transitive dependency that resolves to a git repository, direct tarball URL, or other non-registry source (Exotic_Subdep)
3. THE pnpm configuration SHALL permit direct dependencies explicitly declared in the Nodejs_Api_Package `package.json` to use exotic sources (e.g., the `zig-build` GitHub_Dependency), as `blockExoticSubdeps` applies only to transitive dependencies
4. THE `blockExoticSubdeps` setting SHALL be configured for the Nodejs_Api_Package (either via workspace-level configuration or a per-package `.npmrc` file)

### Requirement 26: Enforce Trust Policy

**User Story:** As a developer, I want pnpm configured to reject packages whose trust level has decreased compared to previous releases, so that account takeover scenarios — where an attacker publishes from a compromised account that lacks the original CI/CD provenance — are detected and blocked before installation.

#### Acceptance Criteria

1. THE pnpm configuration SHALL set `trustPolicy` to `no-downgrade`
2. WHEN a package version is published with a lower trust level than previous versions (e.g., previously published via trusted CI/CD with Provenance_Attestation but now published with a regular token), THE pnpm tool SHALL reject the installation
3. WHERE specific known-safe packages do not meet trust requirements due to their publishing workflow, THE `trustPolicyExclude` setting MAY be used to exclude those packages from the Trust_Policy check
4. THE `trustPolicy` setting SHALL be configured for the Nodejs_Api_Package (either via workspace-level configuration or a per-package `.npmrc` file)

### Requirement 27: Pin Exact Dependency Versions

**User Story:** As a developer, I want all dependency versions saved as exact versions rather than semver ranges, so that unexpected new versions — which could be compromised as in the chalk/debug incident (2025) where stolen credentials were used to publish malicious updates — are not automatically pulled in.

#### Acceptance Criteria

1. THE `.npmrc` file SHALL set `save-exact=true` to ensure new dependencies are saved with exact versions
2. THE `.npmrc` file SHALL set `save-prefix=''` (empty string) to prevent semver range prefixes from being added
3. WHEN new dependencies are added via `pnpm add`, THE version SHALL be saved as an exact version (e.g., `1.2.3`) and SHALL NOT be saved as a semver range (e.g., `^1.2.3` or `~1.2.3`)
4. THE `save-exact` and `save-prefix` settings SHALL be configured for the Nodejs_Api_Package (either via workspace-level configuration or a per-package `.npmrc` file)

### Requirement 28: Enforce Frozen Lockfile in CI

> **Note:** Frozen lockfile is enforced by the unified CI_Workflow's Node.js stage (Requirement 34, Stage 2).

**User Story:** As a developer, I want CI builds to install exactly the dependency versions recorded in the lockfile, so that malicious versions published between lockfile generation and CI execution cannot be pulled in — ensuring reproducible, tamper-resistant builds.

#### Acceptance Criteria

1. THE CI pipeline SHALL run `pnpm install --frozen-lockfile` instead of `pnpm install` to enforce Frozen_Lockfile mode
2. IF the Lockfile (`pnpm-lock.yaml`) is out of date or missing, THEN THE CI install SHALL fail rather than generating a new lockfile
3. THE Lockfile (`pnpm-lock.yaml`) SHALL be committed to version control so that CI can use it for Frozen_Lockfile installs

### Requirement 29: Run npm Audit for Known Vulnerabilities

> **Note:** pnpm audit is run as part of the unified CI_Workflow's Node.js stage (Requirement 34, Stage 2).

**User Story:** As a developer, I want the CI pipeline to scan all dependencies for known vulnerabilities, so that packages with disclosed security issues are detected before deployment — complementing the proactive supply-chain defenses (Minimum_Release_Age, Trust_Policy, Exotic_Subdep blocking) with reactive vulnerability detection.

#### Acceptance Criteria

1. THE CI pipeline SHALL run `pnpm audit` to check for known vulnerabilities in dependencies
2. IF `pnpm audit` reports vulnerabilities at severity level `high` or `critical`, THEN THE CI pipeline SHALL fail the build
3. THE audit SHALL check both direct and transitive dependencies

### Requirement 30: Establish Unified Package Namespace

**User Story:** As a maintainer, I want both the npm package and Go module published under a unified `pulsyflux` namespace (npm organization and GitHub organization), so that all official packages are consistently branded, discoverable, and protected from typosquatting within the scope.

#### Acceptance Criteria

1. WHEN the migration is performed, THE maintainer SHALL create an npm organization named `pulsyflux` on the npm registry to serve as the Npm_Organization scope for all official npm packages
2. WHEN the migration is performed, THE maintainer SHALL create a GitHub organization named `pulsyflux` on GitHub to serve as the GitHub_Organization namespace for all official repositories and Go modules
3. THE Nodejs_Api_Package SHALL be published to the npm registry as `@pulsyflux/broker` (scoped under the `pulsyflux` Npm_Organization)
4. THE Go_Module SHALL be published under the `pulsyflux` GitHub_Organization at `github.com/pulsyflux/broker`
5. THE Nodejs_Api_Package `package.json` `name` field SHALL be changed from `pulsyflux-broker` to `@pulsyflux/broker`
6. THE Nodejs_Api_Package `package.json` SHALL include a `publishConfig` object with `access` set to `"public"`, because scoped packages default to private on the npm registry
7. THE Go_Module `go.mod` module path SHALL be updated from `pulsyflux` to `github.com/pulsyflux/broker` to reflect the new GitHub_Organization path
8. THE repository SHALL be transferred or recreated under the `pulsyflux` GitHub_Organization (e.g., from `github.com/marchuanv/pulsyflux` to `github.com/pulsyflux/broker`)
9. THE Nodejs_Api_Package `package.json` `repository.url` field SHALL be updated to `https://github.com/pulsyflux/broker.git`
10. THE Nodejs_Api_Package `package.json` `homepage` field SHALL be updated to `https://github.com/pulsyflux/broker#readme`
11. THE Nodejs_Api_Package `package.json` `bugs.url` field SHALL be updated to `https://github.com/pulsyflux/broker/issues`
12. AFTER the repository is transferred from `github.com/marchuanv/pulsyflux` to `github.com/pulsyflux/broker`, THE old repository SHALL include a redirect or deprecation notice. THE old Go module path (`pulsyflux`) SHALL NOT be used in any new code. NOTE: The Go_Module_Proxy may have cached the old module path — consumers using the old path will continue to receive the old cached version, which is acceptable as long as no new versions are published under the old path

### Requirement 31: Enforce Go Version Floor

**User Story:** As a package consumer, I want the postinstall script to verify that my Go installation meets the minimum version required by the Go_Module before attempting compilation, so that I get a clear error message instead of a cryptic build failure.

#### Acceptance Criteria

1. THE Go_Module `go.mod` SHALL declare the minimum Go version required (currently `go 1.25`)
2. WHEN the `postinstall` Lifecycle_Script executes on the consumer's machine, THE script SHALL verify that the consumer's Go installation meets the minimum version declared in the Go_Module `go.mod` before attempting compilation
3. IF the consumer's Go version is below the minimum required version, THEN THE `postinstall` Lifecycle_Script SHALL exit with a non-zero exit code and a descriptive error message stating the required minimum Go version and the consumer's current Go version

### Requirement 32: Promote Build Dependencies to Regular Dependencies

**User Story:** As a package consumer, I want the build-time dependencies (`node-addon-api`, `node-api-headers`, `zig-build`) available when I run `pnpm install`, so that the postinstall Build_Pipeline can compile the C++ addon on my machine — devDependencies are NOT installed when a consumer installs a package, so these must be regular dependencies.

#### Acceptance Criteria

1. THE Nodejs_Api_Package `package.json` SHALL list `node-addon-api` as a regular dependency (not devDependency), because it is required by the consumer-side postinstall Build_Pipeline to compile the C++ addon
2. THE Nodejs_Api_Package `package.json` SHALL list `node-api-headers` as a regular dependency (not devDependency), because it is required by the consumer-side postinstall Build_Pipeline to compile the C++ addon
3. THE Nodejs_Api_Package `package.json` SHALL list `zig-build` as a regular dependency (not devDependency), because it is required by the consumer-side postinstall Build_Pipeline to compile the C++ addon via `node build.mjs`
4. THE Nodejs_Api_Package `package.json` SHALL keep `jasmine` and `tsx` as devDependencies, as they are only needed for development and testing and are not required by the consumer-side Build_Pipeline

### Requirement 33: Enforce npm Organization 2FA

**User Story:** As a maintainer, I want two-factor authentication enforced for all members of the `pulsyflux` npm organization, so that compromised credentials alone cannot be used to publish malicious package versions — as demonstrated by the Axios hijack (March 2026) and chalk/debug compromise (2025) where stolen credentials were the attack vector.

#### Acceptance Criteria

1. THE `pulsyflux` Npm_Organization SHALL require two-factor authentication (2FA) for all members
2. THE `pulsyflux` Npm_Organization SHALL require 2FA for publishing packages
3. No member of the Npm_Organization SHALL be permitted to publish packages without 2FA enabled on their npm account

### Requirement 34: Unified GitHub Actions CI/CD Workflow

**User Story:** As a developer, I want a single GitHub Actions workflow file that orchestrates the entire build, test, and publish pipeline from Go module compilation through Node.js addon build, testing, and npm publishing, so that the CI/CD process is centralized, ordered correctly (Go before Node.js), and easy to maintain — replacing the scattered CI references across multiple requirements with one authoritative pipeline definition.

#### Acceptance Criteria

1. THE repository SHALL contain a single GitHub Actions CI_Workflow file (e.g., `.github/workflows/ci.yml`) that orchestrates the full CI/CD pipeline
2. THE CI_Workflow SHALL execute Go stages before Node.js stages, because the Node.js Build_Pipeline depends on the Go shared library produced during the Go stages
3. THE CI_Workflow SHALL run on push to the main branch and on pull requests targeting the main branch
4. THE CI_Workflow SHALL run the following Go_Module stages in order: checkout code, set up Go, run `go mod download` and verify checksums, run `go vet ./...` for static analysis, run `go test ./...` for Go unit tests (broker package), run `govulncheck ./...` for vulnerability scanning (Requirement 22), and compile the Go shared library via `go build -buildmode=c-shared -o broker_lib.dll broker_lib.go`
5. THE CI_Workflow SHALL run the following Node.js stages in order: set up Node.js v24 (Requirement 14), set up pnpm using the version specified in the `packageManager` field of `package.json` (Requirement 17 AC#5), run `pnpm install --frozen-lockfile` (Requirement 28), run `node build.mjs` to compile the C++ addon, run `pnpm exec jasmine` to execute the Node.js test suite, and run `pnpm audit` (Requirement 29)
6. THE CI_Workflow SHALL include an integration test stage that runs the integration tests from the Integration_Test_Directory (Requirement 36) after the Go and Node.js build and unit test stages pass
7. THE CI_Workflow SHALL include a publish stage that runs `pnpm publish --provenance` (Requirement 16) only on version tag pushes matching the pattern `v*.*.*`
8. THE publish stage SHALL only execute if all previous stages (Go, Node.js, and integration tests) pass
9. THE CI_Workflow SHALL use the Go version declared in the Go_Module `go.mod` file (currently Go 1.25)
10. THE CI_Workflow SHALL use Node.js v24 as specified in Requirement 14
11. THE CI_Workflow SHALL set up pnpm using the version specified in the `packageManager` field of the Nodejs_Api_Package `package.json`
12. IF any stage in the CI_Workflow fails, THEN THE CI_Workflow SHALL stop and report the failure without proceeding to subsequent stages
13. THE CI_Workflow SHALL NOT set `GONOSUMCHECK`, `GONOSUMDB`, or `GOFLAGS=-insecure` environment variables (Requirement 20)

### Requirement 35: Local Workflow Testing with Act

**User Story:** As a developer, I want to test the GitHub Actions CI_Workflow locally without pushing to GitHub, using Act (`nektos/act`) which runs workflows in Docker containers that simulate the GitHub Actions runner environment, so that I can iterate on workflow changes quickly and catch CI issues before they reach the remote repository.

#### Acceptance Criteria

1. THE repository SHALL include documentation (in README or a CONTRIBUTING guide) on how to test the GitHub Actions CI_Workflow locally using Act
2. THE CI_Workflow SHALL be compatible with Act for local execution — the CI_Workflow SHALL NOT use GitHub-specific features that are unavailable in Act, or SHALL provide fallbacks where such features are used
3. WHERE the project uses Act for local testing, THE repository MAY include an `.actrc` file to configure default Act settings (e.g., default runner image, platform mappings)
4. WHERE the project uses Act for local testing, THE repository MAY include a `.secrets.example` file documenting which secrets are needed for local workflow testing, without containing actual secret values
5. THE local workflow test SHALL be executable with a single command (e.g., `act push` or `act pull_request`)
6. THE CI_Workflow SHALL support skipping the publish stage during local testing (e.g., via an environment variable check or by not providing npm credentials), so that developers can test the build and test stages without triggering a publish

### Requirement 36: Integration Test Directory

**User Story:** As a developer, I want a dedicated integration test directory under `pulsyflux/test/` that houses CI pipeline validation tests and package installation verification tests for both Go and Node.js consumers, so that I can verify the full build pipeline, published packages, and CI workflow stages work correctly — both locally and in CI.

#### Acceptance Criteria

1. THE repository SHALL contain an Integration_Test_Directory at `pulsyflux/test/` for integration tests
2. THE Integration_Test_Directory SHALL contain Go package installation verification tests that: (a) create a temporary Go module, (b) run `go get github.com/pulsyflux/broker@<version>` to fetch the published Go_Module, (c) import and use the broker package to verify the Go_Module works as an external dependency, and (d) verify that `go build -buildmode=c-shared` produces a valid shared library from the fetched source
3. THE Integration_Test_Directory SHALL contain Node.js package installation verification tests that: (a) create a temporary directory with a minimal `package.json`, (b) run `pnpm install @pulsyflux/broker` to install the published Nodejs_Api_Package, (c) verify the postinstall Build_Pipeline executes and produces `broker_lib.dll` and `broker_addon.node`, and (d) import `@pulsyflux/broker` and verify the Server and Client classes are functional
4. THE Integration_Test_Directory SHALL contain CI pipeline validation tests that verify each stage of the CI_Workflow (Requirement 34) produces the expected outputs
5. THE integration tests SHALL be runnable locally via a single command (e.g., `pnpm test:integration` or a shell script)
6. THE CI_Workflow (Requirement 34) SHALL include a stage that runs the integration tests after the build and unit test stages pass (see Requirement 34 AC#6)
