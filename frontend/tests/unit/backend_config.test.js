import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { resolveBackendUrls } from "../../src/network/backend_config.js";

test("production defaults use the public API domain for HTTP and WebSocket", () => {
  assert.deepEqual(resolveBackendUrls(), {
    apiBaseUrl: "https://api.survive.phfaria.com",
    webSocketUrl: "wss://api.survive.phfaria.com/ws",
  });
});

test("API overrides propagate to WebSocket when there is no explicit socket URL", () => {
  assert.deepEqual(
    resolveBackendUrls({ VITE_API_URL: "http://localhost:18080/" }),
    {
      apiBaseUrl: "http://localhost:18080",
      webSocketUrl: "ws://localhost:18080/ws",
    },
  );
  assert.deepEqual(
    resolveBackendUrls({ VITE_API_URL: "https://example.test/backend/" }),
    {
      apiBaseUrl: "https://example.test/backend",
      webSocketUrl: "wss://example.test/backend/ws",
    },
  );
});

test("explicit WebSocket URL remains prioritized and final slashes/whitespace are removed", () => {
  assert.deepEqual(
    resolveBackendUrls({
      VITE_API_URL: "  https://api.survive.phfaria.com///  ",
      VITE_WS_URL: "  wss://other.example.test/ws/  ",
    }),
    {
      apiBaseUrl: "https://api.survive.phfaria.com",
      webSocketUrl: "wss://other.example.test/ws",
    },
  );
});

test("empty variables still use the production defaults", () => {
  assert.deepEqual(
    resolveBackendUrls({ VITE_API_URL: "  ", VITE_WS_URL: " " }),
    resolveBackendUrls(),
  );
});

test("Vercel CSP allows both configured production backend origins", async () => {
  const config = JSON.parse(
    await readFile(new URL("../../vercel.json", import.meta.url), "utf8"),
  );
  const headers = config.headers.find(
    (rule) => rule.source === "/(.*)",
  ).headers;
  const csp = headers.find(
    (header) => header.key === "Content-Security-Policy",
  ).value;
  const sources = csp
    .split(";")
    .map((directive) => directive.trim())
    .find((directive) => directive.startsWith("connect-src "))
    .split(/\s+/);
  const { apiBaseUrl, webSocketUrl } = resolveBackendUrls();
  assert.ok(sources.includes(new URL(apiBaseUrl).origin));
  assert.ok(sources.includes(new URL(webSocketUrl).origin));
  assert.ok(!sources.some((source) => source.includes("onrender.com")));
});
