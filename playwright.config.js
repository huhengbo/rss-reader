import { defineConfig, devices } from '@playwright/test';

const mobileTest = '**/mobile-device.spec.js';

export default defineConfig({
  testDir: './tests/browser',
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  timeout: 30000,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    locale: 'zh-CN',
    timezoneId: 'Asia/Shanghai'
  },
  projects: [
    { name: 'chromium', testIgnore: mobileTest, use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', testIgnore: mobileTest, use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', testIgnore: mobileTest, use: { ...devices['Desktop Safari'] } },
    { name: 'mobile-chromium', testMatch: mobileTest, use: { ...devices['Pixel 5'] } },
    { name: 'mobile-webkit', testMatch: mobileTest, use: { ...devices['iPhone 13'] } },
    // Native tab zoom requires a separate persistent Chromium context.
    { name: 'chromium-zoom', testDir: './tests/zoom', use: { browserName: 'chromium' } }
  ],
  webServer: {
    command: 'go run ./internal/server/testdata/ui',
    url: 'http://127.0.0.1:4173/healthz',
    reuseExistingServer: false,
    timeout: 120000
  }
});
