import { defineConfig } from '@playwright/test';

const frontendPort = process.env.FRONTEND_PORT ?? '4321';
const baseURL =
  process.env.PLAYWRIGHT_BASE_URL ?? `http://127.0.0.1:${frontendPort}`;
const shouldStartWebServer = process.env.PLAYWRIGHT_DISABLE_WEBSERVER !== '1';

export default defineConfig({
  testDir: './tests/smoke',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  webServer: shouldStartWebServer
    ? {
        command: `bun run preview -- --host 127.0.0.1 --port ${frontendPort}`,
        url: baseURL,
        reuseExistingServer: false,
        cwd: '.',
        timeout: 120_000,
      }
    : undefined,
});
