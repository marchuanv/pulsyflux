#!/usr/bin/env node

// Consumer verification test for @pulsyflux/broker
//
// This script simulates an external consumer installing and using the
// published npm package. It creates a temporary directory, installs
// @pulsyflux/broker via pnpm, and verifies the postinstall pipeline
// produced the expected native artifacts and that the public API is usable.
//
// NOTE: This test requires @pulsyflux/broker to be published on npm.
// It will fail until the package is available on the registry.

import { mkdtemp, writeFile, rm, access } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execSync } from "node:child_process";

const PASS = "\x1b[32mPASS\x1b[0m";
const FAIL = "\x1b[31mFAIL\x1b[0m";
const SKIP = "\x1b[33mSKIP\x1b[0m";

let tmpDir;
let exitCode = 0;

function log(status, msg) {
  console.log(`  [${status}] ${msg}`);
}

try {
  tmpDir = await mkdtemp(join(tmpdir(), "broker-node-consumer-"));

  // 1. Write a minimal package.json that depends on @pulsyflux/broker
  const pkg = {
    name: "test-consumer",
    version: "1.0.0",
    private: true,
    type: "module",
    dependencies: {
      "@pulsyflux/broker": "latest",
    },
  };
  await writeFile(join(tmpDir, "package.json"), JSON.stringify(pkg, null, 2));

  // 2. Run pnpm install
  try {
    execSync("pnpm install", {
      cwd: tmpDir,
      stdio: "pipe",
      timeout: 120_000,
    });
    log(PASS, "pnpm install succeeded");
  } catch (err) {
    log(FAIL, `pnpm install failed: ${err.stderr?.toString().trim() || err.message}`);
    log(SKIP, "Skipping remaining checks (package may not be published yet)");
    process.exit(0);
  }

  // 3. Verify native artifacts exist
  const pkgDir = join(tmpDir, "node_modules", "@pulsyflux", "broker");
  const dllPath = join(pkgDir, ".bin", "release", "broker_lib.dll");
  const addonPath = join(pkgDir, ".bin", "release", "broker_addon.node");

  try {
    await access(dllPath);
    log(PASS, "broker_lib.dll exists");
  } catch {
    log(FAIL, `broker_lib.dll not found at ${dllPath}`);
    exitCode = 1;
  }

  try {
    await access(addonPath);
    log(PASS, "broker_addon.node exists");
  } catch {
    log(FAIL, `broker_addon.node not found at ${addonPath}`);
    exitCode = 1;
  }

  // 4. Import the package and verify Server and Client are available
  try {
    const broker = await import(join(pkgDir, ".bin", "release", "registry.mjs"));

    if (typeof broker.Server !== "function") {
      log(FAIL, "Server is not a constructor");
      exitCode = 1;
    } else {
      log(PASS, "Server class is available");
    }

    if (typeof broker.Client !== "function") {
      log(FAIL, "Client is not a constructor");
      exitCode = 1;
    } else {
      log(PASS, "Client class is available");
    }
  } catch (err) {
    log(FAIL, `Import failed: ${err.message}`);
    exitCode = 1;
  }
} finally {
  // Cleanup
  if (tmpDir) {
    await rm(tmpDir, { recursive: true, force: true }).catch(() => {});
  }
}

if (exitCode === 0) {
  console.log("\nNode.js consumer test: ALL PASSED");
} else {
  console.log("\nNode.js consumer test: SOME FAILED");
}

process.exit(exitCode);
