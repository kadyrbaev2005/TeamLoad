import { defineConfig } from "@playwright/test";

// Runs against a running stack (`docker compose up --build`).
// PW_CHANNEL=chrome uses the installed Chrome instead of Playwright's Chromium.
export default defineConfig({
  testDir: "./e2e",
  timeout: 90_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  reporter: [["list"]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:8080",
    channel: process.env.PW_CHANNEL || undefined,
    viewport: { width: 1280, height: 900 },
    trace: "retain-on-failure",
  },
});
