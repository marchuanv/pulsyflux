/**
 * postinstall.mjs — Consumer-side build pipeline for @pulsyflux/broker.
 *
 * Runs during `pnpm install` to fetch the Go module, compile the Go shared
 * library and C++ addon locally on the consumer's machine.
 *
 * Steps:
 *   1. Check Go version >= 1.25
 *   2. Fetch Go module via `go get`
 *   3. Build Go shared library (.dll)
 *   4. Build C++ addon (.node)
 *   5. Verify output artifacts exist
 */

import { execSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const RELEASE_DIR = join(__dirname, '.bin', 'release');
const MIN_GO_VERSION = '1.25.0';

/**
 * Compare two semver-style version strings (e.g. "1.25.3" vs "1.25.0").
 * Returns  1 if a > b, -1 if a < b, 0 if equal.
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */
export function compareVersions(a, b) {
  const partsA = a.split('.').map(Number);
  const partsB = b.split('.').map(Number);
  const len = Math.max(partsA.length, partsB.length);

  for (let i = 0; i < len; i++) {
    const numA = partsA[i] ?? 0;
    const numB = partsB[i] ?? 0;
    if (numA > numB) return 1;
    if (numA < numB) return -1;
  }
  return 0;
}

/**
 * Parse `go version` output and check it meets the minimum.
 * Accepts strings like "go version go1.25.3 windows/amd64".
 * @param {string} versionOutput — raw output from `go version`
 * @param {string} [minimum]     — minimum version (default: 1.25.0)
 * @returns {boolean}
 */
export function checkGoVersion(versionOutput, minimum = MIN_GO_VERSION) {
  const match = /go(\d+\.\d+(?:\.\d+)?)/.exec(versionOutput);
  if (!match) return false;
  return compareVersions(match[1], minimum) >= 0;
}

// ---------------------------------------------------------------------------
// Pipeline helpers
// ---------------------------------------------------------------------------

function run(cmd, label) {
  console.log(`[postinstall] ${label}...`);
  try {
    execSync(cmd, { cwd: __dirname, stdio: 'inherit' });
  } catch (err) {
    console.error(`[postinstall] FAILED: ${label}`);
    console.error(err.message);
    process.exit(1);
  }
}

function readGoModuleVersion() {
  const pkgPath = join(__dirname, 'package.json');
  const pkg = JSON.parse(readFileSync(pkgPath, 'utf-8'));
  const version = pkg.goModuleVersion;
  if (!version) {
    console.error('[postinstall] FAILED: "goModuleVersion" field missing from package.json');
    process.exit(1);
  }
  return version;
}

// ---------------------------------------------------------------------------
// Main pipeline
// ---------------------------------------------------------------------------

function main() {
  // 1. Check Go version
  let goVersionOutput;
  try {
    goVersionOutput = execSync('go version', { encoding: 'utf-8' }).trim();
  } catch {
    console.error('[postinstall] FAILED: Go is not installed or not in PATH');
    process.exit(1);
  }

  if (!checkGoVersion(goVersionOutput)) {
    console.error(
      `[postinstall] FAILED: Go >= ${MIN_GO_VERSION} required, got: ${goVersionOutput}`
    );
    process.exit(1);
  }
  console.log(`[postinstall] Go version OK: ${goVersionOutput}`);

  // Ensure output directory exists
  if (!existsSync(RELEASE_DIR)) {
    mkdirSync(RELEASE_DIR, { recursive: true });
  }

  // 2. Fetch Go module
  const goModVersion = readGoModuleVersion();
  run(
    `go get github.com/pulsyflux/broker@${goModVersion}`,
    `Fetching Go module github.com/pulsyflux/broker@${goModVersion}`
  );

  // 3. Build Go shared library
  run(
    `go build -buildmode=c-shared -o .bin/release/broker_lib.dll`,
    'Building Go shared library'
  );

  // 4. Build C++ addon
  run('node build.mjs', 'Building C++ addon');

  // 5. Verify outputs
  const dllPath = join(RELEASE_DIR, 'broker_lib.dll');
  const addonPath = join(RELEASE_DIR, 'broker_addon.node');

  if (!existsSync(dllPath)) {
    console.error(`[postinstall] FAILED: broker_lib.dll not found at ${dllPath}`);
    process.exit(1);
  }
  if (!existsSync(addonPath)) {
    console.error(`[postinstall] FAILED: broker_addon.node not found at ${addonPath}`);
    process.exit(1);
  }

  console.log('[postinstall] Build complete — all artifacts verified.');
}

main();
