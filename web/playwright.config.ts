import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:18084";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  expect: { timeout: 8_000 },
  fullyParallel: false,
  reporter: [["list"]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium-desktop",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 920 } },
    },
    {
      name: "chromium-mobile",
      use: { ...devices["Pixel 7"], viewport: { width: 412, height: 915 } },
    },
  ],
  webServer: {
    command:
      "cd .. && ADDR=127.0.0.1:18084 ADMIN_TOKEN=admin-token ADMIN_TOKEN_ACCOUNTS='release-token:release.admin' WEB_DIST=web/dist go run -buildvcs=false ./cmd/server",
    url: `${baseURL}/readyz`,
    reuseExistingServer: false,
    timeout: 30_000,
  },
});
