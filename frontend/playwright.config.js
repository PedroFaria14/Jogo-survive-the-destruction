import { defineConfig } from "@playwright/test";

// Requer backend local separado. Nunca usa endpoints de produção nesta suíte.
export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 45000,
  expect: { timeout: 15000 },
  reporter: "list",
  use: {
    baseURL: "http://localhost:4173",
    browserName: "chromium",
    locale: "pt-BR",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  webServer: {
    command:
      process.env.E2E_PREVIEW === "1"
        ? "pnpm run preview --host localhost --port 4173 --strictPort"
        : "pnpm run dev --host localhost --port 4173 --strictPort",
    url: "http://localhost:4173",
    reuseExistingServer: false,
    env: {
      VITE_WS_URL: "ws://127.0.0.1:18080/ws",
      VITE_API_URL: "http://127.0.0.1:18080",
    },
  },
});
