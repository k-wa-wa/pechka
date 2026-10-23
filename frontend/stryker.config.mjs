// @ts-check
/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  packageManager: "npm",
  reporters: ["html", "clear-text", "progress"],
  testRunner: "vitest",
  testRunnerNodeArgs: ["--experimental-vm-modules"],
  coverageAnalysis: "perTest",
  mutate: [
    "components/**/*.{ts,tsx}",
    "lib/**/*.{ts,tsx}",
    "!components/**/*.stories.tsx",
    "!components/**/*.test.{ts,tsx}",
    "!lib/**/*.test.{ts,tsx}",
    "!lib/types.ts",
  ],
  thresholds: {
    high: 80,
    low: 60,
    break: 50,
  },
};
