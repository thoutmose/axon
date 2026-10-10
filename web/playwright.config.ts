import { defineConfig, devices } from '@playwright/test';

// End-to-end tests against a running stack (docker compose up), e.g.
// docker run --network host -v $PWD:/work -w /work mcr.microsoft.com/playwright:v1.63.0-noble npx playwright test
export default defineConfig({
	testDir: 'test',
	fullyParallel: false,
	workers: 1, // one shared database: the tests are one story
	retries: 0,
	reporter: [['list']],
	use: {
		baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
		trace: 'retain-on-failure'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
});
