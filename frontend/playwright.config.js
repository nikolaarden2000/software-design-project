const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './e2e',
  timeout: 30000,
  workers: 1,

  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:8081',
    headless: process.env.HEADLESS !== 'false',
    viewport: {
      width: 1366,
      height: 900
    },
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure'
  },

  reporter: [
    ['list'],
    ['html']
  ]
});