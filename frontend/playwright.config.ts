import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: "http://localhost:3000",
    viewport: { width: 1280, height: 800 },
  },
  projects: [
    {
      name: "chromium",
      use: { browserName: "chromium" },
    },
  ],
  webServer: [
    {
      command: "node e2e/mock-api.mjs",
      port: 9999,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: "npm run start",
      port: 3000,
      env: {
        API_URL: "http://localhost:9999",
      },
      reuseExistingServer: !process.env.CI,
    },
  ],
});
