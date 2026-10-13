import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/demo",
  fullyParallel: true,
  workers: 2,
  retries: 0,
  timeout: 45_000,
  use: {
    baseURL: "http://127.0.0.1:43179/Shortli/",
    locale: "en-US",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "npm run preview:demo -- --base /Shortli/",
    url: "http://127.0.0.1:43179/Shortli/",
    reuseExistingServer: false,
    timeout: 30_000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
