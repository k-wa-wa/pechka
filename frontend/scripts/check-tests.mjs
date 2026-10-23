#!/usr/bin/env node
// Fails if a component under components/ has no matching *.test.tsx file,
// or if a non-type logic file in lib/ has no matching *.test.ts(x).
import { readdirSync } from "node:fs";
import { join } from "node:path";

const COMPONENTS_DIR = join(import.meta.dirname, "..", "components");
const LIB_DIR = join(import.meta.dirname, "..", "lib");

let failed = false;

// Check components/
const compFiles = readdirSync(COMPONENTS_DIR);
const components = compFiles
  .filter((f) => f.endsWith(".tsx") && !f.endsWith(".stories.tsx") && !f.endsWith(".test.tsx"))
  .map((f) => f.replace(/\.tsx$/, ""));
const compTests = new Set(
  compFiles.filter((f) => f.endsWith(".test.tsx")).map((f) => f.replace(/\.test\.tsx$/, "")),
);

const missingCompTests = components.filter((c) => !compTests.has(c));
if (missingCompTests.length > 0) {
  console.error("Missing unit test files for components:");
  for (const name of missingCompTests) {
    console.error(`  - components/${name}.tsx (expected components/${name}.test.tsx)`);
  }
  failed = true;
}

// Check lib/
const libFiles = readdirSync(LIB_DIR);
const libModules = libFiles
  .filter(
    (f) => f.endsWith(".ts") && !f.endsWith(".d.ts") && !f.endsWith(".test.ts") && f !== "types.ts",
  )
  .map((f) => f.replace(/\.ts$/, ""));
const libTests = new Set(
  libFiles.filter((f) => f.endsWith(".test.ts")).map((f) => f.replace(/\.test\.ts$/, "")),
);

const missingLibTests = libModules.filter((m) => !libTests.has(m));
if (missingLibTests.length > 0) {
  console.error("Missing unit test files for lib modules:");
  for (const name of missingLibTests) {
    console.error(`  - lib/${name}.ts (expected lib/${name}.test.ts)`);
  }
  failed = true;
}

if (failed) {
  process.exit(1);
}

console.log(`OK: all required frontend components and lib modules have matching tests.`);
