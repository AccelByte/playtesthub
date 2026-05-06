import { defineConfig, devices } from '@playwright/test';

// PRD §6 Accessibility CI gate (M3 phase 15).
//
// We point Playwright at `vite preview` over the production bundle so the
// a11y assertions run against the same code shipped to GitHub Pages. The
// preview server boots in <1s on a built bundle; the test stubs every
// `/v1/...` and `/config.json` request via `page.route` so no backend is
// required at audit time.

export default defineConfig({
  testDir: 'tests/a11y',
  // a11y is a single-process audit; no parallel-flake risk and the spec is small.
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    // `npm run build` is run separately by CI / scripts so a bad build
    // surfaces as a build failure (not a Playwright timeout). Locally
    // a stale dist/ is the dev's problem; the smoke harness rebuilds.
    command: 'npx vite preview --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173/',
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
