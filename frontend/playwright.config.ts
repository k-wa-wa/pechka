import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:9002',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          executablePath: process.env.CHROMIUM_PATH,
        },
      },
    },
  ],
  webServer: [
    {
      command: 'MOCK_PORT=9001 node e2e/mock-server.mjs',
      port: 9001,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'NEXT_PUBLIC_API_URL=http://localhost:9001 npm run build && cp -r .next/static .next/standalone/.next/static && cp -r public .next/standalone/public && HOSTNAME=0.0.0.0 PORT=9002 node .next/standalone/server.js',
      port: 9002,
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
      env: {
        NEXT_PUBLIC_API_URL: 'http://localhost:9001',
        PORT: '9002',
        HOSTNAME: '0.0.0.0',
      },
    },
  ],
})
