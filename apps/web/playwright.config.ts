/**
 * The suite that watches the whole chain: a real browser, a real server, a real
 * database, real PNG bytes.
 *
 * Everything below it is already covered — Go tests the Go, the component tests
 * mount a widget in a fake page. What nobody checks otherwise is that the parts
 * still agree, and that is where every defect of the last three days was found.
 *
 * Headless only, per `CLAUDE.md`. Bring-up belongs to `just fe-test-e2e`, which
 * owns the database and the server; this file only points at them.
 */
import { defineConfig, devices } from '@playwright/test'
import { SIGNED_IN } from './e2e/session'

const WEB = process.env.OZALID_E2E_WEB ?? 'http://localhost:4174'

export default defineConfig({
  testDir: './e2e',
  // One worker: the suite seeds through the API and reads what it seeded. Two
  // workers would be two reviewers on one book.
  workers: 1,
  fullyParallel: false,
  // A failing end-to-end test is a signal, not weather. Retrying would turn a
  // real defect into an intermittent one.
  retries: 0,
  reporter: process.env.CI ? 'list' : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: WEB,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    // Nothing here is public any more, so the suite signs in first — through
    // the interface, once — and every test after it runs as that reviewer.
    { name: 'setup', testMatch: /.*\.setup\.ts/ },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], storageState: SIGNED_IN },
      dependencies: ['setup'],
    },
  ],
})
